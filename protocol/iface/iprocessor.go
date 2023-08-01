package iface

import (
	"github.com/gogf/gf/container/gtype"
	_struct "ws-go/protocol/entity"
)

type INodeProcessor interface {
	ReceiveHandleMessage(messageStruct *_struct.ChatMessage)
}

type IBuildProcessor interface {
	SendBuilder(b NodeBuilder)
}

// 对外提供Api 接口
type INodeApi interface {
	SendNormalAndRead(id, to, participant string)
	SendReceiptRetry(to, id, participant, t string, count gtype.Int32)
	GetPreKeys(bool, ...string) error
}
