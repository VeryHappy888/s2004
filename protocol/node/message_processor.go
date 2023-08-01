package node

import (
	"fmt"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/guuid"
	"strconv"
	"strings"
	"time"
	"ws-go/libsignal/protocol"
	entity "ws-go/protocol/entity"
	_interface "ws-go/protocol/iface"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/utils/promise"
)

const NodeMessage = "message"
const NodeReceipt = "receipt"

type MessageNode struct {
	*BaseNode
	id string
}

// GetMsgId
func (m *MessageNode) GetMsgId() string {
	return m.id
}

//func (m *MessageNode) Builder() ([]byte,error) {
//	return hex.DecodeString("00f8081607faff878617607567005f03051f04fb10e26128b3ca40172ee1020f8a75d310dcf801f80612130d052cfc42330a210539b9b724937542e8e810037c88374d12ecce41190d5787d3e41b4237fa36ac2c1013180022108efefe798e75b2cb8eabcc07bb6de50cc40a64cae881a43f ")
//}

// createParticipants
func createParticipants(cs map[string]protocol.CiphertextMessage) *newxxmp.Node {
	if cs == nil {
		return nil
	}
	/*
	 <participants>
	        <to S="84878185730@s.whatsapp.net">
	            <enc v="2" type="msg">fcb3330a21059a5f4bf5fe65636bb9db7374f60de1b11fcaff0b37286ff9e987768fc4328d6710001800228001b7b0fb33c3cf63254d8627ce3de1fc1067e482831e7eae7712ced3c18cd093da61a997e28db1c5986f8d6edf2c9f1e1f28b6db580f424460b7a24741369b2eabe1300d92b3741d2b5289750adc3f178fce323f15faea30760ef044b566cdb9e77dbb08c163417fec2cf967d7068f697b9eb3a9143ab5668fbf8074aae119ab6f970627ddfa34eebb</enc>
	        </to>
	        <to S="886955754531@s.whatsapp.net">
	            <enc v="2" type="pkmsg">fd00010a3308c7da8305122105edbc1ded0bd98c7ed85db25108dfb3171ceb8f81f409147add01bdec61903e4f1a21052367ac4923657187e382906d0b203449b650e264f2a08002aa9f514492adae5922b301330a2105eb151457ca23b140d3aa888fe51a39a2566ca55a1641cdc6c4a51024205f7c73100018002280014c94e2f027f33157520f2c0e91258f490f4c5a1e4fc439fd75b658bcabedb8c25ec06b19da2f4d36bc74ad50ec89b42ed3c58e991337e2109bd17718732e53bfcf5ee080b0302b9c84dd59700efa5e75cdb871e58e0c4c48e80c81d346983ac02eb4ba6f45c69f2e62cbcbd31ecfb5f0d48adcf811594735d35c677c577c38cb032fcb98c7d4617d289fb1b294063012</enc>
	        </to>
	    </participants>
	*/
	toNodes := &newxxmp.Nodes{}
	for jid, ciphertextMessage := range cs {
		encType := protocol.GetEncTypeString(ciphertextMessage.Type())
		if !strings.Contains(jid, "@s.whatsapp.net") {
			jid += "@s.whatsapp.net"
		}
		// toNode
		toNode := newxxmp.EmptyNode("to")
		toNode.Attributes.AddAttr("jid", jid)
		// encNode
		encNode := newxxmp.EmptyNode("enc")
		encNode.Attributes.AddAttr("v", "2")
		encNode.Attributes.AddAttr("type", encType)
		encNode.SetData(ciphertextMessage.Serialize())
		// add enc node
		toNode.Children.AddNode(encNode)
		// add to node
		toNodes.AddNode(toNode)
	}
	// participants
	return newxxmp.EmptyNode("participants", toNodes)
}

func createSnsParticipants(cs map[string]protocol.CiphertextMessage) *newxxmp.Node {
	if cs == nil {
		return nil
	}
	toNodes := &newxxmp.Nodes{}

	for jid, ciphertextMessage := range cs {
		encType := protocol.GetEncTypeString(ciphertextMessage.Type())
		if !strings.Contains(jid, "@s.whatsapp.net") {
			jid += "@s.whatsapp.net"
		}
		//jid = strings.ReplaceAll(jid, "@s.whatsapp.net", ".0:0@s.whatsapp.net")
		// toNode
		toNode := newxxmp.EmptyNode("to")
		toNode.Attributes.AddAttr("jid", jid)
		// encNode
		encNode := newxxmp.EmptyNode("enc")
		encNode.Attributes.AddAttr("v", "2")
		encNode.Attributes.AddAttr("type", encType)
		encNode.SetData(ciphertextMessage.Serialize())
		// add enc node
		toNode.Children.AddNode(encNode)
		// add to node
		toNodes.AddNode(toNode)
	}
	// participants
	return newxxmp.EmptyNode("participants", toNodes)
}

// createMessageNode
func createMessageNode(to string, veriFiledName uint64, msgType string, c protocol.CiphertextMessage, participants *newxxmp.Node, phash ...string) *MessageNode {
	var encType string
	//return &MessageNode{BaseNode:NewBaseNode()}
	encType = protocol.GetEncTypeString(c.Type())
	// 暂时随机
	id := strings.ToUpper(strings.ReplaceAll(guuid.New().String(), "-", ""))
	// default promise 超时100秒
	p := &MessageNode{id: id, BaseNode: NewBaseNode()}
	// message node
	messageNode := newxxmp.EmptyNode(NodeMessage)
	messageNode.Attributes.AddAttr("to", to)
	messageNode.Attributes.AddAttr("type", msgType)
	messageNode.Attributes.AddAttr("id", id)
	if veriFiledName != 0 {
		//商业版本
		messageNode.Attributes.AddAttr("verified_name", strconv.FormatUint(veriFiledName, 10))
	}
	// enc node
	encNode := newxxmp.EmptyNode("enc")
	encNode.Attributes.AddAttr("v", "2")
	encNode.Attributes.AddAttr("type", encType)
	encNode.SetData(c.Serialize())
	// set enc node
	messageNode.Children.AddNode(encNode)
	// message node attr phash
	if len(phash) > 0 && phash[0] != "" {
		messageNode.Attributes.AddAttr("phash", phash[0])
	}
	// message node participants
	if participants != nil {
		messageNode.Children.AddNode(participants)
	}

	p.Node = messageNode
	return p
}

// createIqSnsText 发送动态文本
func CreateIqSnsText(id gtype.Int32, veriFiledName uint64, jidVal string, c protocol.CiphertextMessage, participants *newxxmp.Node, pHash string) *IqNode {
	//<message to='status@broadcast' type='text' id='' phash='2:D8xEQPQ7'><enc v='2' type='skmsg'>内容</enc>
	//<participants><to jid='13166613347.0:0@s.whatsapp.net'/><to jid='8613538240891.0:0@s.whatsapp.net'/></participants>
	//</message>
	// 暂时随机
	// default promise 超时100秒
	//return &MessageNode{BaseNode:NewBaseNode()}
	var encType string
	encType = protocol.GetEncTypeString(protocol.SENDERKEY_TYPE)
	// message node
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeMessage)
	iqNode.Attributes.AddAttr("to", "status@broadcast")
	iqNode.Attributes.AddAttr("type", "text")
	iqNode.Attributes.AddAttr("id", strings.ToUpper(strings.ReplaceAll(guuid.New().String(), "-", "")))
	if veriFiledName != 0 {
		//商业版本
		iqNode.Attributes.AddAttr("verified_name", strconv.FormatUint(veriFiledName, 10))
	}
	iqNode.Attributes.AddAttr("phash", pHash)
	encNode := newxxmp.EmptyNode("enc")
	encNode.Attributes.AddAttr("v", "2")
	encNode.Attributes.AddAttr("type", encType)
	encNode.SetData(c.Serialize())
	iqNode.Children.AddNode(encNode)
	if participants != nil {
		iqNode.Children.AddNode(participants)
	}
	i.Node = iqNode
	return i
}

func createImageMessageNode(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, participants *newxxmp.Node, phash ...string) *MessageNode {
	var encType string
	//return &MessageNode{BaseNode:NewBaseNode()}
	encType = protocol.GetEncTypeString(c.Type())
	// 暂时随机
	id := strings.ToUpper(strings.ReplaceAll(guuid.New().String(), "-", ""))
	// default promise 超时100秒
	p := &MessageNode{id: id, BaseNode: NewBaseNode()}
	// message node
	messageNode := newxxmp.EmptyNode(NodeMessage)
	messageNode.Attributes.AddAttr("to", to)
	messageNode.Attributes.AddAttr("type", msgType)
	messageNode.Attributes.AddAttr("id", id)
	if veriFiledName != 0 {
		//商业版本
		messageNode.Attributes.AddAttr("verified_name", strconv.FormatUint(veriFiledName, 10))
	}
	// enc node
	encNode := newxxmp.EmptyNode("enc")
	encNode.Attributes.AddAttr("v", "2")
	encNode.Attributes.AddAttr("type", encType)
	encNode.Attributes.AddAttr("mediatype", "image") //image
	encNode.SetData(c.Serialize())
	// set enc node
	messageNode.Children.AddNode(encNode)
	// message node attr phash
	if len(phash) > 0 && phash[0] != "" {
		messageNode.Attributes.AddAttr("phash", phash[0])
	}
	// message node participants
	if participants != nil {
		messageNode.Children.AddNode(participants)
	}

	p.Node = messageNode
	return p
}

func createAudioMessageNode(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, participants *newxxmp.Node, phash ...string) *MessageNode {
	var encType string
	//return &MessageNode{BaseNode:NewBaseNode()}
	encType = protocol.GetEncTypeString(c.Type())
	// 暂时随机
	id := strings.ToUpper(strings.ReplaceAll(guuid.New().String(), "-", ""))
	// default promise 超时100秒
	p := &MessageNode{id: id, BaseNode: NewBaseNode()}
	// message node
	messageNode := newxxmp.EmptyNode(NodeMessage)
	messageNode.Attributes.AddAttr("to", to)
	messageNode.Attributes.AddAttr("type", msgType)
	messageNode.Attributes.AddAttr("id", id)
	if veriFiledName != 0 {
		//商业版本
		messageNode.Attributes.AddAttr("verified_name", strconv.FormatUint(veriFiledName, 10))
	}
	// enc node
	encNode := newxxmp.EmptyNode("enc")
	encNode.Attributes.AddAttr("v", "2")
	encNode.Attributes.AddAttr("type", encType)
	encNode.Attributes.AddAttr("mediatype", "ptt") //ptt
	encNode.SetData(c.Serialize())
	// set enc node
	messageNode.Children.AddNode(encNode)
	// message node attr phash
	if len(phash) > 0 && phash[0] != "" {
		messageNode.Attributes.AddAttr("phash", phash[0])
	}
	// message node participants
	if participants != nil {
		messageNode.Children.AddNode(participants)
	}

	p.Node = messageNode
	return p
}

//createVideoMessageNode
func createVideoMessageNode(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, participants *newxxmp.Node, phash ...string) *MessageNode {
	var encType string
	//return &MessageNode{BaseNode:NewBaseNode()}
	encType = protocol.GetEncTypeString(c.Type())
	// 暂时随机
	id := strings.ToUpper(strings.ReplaceAll(guuid.New().String(), "-", ""))
	// default promise 超时100秒
	p := &MessageNode{id: id, BaseNode: NewBaseNode()}
	// message node
	messageNode := newxxmp.EmptyNode(NodeMessage)
	messageNode.Attributes.AddAttr("to", to)
	messageNode.Attributes.AddAttr("type", msgType)
	messageNode.Attributes.AddAttr("id", id)
	if veriFiledName != 0 {
		//商业版本
		messageNode.Attributes.AddAttr("verified_name", strconv.FormatUint(veriFiledName, 10))
	}
	// enc node
	encNode := newxxmp.EmptyNode("enc")
	encNode.Attributes.AddAttr("v", "2")
	encNode.Attributes.AddAttr("type", encType)
	encNode.Attributes.AddAttr("mediatype", "video") //video
	encNode.SetData(c.Serialize())
	// set enc node
	messageNode.Children.AddNode(encNode)
	// message node attr phash
	if len(phash) > 0 && phash[0] != "" {
		messageNode.Attributes.AddAttr("phash", phash[0])
	}
	// message node participants
	if participants != nil {
		messageNode.Children.AddNode(participants)
	}

	p.Node = messageNode
	return p
}

// createVcardMessageNode
func createVcardMessageNode(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, participants *newxxmp.Node, phash ...string) *MessageNode {
	var encType string
	//return &MessageNode{BaseNode:NewBaseNode()}
	encType = protocol.GetEncTypeString(c.Type())
	// 暂时随机
	id := strings.ToUpper(strings.ReplaceAll(guuid.New().String(), "-", ""))
	// default promise 超时100秒
	p := &MessageNode{id: id, BaseNode: NewBaseNode()}
	// message node
	messageNode := newxxmp.EmptyNode(NodeMessage)
	messageNode.Attributes.AddAttr("to", to)
	messageNode.Attributes.AddAttr("type", msgType)
	messageNode.Attributes.AddAttr("id", id)
	if veriFiledName != 0 {
		//商业版本
		messageNode.Attributes.AddAttr("verified_name", strconv.FormatUint(veriFiledName, 10))
	}
	// enc node
	encNode := newxxmp.EmptyNode("enc")
	encNode.Attributes.AddAttr("v", "2")
	encNode.Attributes.AddAttr("type", encType)
	encNode.Attributes.AddAttr("mediatype", "contact") //contact
	encNode.SetData(c.Serialize())
	// set enc node
	messageNode.Children.AddNode(encNode)
	// message node attr phash
	if len(phash) > 0 && phash[0] != "" {
		messageNode.Attributes.AddAttr("phash", phash[0])
	}
	// message node participants
	if participants != nil {
		messageNode.Children.AddNode(participants)
	}

	p.Node = messageNode
	return p
}

// MessageProcessor
type MessageProcessor struct {
}

func NewMessageProcessor() *MessageProcessor {
	return &MessageProcessor{}
}
func (m *MessageProcessor) Handle(node *newxxmp.Node) interface{} {
	if node == nil {
		return nil
	}
	tag := node.GetTag()
	switch tag {
	case NodeMessage:
		return m.handleTagMessage(node)
	case NodeReceipt:
		return m.handleTagReceipt(node)
	}

	return nil
}

// handleTagMessage 处理 tag message
func (m *MessageProcessor) handleTagMessage(node *newxxmp.Node) interface{} {
	r := entity.EmptyChatMessage()
	// from
	attrFrom := node.GetAttributeByValue("from")
	r.SetFrom(attrFrom)
	// type
	attrType := node.GetAttributeByValue("type")
	r.SetContextType(attrType)
	// id
	attrId := node.GetAttributeByValue("id")
	r.SetId(attrId)
	// time
	attrTime := node.GetAttributeByValue("t")
	r.SetT(attrTime)
	// participant
	participant := node.GetAttributeByValue("participant")
	r.SetParticipant(participant)
	// enc
	encList := make([]*entity.Enc, 0)
	for _, chaild := range node.GetChildren() {
		if chaild.GetTag() == "enc" {

			encList = append(encList, entity.NewEnc(
				chaild.GetData(),
				chaild.GetAttributeByValue("type"),
				chaild.GetAttributeByValue("v")))
		}
	}
	r.SetEncList(encList)
	//m.ReceiveHandleMessage(r)
	return r
}

// handleTagReceipt 处理 tag Receipt
func (m *MessageProcessor) handleTagReceipt(node *newxxmp.Node) interface{} {
	var id, from, receiptType string
	// 发送 ack receipt
	// id
	id = node.GetAttributeByValue("id")
	// from
	from = node.GetAttributeByValue("from")
	// type
	receiptType = node.GetAttributeByValue("type")
	// participant
	participantAttr := node.GetAttributeByValue("participant")
	// create
	return entity.NewReceipt(from, id, receiptType, participantAttr)
}

// BuildMessage
func (m *MessageProcessor) BuildMessage(to string, veriFiledName uint64, msgType string, c protocol.CiphertextMessage, cs map[string]protocol.CiphertextMessage, phash ...string) *MessageNode {
	var participantsNode *newxxmp.Node
	if cs != nil {
		// 创建群聊后首次发送需要将秘钥分发给群成员
		participantsNode = createParticipants(cs)
	}
	// create message node
	messageNode := createMessageNode(to, veriFiledName, msgType, c, participantsNode, phash...)
	return messageNode
}

// BuildIqCreateIqSnsText 动态发送文本
func (i *IqProcessor) BuildIqCreateIqSnsText(veriFiledName uint64, jid string, c protocol.CiphertextMessage, cs map[string]protocol.CiphertextMessage, pHash string) (build *IqNode) {
	iqId := i.iqId()
	var participantsNode *newxxmp.Node
	if cs != nil {
		// 创建群聊后首次发送需要将秘钥分发给群成员
		participantsNode = createSnsParticipants(cs)
	}
	//create
	build = CreateIqSnsText(iqId, veriFiledName, jid, c, participantsNode, pHash)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/

	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateIqSnsText time out id:%d", iqId.Val()))
	})
	return build
}

func (m *MessageProcessor) BuildImageMessage(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, cs map[string]protocol.CiphertextMessage, phash ...string) *MessageNode {
	var participantsNode *newxxmp.Node
	if cs != nil {
		// 创建群聊后首次发送需要将秘钥分发给群成员
		participantsNode = createParticipants(cs)
	}
	// create message node
	messageNode := createImageMessageNode(veriFiledName, to, msgType, c, participantsNode, phash...)
	return messageNode
}
func (m *MessageProcessor) BuildAudioMessage(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, cs map[string]protocol.CiphertextMessage, phash ...string) *MessageNode {
	var participantsNode *newxxmp.Node
	if cs != nil {
		// 创建群聊后首次发送需要将秘钥分发给群成员
		participantsNode = createParticipants(cs)
	}
	// create message node
	messageNode := createAudioMessageNode(veriFiledName, to, msgType, c, participantsNode, phash...)
	return messageNode
}

// BuildVcardMessage
func (m *MessageProcessor) BuildVcardMessage(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, cs map[string]protocol.CiphertextMessage, phash ...string) *MessageNode {
	var participantsNode *newxxmp.Node
	if cs != nil {
		// 创建群聊后首次发送需要将秘钥分发给群成员
		participantsNode = createParticipants(cs)
	}
	// create message node
	messageNode := createVcardMessageNode(veriFiledName, to, msgType, c, participantsNode, phash...)
	return messageNode
}

// BuildVideoMessage
func (m *MessageProcessor) BuildVideoMessage(veriFiledName uint64, to, msgType string, c protocol.CiphertextMessage, cs map[string]protocol.CiphertextMessage, phash ...string) *MessageNode {
	var participantsNode *newxxmp.Node
	if cs != nil {
		// 创建群聊后首次发送需要将秘钥分发给群成员
		participantsNode = createParticipants(cs)
	}
	// create message node
	messageNode := createVideoMessageNode(veriFiledName, to, msgType, c, participantsNode, phash...)
	return messageNode
}

// BuildNormalReceipt
func (m MessageProcessor) BuildNormalReceipt(id, to, participant string, read bool) _interface.NodeBuilder {
	return createNormalReceipt(id, to, participant, read)
}
