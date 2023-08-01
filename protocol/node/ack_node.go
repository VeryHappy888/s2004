package node

import (
	"ws-go/protocol/newxxmp"
)

const NodeAck = "ack"
const ClassReceipt = "receipt"
const ClassNotification = "notification"
const ClassCall = "call"

type AckNode struct {
	*BaseNode
	id string
}

// createAck 发送确认
func createAck(id, to, xtype, class, participant string) *AckNode {
	if class == "" {
		class = "receipt"
	}
	//<ack id="0B00AEDE10AFA26D1F2440734C9A82C5" to="886955754531@s.whatsapp.net" class="receipt"/>
	a := &AckNode{BaseNode: NewBaseNode(), id: id}
	n := &newxxmp.Node{
		Tag: NodeAck,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("id", id),
			newxxmp.NewAttribute("to", to),
			newxxmp.NewAttribute("class", class),
		},
	}
	// participant
	if participant != "" {
		n.Attributes = append(
			n.Attributes,
			newxxmp.NewAttribute("participant", participant))
	}
	// type
	if xtype != "" {
		n.Attributes = append(
			n.Attributes,
			newxxmp.NewAttribute("type", xtype))
	}

	a.Node = n
	return a
}

type AckProcessor struct{}
