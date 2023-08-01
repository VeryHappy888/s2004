package node

import (
	"ws-go/protocol/define"
	"ws-go/protocol/newxxmp"
)

type IqNode_ struct {
	*BaseNode
	Id string
}

// createPingIqNode
func createPingIqNode(id string) *IqNode_ {
	//<iq id="2" xmlns="w:p" type="get" to="@s.whatsapp.net">
	//    <ping/>
	//</iq>
	// IqNode
	i := &IqNode_{Id: id, BaseNode: NewBaseNode()}
	// iq node
	iqNode := newxxmp.EmptyNode(define.NodeIqTag)
	iqNode.Attributes.AddAttr(define.AttrKeyXmlns, define.AttrValueWp)
	iqNode.Attributes.AddAttr(define.AttrKeyTo, define.AttrKeyWsServer)
	// ping node
	pingNode := newxxmp.EmptyNode(define.NodePingTag)
	// set ping to iq node children
	iqNode.Children.AddNode(pingNode)
	// set parent node
	i.SetParentNode(iqNode)
	return i
}

// createConfigIqNode
func createConfigIqNode(id string) *IqNode_ {
	// node
	//<iq id="1" xmlns="urn:xmpp:whatsapp:push" type="get" to="@s.whatsapp.net">
	//    <config/>
	//</iq>
	i := &IqNode_{Id: id, BaseNode: NewBaseNode()}
	// iq node
	IqNode := newxxmp.EmptyNode(define.NodePingTag)
	IqNode.Attributes.AddAttr("id", id)
	IqNode.Attributes.AddAttr(define.AttrKeyXmlns, "urn:xmpp:whatsapp:push")
	IqNode.Attributes.AddAttr("type", "get")
	IqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	// config node
	configNode := newxxmp.EmptyNode(define.NodeConfigTag)
	// set config node to iq node children
	IqNode.Children.AddNode(configNode)
	i.Node = IqNode
	return i
}
