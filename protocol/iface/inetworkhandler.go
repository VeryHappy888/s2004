package iface

// NetWorkHandler
type NetWorkHandler interface {
	OnHandShakeFailed(err error)
	OnRecvData(d []byte)
	OnConnect()
	OnError(err error)
	OnDisconnect()
}
