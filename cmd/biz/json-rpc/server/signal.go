package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/biz"
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
	SID string `json:"sid"`
	jwt.StandardClaims
}

// authenticateRoom checks both the connection token AND an optional message token for SID claims
// returns nil for success and returns an error if there are no valid claims for the SID
func authenticateRoom(connClaims *claims, keyFunc jwt.Keyfunc, msg proto.Authenticatable) error {
	log.Debugf("authenticateRoom: checking claims on token %v", msg.Token())
	// connection token has valid claim on this room, succeed early
	if connClaims != nil && msg.Room() == proto.SID(connClaims.SID) {
		log.Debugf("authenticateRoom: valid sid in connection claims %v", msg.Room())
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
	if msgClaims != nil && msg.Room() == proto.SID(msgClaims.SID) {
		log.Debugf("authenticateRoom: valid sid in msg claims %v", msg.Room())
		return nil
	}

	// if this is reached, a token was passed but it did not have a valid SID claim
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

// Signal represents Signal server
type Signal struct {
	conf   signalConf
	closed chan bool
	bs     *biz.Server
}

// newSignal create signal server instance
func newSignal(conf signalConf) *Signal {
	return &Signal{
		conf:   conf,
		closed: make(chan bool),
	}
}

// Start signal server
func (s *Signal) Start(bs *biz.Server) {
	s.bs = bs
	go s.serve()
}

// start signal server
func (s *Signal) serve() {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.Handle(s.conf.WebSocketPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get user id
		parms := r.URL.Query()
		fields := parms["uid"]
		if fields == nil || len(fields) == 0 {
			log.Errorf("invalid uid")
			http.Error(w, "invalid uid", http.StatusForbidden)
			return
		}
		uid := proto.UID(fields[0])
		log.Infof("peer connected, uid=%s", uid)

		// authenticate connection
		var connClaims *claims
		if s.conf.AuthConnection.Enabled {
			if token := r.URL.Query()["token"]; len(token) > 0 {
				// parsing and validating a token
				t, err := jwt.ParseWithClaims(token[0], &claims{}, s.conf.AuthConnection.KeyFunc)
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
		if s.conf.AuthRoom.Enabled {
			auth = func(msg proto.Authenticatable) error {
				return authenticateRoom(connClaims, s.conf.AuthRoom.KeyFunc, msg)
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
		p := newPeer(r.Context(), ws, uid, s.bs, auth)
		defer p.Close()
		log.Infof("new peer: %s, %s", r.RemoteAddr, p.UID())

		// wait the peer disconnecting
		select {
		case <-p.conn.DisconnectNotify():
			log.Infof("peer disconnected: %s, %s", r.RemoteAddr, p.UID())
			break
		case <-s.closed:
			log.Infof("server closed: disconnect peer, %s, %s", r.RemoteAddr, p.UID())
			break
		}
	}))

	// start web server
	var err error
	if s.conf.Cert == "" || s.conf.Key == "" {
		log.Infof("non-TLS WebSocketServer listening on: %s:%d", s.conf.Host, s.conf.Port)
		err = http.ListenAndServe(s.conf.Host+":"+strconv.Itoa(s.conf.Port), nil)
	} else {
		log.Infof("TLS WebSocketServer listening on: %s:%d", s.conf.Host, s.conf.Port)
		err = http.ListenAndServeTLS(s.conf.Host+":"+strconv.Itoa(s.conf.Port), s.conf.Cert, s.conf.Key, nil)
	}
	if err != nil {
		log.Errorf("http serve error: %v", err)
	}
}

// close signal server
func (s *Signal) close() {
	close(s.closed)
}
