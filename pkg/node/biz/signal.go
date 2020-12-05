package biz

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

// authConfig auth config
type authConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
	KeyType string `mapstructure:"key_type"`
}

// KeyFunc auth key types
func (a authConfig) KeyFunc(t *jwt.Token) (interface{}, error) {
	// nolint: gocritic
	switch a.KeyType {
	//TODO: add more support for keytypes here
	default:
		return []byte(a.Key), nil
	}
}

// claims custom claims type for jwt
type claims struct {
	UID string `json:"uid"`
	RID string `json:"rid"`
	*jwt.StandardClaims
}

// authenticateRoom checks both the connection token AND an optional message token for RID claims
// returns nil for success and returns an error if there are no valid claims for the RID
func authenticateRoom(connClaims *claims, keyFunc jwt.Keyfunc, msg proto.Authenticatable) error {
	log.Debugf("authenticateRoom: checking claims on token %v", msg.Token())
	// connection token has valid claim on this room, succeed early
	if connClaims != nil && msg.Room() == proto.RID(connClaims.RID) {
		log.Debugf("authenticateRoom: valid rid in connection claims %v", msg.Room())
		return nil
	}

	// check for a message level proto.RoomToken
	var msgClaims *claims = nil
	if t := msg.Token(); t != "" {
		token, err := jwt.ParseWithClaims(t, &claims{}, keyFunc)
		if err != nil {
			log.Debugf("authenticateRoom: error parsing token: %v", err)
			return errors.New("invalid room token")
		}
		log.Debugf("authenticateRoom: Got Token %#v", token)
		msgClaims = token.Claims.(*claims)
	}

	// no tokens were passed in
	if connClaims == nil && msgClaims == nil {
		return errors.New("authorization token required for access")
	}

	// message token is valid, succeed
	if msgClaims != nil && msg.Room() == proto.RID(msgClaims.RID) {
		log.Debugf("authenticateRoom: valid rid in msg claims %v", msg.Room())
		return nil
	}

	// if this is reached, a token was passed but it did not have a valid RID claim
	return errors.New("permission not sufficient for room")
}

// signalConf represents signal server configuration
type signalConf struct {
	Host           string     `mapstructure:"host"`
	Port           int        `mapstructure:"port"`
	Cert           string     `mapstructure:"cert"`
	Key            string     `mapstructure:"key"`
	WebSocketPath  string     `mapstructure:"path"`
	AuthConnection authConfig `mapstructure:"auth_connection"`
	AuthRoom       authConfig `mapstructure:"auth_room"`
}

// signal represents signal server
type signal struct {
	s      *server
	closed chan bool
}

// newSignal create signal server instance
func newSignal(s *server) *signal {
	return &signal{
		s:      s,
		closed: make(chan bool),
	}
}

// start signal server
func (s *signal) start(conf signalConf) {
	go s.serve(conf)
}

// start signal server
func (s *signal) serve(conf signalConf) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.Handle(conf.WebSocketPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// authenticate connection
		var connClaims *claims
		if conf.AuthConnection.Enabled {
			if token := r.URL.Query()["token"]; len(token) > 0 {
				// parsing and validating a token
				t, err := jwt.ParseWithClaims(token[0], &claims{}, conf.AuthConnection.KeyFunc)
				if err != nil {
					log.Errorf("invalid token: %v", err)
					http.Error(w, "invalid token", http.StatusForbidden)
					return
				}
				connClaims = t.Claims.(*claims)
			}
		}

		// authenticate message
		var auth func(msg proto.Authenticatable) error
		if conf.AuthRoom.Enabled {
			auth = func(msg proto.Authenticatable) error {
				return authenticateRoom(connClaims, conf.AuthRoom.KeyFunc, msg)
			}
		}

		// upgrade to websocket connection
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Errorf("upgrade websocket error: %v", err)
			http.Error(w, "upgrade websocket error", http.StatusForbidden)
			return
		}
		defer ws.Close()

		// create a peer
		p := newPeer(r.Context(), ws, s.s, auth)
		defer p.close()

		// wait the peer disconnecting
		select {
		case <-p.disconnectNotify():
			log.Infof("peer disconnected, uid=%s", p.uid)
			break
		case <-s.closed:
			log.Infof("server closed, disconnect peer, uid=%s", p.uid)
			break
		}
	}))

	// start web server
	var err error
	if conf.Cert == "" || conf.Key == "" {
		log.Infof("non-TLS WebSocketServer listening on: %s:%d", conf.Host, conf.Port)
		err = http.ListenAndServe(conf.Host+":"+strconv.Itoa(conf.Port), nil)
	} else {
		log.Infof("TLS WebSocketServer listening on: %s:%d", conf.Host, conf.Port)
		err = http.ListenAndServeTLS(conf.Host+":"+strconv.Itoa(conf.Port), conf.Cert, conf.Key, nil)
	}
	if err != nil {
		log.Errorf("http serve error: %v", err)
	}
}

// close signal server
func (s *signal) close() {
	close(s.closed)
}
