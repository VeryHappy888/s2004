package iface

import (
	"ws-go/noise"
)

// IHandshake
type IHandshake interface {
	RunHandshake(SegmentProcessor) error
	GetCipherStateGroup() (csIn *noise.CipherState, csOut *noise.CipherState)
	UpdateHandshakeSettings(i interface{})
}
