package proto

type PubInfo struct {
	MediaInfo
	Info ClientUserInfo `json:"info"`
}

type GetPubResp struct {
	RoomInfo
	Pubs []PubInfo
}

type GetMediaParams struct {
	RID RID
	MID MID
}

type FindServiceParams struct {
	Service string
	MID     MID
}

type GetSFURPCParams struct {
	ID          string
	Name        string
	Service     string
	GRPCAddress string
}
