package iface

import "ws-go/noise"

// ICiphersStateGroup
type ICiphersStateGroup interface {
	SetCiphersStateGroup(csIn, csOut *noise.CipherState)
}
