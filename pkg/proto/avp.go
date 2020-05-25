package proto

import "encoding/json"

// ElementInfo describes an a/v process
type ElementInfo struct {
	MID    string          `json:"mid"`
	RID    string          `json:"rid"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}
