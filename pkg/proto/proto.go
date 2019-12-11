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
	ClientBroadcast   = "broadcast"

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

	IslbKeepAlive      = "keepAlive"
	IslbClientOnJoin   = ClientOnJoin
	IslbClientOnLeave  = ClientOnLeave
	IslbOnStreamAdd    = ClientOnStreamAdd
	IslbOnStreamRemove = ClientOnStreamRemove
	IslbOnBroadcast    = ClientBroadcast

	IslbID = "islb"
)
