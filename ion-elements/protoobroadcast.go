package elements

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/sssgun/ion/pkg/log"
	"github.com/sssgun/ion/pkg/process"
	"github.com/sssgun/ion/pkg/process/samples"
	"github.com/sssgun/ion/pkg/proto"
	"github.com/sssgun/ion/pkg/util"
)

const (
	// TypeProtooBroadcast .
	TypeProtooBroadcast = "ProtooBroadcast"
)

// ProtooBroadcastConfig .
type ProtooBroadcastConfig struct {
	ID        string
	RID       string
	UID       string
	Requestor *nprotoo.Requestor
}

// ProtooBroadcast instance
type ProtooBroadcast struct {
	id        string
	rid       string
	uid       string
	requestor *nprotoo.Requestor
}

// NewProtooBroadcast instance
func NewProtooBroadcast(config ProtooBroadcastConfig) *ProtooBroadcast {
	b := &ProtooBroadcast{
		id:        config.ID,
		rid:       config.RID,
		uid:       config.UID,
		requestor: config.Requestor,
	}

	log.Infof("New ProtooBroadcast with config: %+v", config)

	return b
}

// Type for Protoobroadcast
func (b *ProtooBroadcast) Type() string {
	return TypeProtooBroadcast
}

func (b *ProtooBroadcast) Write(sample *samples.Sample) error {
	sample.Properties["uid"] = b.uid
	b.requestor.AsyncRequest(proto.ClientBroadcast, util.Map("rid", b.rid, "info", sample.Properties))
	return nil
}

func (b *ProtooBroadcast) Read() <-chan *samples.Sample {
	return nil
}

// Attach attach a child element
func (b *ProtooBroadcast) Attach(e process.Element) error {
	return ErrAttachNotSupported
}

// Close Protoobroadcast
func (b *ProtooBroadcast) Close() {
	log.Infof("Protoobroadcast.Close() %s", b.id)
}
