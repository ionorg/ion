package signal

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	conf "github.com/pion/ion/pkg/conf/biz"
	"github.com/pion/ion/pkg/log"
)

// WebSocketServerConfig represents websocket server configuration
type WebSocketServerConfig struct {
	Host          string
	Port          int
	CertFile      string
	KeyFile       string
	WebSocketPath string

	AuthConnection conf.AuthConfig
}

// MsgHandler type
type MsgHandler func(ws *transport.WebSocketTransport, request *http.Request)

type contextKey struct {
	name string
}

var claimsCtxKey = &contextKey{"claims"}
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func getClaims(connectionAuth conf.AuthConfig, r *http.Request) (*Claims, error) {
	vars := r.URL.Query()

	log.Debugf("Authenticating token")
	tokenParam := vars["access_token"]
	if tokenParam == nil || len(tokenParam) < 1 {
		return nil, errors.New("no token")
	}

	tokenStr := tokenParam[0]

	log.Debugf("checking claims on token %v", tokenStr)
	// Passing nil for keyFunc, since token is expected to be already verified (by a proxy)
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, connectionAuth.KeyFunc)
	if err != nil {
		return nil, err
	}
	return token.Claims.(*Claims), nil
}

func handler(connectionAuth conf.AuthConfig, msgHandler MsgHandler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if connectionAuth.Enabled {
			// extract connection level claims from auth token
			claims, err := getClaims(connectionAuth, r)
			if err != nil {
				log.Errorf("Error authenticating user => %s", err)
				http.Error(w, "Invalid token", http.StatusForbidden)
				return
			}

			log.Debugf("authenticated user with claims %#v", claims)

			// put it in context
			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			// and call the next with our new context
			r = r.WithContext(ctx)
		}

		responseHeader := http.Header{}
		responseHeader.Add("Sec-WebSocket-Protocol", "protoo")
		socket, err := upgrader.Upgrade(w, r, responseHeader)
		if err != nil {
			log.Errorf("Error upgrading => %s", err)
			http.Error(w, "Error upgrading socket", http.StatusBadRequest)
			return
		}
		logger.Debugf("Creating new WebSocket")
		wsTransport := transport.NewWebSocketTransport(socket)
		wsTransport.Start()

		msgHandler(wsTransport, r)
	}
}

// NewWebSocketServer for signaling
func NewWebSocketServer(cfg WebSocketServerConfig, msgHandler MsgHandler) error {
	// Websocket handle func
	http.HandleFunc(cfg.WebSocketPath, handler(cfg.AuthConnection, msgHandler))

	if cfg.CertFile == "" || cfg.KeyFile == "" {
		logger.Infof("non-TLS WebSocketServer listening on: %s:%d", cfg.Host, cfg.Port)
		return http.ListenAndServe(cfg.Host+":"+strconv.Itoa(cfg.Port), nil)
	}

	logger.Infof("TLS WebSocketServer listening on: %s:%d", cfg.Host, cfg.Port)
	return http.ListenAndServeTLS(cfg.Host+":"+strconv.Itoa(cfg.Port), cfg.CertFile, cfg.KeyFile, nil)
}

// ForContext finds the request claims from the context.
func ForContext(ctx context.Context) *Claims {
	raw, _ := ctx.Value(claimsCtxKey).(*Claims)
	return raw
}
