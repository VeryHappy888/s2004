// Package serialize provides a serialization structure to serialize and
// deserialize Signal objects into storeable and transportable bytes.
package serialize

import (
	groupRecord "ws-go/libsignal/groups/state/record"
	"ws-go/libsignal/protocol"
	"ws-go/libsignal/state/record"
)

// NewSerializer will return a new serializer object that will be used
// to encode/decode Signal objects into bytes.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serializer is a structure to serialize Signal objects
// into bytes. This allows you to use any serialization format
// to store or send Signal objects.
type Serializer struct {
	SenderKeyRecord              groupRecord.SenderKeySerializer
	SenderKeyState               groupRecord.SenderKeyStateSerializer
	SignalMessage                protocol.SignalMessageSerializer
	PreKeySignalMessage          protocol.PreKeySignalMessageSerializer
	SenderKeyMessage             protocol.SenderKeyMessageSerializer
	SenderKeyDistributionMessage protocol.SenderKeyDistributionMessageSerializer
	SignedPreKeyRecord           record.SignedPreKeySerializer
	PreKeyRecord                 record.PreKeySerializer
	State                        record.StateSerializer
	Session                      record.SessionSerializer
}
