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
	ClientOnPublish   = "onPublish"
	ClientOnUnpublish = "onUnpublish"

	// ion to islb
	IslbGetPubs      = "getPubs"
	IslbGetMediaInfo = "getMediaInfo"
	IslbRelay        = "relay"
	IslbUnrelay      = "unRelay"
	IslbPublish      = "publish"
	IslbOnPublish    = "onPublish"
	IslbUnpublish    = "unpublish"
	IslbOnUnpublish  = "onUnpublish"
	IslbSubscribe    = "subscribe"
	IslbKeepAlive    = "keepAlive"

	IslbID = "islb"
)
