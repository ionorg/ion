package proto

type PubInfo struct {
	MediaInfo
	Info   ClientUserInfo `json:"info"`
	Tracks TrackMap       `json:"tracks"`
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
