package iface

import "net"

type INetWork interface {
	GetConn() net.Conn
	SetConn(c net.Conn)
}
