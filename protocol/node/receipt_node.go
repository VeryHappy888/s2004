package node

import (
	"github.com/gogf/gf/container/gtype"
	"ws-go/protocol/newxxmp"
)

type ReceiptNode struct {
	*BaseNode
	id string
}

func createReceiptCall(to, id string, offer *newxxmp.Node) *ReceiptNode {
	r := &ReceiptNode{BaseNode: NewBaseNode(), id: id}
	receiptNode := newxxmp.EmptyNode(NodeReceipt)
	receiptNode.Attributes.AddAttr("to", to)
	receiptNode.Attributes.AddAttr("id", id)
	receiptNode.Children.AddNode(offer)
	r.Node = receiptNode
	return r
}
func createAckCall(to, id string, node *newxxmp.Node) *ReceiptNode {
	//<ack class="call" id="5313D7EF1195683BC91E5BF37C1B3C07" to="971509514271@s.whatsapp.net"/>
	r := &ReceiptNode{BaseNode: NewBaseNode(), id: id}
	ack := newxxmp.EmptyNode("ack")
	ack.Attributes.AddAttr("id", id)
	ack.Attributes.AddAttr("class", "call")
	ack.Attributes.AddAttr("to", to)
	if node.GetChildrenByTag("terminate") != nil {
		ack.Attributes.AddAttr("type", "terminate")
	}
	if node.GetChildrenByTag("relaylatency") != nil {
		ack.Attributes.AddAttr("type", "relaylatency")
	}
	r.Node = ack
	return r
}

// createReceiptRetry 解密失败时发送重试
func createReceiptRetry(to, id, participant, t string, count gtype.Int32, more ...*newxxmp.Node) *ReceiptNode {
	r := &ReceiptNode{BaseNode: NewBaseNode(), id: id}
	receiptNode := newxxmp.EmptyNode(NodeReceipt)
	receiptNode.Attributes.AddAttr("to", to)
	receiptNode.Attributes.AddAttr("id", id)
	receiptNode.Attributes.AddAttr("type", "retry")
	// participant
	if participant != "" {
		receiptNode.Attributes.AddAttr("participant", participant)
	}

	retryNode := newxxmp.EmptyNode("retry")
	retryNode.Attributes.AddAttr("v", "1")
	retryNode.Attributes.AddAttr("count", count.String())
	retryNode.Attributes.AddAttr("id", id)
	retryNode.Attributes.AddAttr("t", t)
	// set receiptNode to retryNode
	receiptNode.Children.AddNode(retryNode)
	// more children
	for _, i := range more {
		if i != nil {
			receiptNode.Children.AddNode(i)
		}
	}

	r.Node = receiptNode
	return r
}

//createNormalReceipt 收到消息时发送
func createNormalReceipt(id, to, participant string, read bool) *ReceiptNode {
	// <receipt to="8617607567005@s.whatsapp.net" id="3A28D514E2B5E74FDDCC"/>
	// 已读消息
	// <receipt to="8617607567005@s.whatsapp.net" id="3AFDFD94CACB270AE65D" type="read"/>
	a := &ReceiptNode{BaseNode: NewBaseNode(), id: id}
	n := &newxxmp.Node{
		Tag: NodeReceipt,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("id", id),
			newxxmp.NewAttribute("to", to),
		},
	}
	// participant
	if participant != "" {
		n.Attributes = append(n.Attributes, newxxmp.NewAttribute("participant", participant))
	}
	// read
	if read {
		n.Attributes = append(n.Attributes, newxxmp.NewAttribute("type", "read"))
	}
	a.Node = n
	return a
}
