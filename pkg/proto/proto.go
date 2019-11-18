package proto

const (
	// client to ion
	ClientLogin       = "login"
	ClientJoin        = "join"
	ClientLeave       = "leave"
	ClientPublish     = "publish"
	ClientUnPublish   = "unpublish"
	ClientSubscribe   = "subscribe"
	ClientUnSubscribe = "unsubscribe"
	ClientClose       = "close"

	// ion to client
	ClientOnJoin         = "peer-join"
	ClientOnLeave        = "peer-leave"
	ClientOnStreamAdd    = "stream-add"
	ClientOnStreamRemove = "stream-remove"

	// ion to islb
	IslbGetPubs      = "getPubs"
	IslbGetMediaInfo = "getMediaInfo"
	IslbRelay        = "relay"
	IslbUnrelay      = "unRelay"

	IslbSubscribe      = "subscribe"
	IslbKeepAlive      = "keepAlive"
	IslbClientOnJoin   = "peer-join"
	IslbClientOnLeave  = "peer-leave"
	IslbOnStreamAdd    = "stream-add"
	IslbOnStreamRemove = "stream-remove"

	IslbID = "islb"
)
