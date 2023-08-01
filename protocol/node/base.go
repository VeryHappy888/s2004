package node

import (
	"log"
	"ws-go/protocol/impl"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/utils/promise"
)

// BaseNode Default node impl
type BaseNode struct {
	*newxxmp.Node
	*impl.ResultPromise
}

// NewBaseNode
func NewBaseNode() *BaseNode {
	return &BaseNode{
		Node:          nil,
		ResultPromise: impl.NewResultPromise(),
	}
}

// SetParentNode
func (b *BaseNode) SetParentNode(node *newxxmp.Node) {
	b.Node = node
}

// SuccessNotice 成功通知回调
func (b *BaseNode) SuccessNotice(any promise.Any) {
	b.Promise.SuccessResolve(any)
}

// Builder build node to ixxmp data
func (b *BaseNode) Builder() ([]byte, error) {
	xxmpData := b.GetTokenArray().GetBytes(true)
	log.Println("PresenceNode builder node ", b.Node.GetString())
	return xxmpData, nil
}
