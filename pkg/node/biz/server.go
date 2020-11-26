package biz

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

const (
	statCycle = time.Second * 3
)

// authConfig auth config
type authConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
	KeyType string `mapstructure:"key_type"`
}

// KeyFunc auth key types
func (a authConfig) KeyFunc(t *jwt.Token) (interface{}, error) {
	switch a.KeyType {
	//TODO: add more support for keytypes here
	default:
		return []byte(a.Key), nil
	}
}

// claims supported in JWT
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

// SignalConfig represents signaling server configuration
type signalConf struct {
	Host           string     `mapstructure:"host"`
	Port           int        `mapstructure:"port"`
	Cert           string     `mapstructure:"cert"`
	Key            string     `mapstructure:"key"`
	WebSocketPath  string     `mapstructure:"path"`
	AuthConnection authConfig `mapstructure:"auth_connection"`
	AuthRoom       authConfig `mapstructure:"auth_room"`
}

// server represents signal server
type server struct {
	closed chan bool
}

// newServer create signal server instance
func newServer(conf signalConf) *server {
	s := &server{
		closed: make(chan bool),
	}
	go s.stat()
	go s.start(conf)
	return s
}

// start signal server
func (s *server) start(conf signalConf) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.Handle(conf.WebSocketPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get user id
		parms := r.URL.Query()
		id := parms["id"]
		if id == nil || len(id) < 1 {
			log.Errorf("invalid id")
			http.Error(w, "invalid id", http.StatusForbidden)
			return
		}
		uid := proto.UID(id[0])
		log.Infof("peer connected, uid=%s", uid)

		// authenticate connection
		var cc *claims
		if conf.AuthConnection.Enabled {
			if token := parms["token"]; token != nil && len(token) > 0 {
				// passing nil for keyFunc, since token is expected to be already verified (by a proxy)
				t, err := jwt.ParseWithClaims(token[0], &claims{}, conf.AuthConnection.KeyFunc)
				if err != nil {
					log.Errorf("invalid token: %v", err)
					http.Error(w, "invalid token", http.StatusForbidden)
					return
				}
				cc = t.Claims.(*claims)
			}
		}

		// authenticate message
		var auth func(msg proto.Authenticatable) error
		if conf.AuthRoom.Enabled {
			auth = func(msg proto.Authenticatable) error {
				return authenticateRoom(cc, conf.AuthRoom.KeyFunc, msg)
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
		p := newPeer(r.Context(), ws, uid, auth)
		defer p.close()

		// wait the peer disconnecting
		select {
		case <-p.disconnectNotify():
			log.Infof("peer disconnected, uid=%s", p.id)
			break
		case <-s.closed:
			log.Infof("server closed, disconnect peer, uid=%s", p.id)
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
func (s *server) close() {
	close(s.closed)
}

// stat peers
func (s *server) stat() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			break
		case <-s.closed:
			log.Infof("stop stat")
			return
		}

		var info string
		roomLock.RLock()
		for rid, room := range rooms {
			info += fmt.Sprintf("room: %s\npeers: %d\n", rid, len(room.getPeers()))
			if len(room.getPeers()) == 0 {
				delete(rooms, rid)
			}
		}
		roomLock.RUnlock()
		if len(info) > 0 {
			log.Infof("\n----------------signal-----------------\n" + info)
		}
	}
}
