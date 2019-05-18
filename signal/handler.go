package signal

import (
	"time"

	"github.com/centrifugal/centrifuge-go"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pion/sfu/conf"
)

type Handler struct {
}

func (h *Handler) OnSubscribeSuccess(sub *centrifuge.Subscription, e centrifuge.SubscribeSuccessEvent) {
}

func (h *Handler) OnSubscribeError(sub *centrifuge.Subscription, e centrifuge.SubscribeErrorEvent) {
}

func (h *Handler) OnUnsubscribe(sub *centrifuge.Subscription, e centrifuge.UnsubscribeEvent) {
}

func (h *Handler) OnPublish(sub *centrifuge.Subscription, e centrifuge.PublishEvent) {
}

func (h *Handler) createSubscribeToken(channel, client string) string {
	claims := jwt.MapClaims{"channel": channel, "client": client}
	if conf.Cfg.Centrifugo.Expire > 0 {
		claims["exp"] = time.Now().Unix() + conf.Cfg.Centrifugo.Expire
	}
	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(conf.Cfg.Centrifugo.Key))
	if err != nil {
		panic(err)
	}
	return t
}

func (h *Handler) OnPrivateSub(cc *centrifuge.Client, e centrifuge.PrivateSubEvent) (string, error) {
	token := h.createSubscribeToken(e.Channel, e.ClientID)
	return token, nil
}

func (h *Handler) OnConnect(*centrifuge.Client, centrifuge.ConnectEvent) {
}

func (h *Handler) OnError(*centrifuge.Client, centrifuge.ErrorEvent) {
}

func (h *Handler) OnDisconnect(*centrifuge.Client, centrifuge.DisconnectEvent) {
}
