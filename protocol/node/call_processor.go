package node

import (
	"ws-go/protocol/iface"
	"ws-go/protocol/newxxmp"
)

const NodeCall = "call"

// CallProcessor
type CallProcessor struct {
	iface.IBuildProcessor
}

func NewCallProcessor(b iface.IBuildProcessor) *CallProcessor {
	return &CallProcessor{IBuildProcessor: b}
}

// Handle
func (c *CallProcessor) Handle(node *newxxmp.Node) {
	if node.GetTag() != NodeCall {
		return
	}

	childNode := node.GetChildrenIndex(0)
	switch childNode.GetTag() {
	case "offer":
		c.handleOffer(node)
	case "terminate":
		c.handleTerminate(node)
	}

}

// handleTerminate
func (c *CallProcessor) handleTerminate(node *newxxmp.Node) {
	// to
	to := node.GetAttributeByValue("from")
	// id
	id := node.GetAttributeByValue("id")
	c.SendBuilder(createAck(id, to, "terminate", ClassCall, ""))
}

// handleOffer
func (c *CallProcessor) handleOffer(node *newxxmp.Node) {
	offerNode := node.GetChildrenByTag("offer")
	if offerNode == nil {
		return
	}
	offerNode.Children = nil
	// to
	to := node.GetAttributeByValue("from")
	// id
	id := node.GetAttributeByValue("id")

	//v := createReceiptCall(to, id, offerNode)
	//
	v := createAckCall(to, id, node)
	//<ack class="call" id="0C0029B407BBA84863B008D0E9B8069C" to="971509514271@s.whatsapp.net" type="terminate"/>

	//fmt.Println("handleOffer", v.GetString())
	c.SendBuilder(v)
}
