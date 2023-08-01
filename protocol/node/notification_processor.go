package node

import (
	"ws-go/protocol/entity"
	_interface "ws-go/protocol/iface"
	"ws-go/protocol/newxxmp"
)

const NodeNotification = "notification"
const (
	notificationTypeWgp2     = "w:gp2"
	notificationTypePicture  = "picture"
	notificationTypeContact  = "contacts"
	notificationTypeEncrypt  = "encrypt"
	notificationTypeBusiness = "business"
)

// NotificationProcessor
type NotificationProcessor struct {
	_interface.IBuildProcessor
}

// NewNotificationProcessor
func NewNotificationProcessor(p _interface.IBuildProcessor) *NotificationProcessor {
	return &NotificationProcessor{IBuildProcessor: p}
}

// handle
func (n *NotificationProcessor) handle(node *newxxmp.Node) interface{} {
	if node == nil || node.GetTag() != NodeNotification {
		return nil
	}
	// notification
	attrType := node.GetAttributeByValue("type")
	switch attrType {
	case notificationTypeWgp2:
		n.handleWgp2Notification(node)
	case notificationTypePicture:
		n.handlePictureNotification(node)
	case notificationTypeContact:
		n.handleContactNotification(node)
	case notificationTypeEncrypt:
		n.handleEncryptNotification(node)
	default:
		// 发送确认信号
		n.sendNotificationAck(*node)
	}
	// 返回给 handler 处理
	return entity.NewNotification(attrType, node.GetChildrenIndex(0).GetTag())
}

// handleContactNotification 同步联系人通知
func (n *NotificationProcessor) handleEncryptNotification(node *newxxmp.Node) {
	// 发送确认信号
	n.sendNotificationAck(*node)
	// TODO 继续处理
	children := node.GetChildrenIndex(0)
	switch children.GetTag() {
	case "identity":
		// TODO 获取新的秘钥
	}
}

// handleContactNotification 同步联系人通知
func (n *NotificationProcessor) handleContactNotification(node *newxxmp.Node) {
	// 发送确认信号
	n.sendNotificationAck(*node)
	// TODO 继续处理
	children := node.GetChildrenIndex(0)
	switch children.GetTag() {
	case "update":

	}
}

// handlePictureNotification
func (n *NotificationProcessor) handlePictureNotification(node *newxxmp.Node) {
	// 先发送确认信号
	n.sendNotificationAck(*node)
}

// handleWgp2Notification type w:gp2
func (n *NotificationProcessor) handleWgp2Notification(node *newxxmp.Node) {
	// 先发送确认信号
	n.sendNotificationAck(*node)
	// 处理群事件
	children := node.GetChildrenIndex(0)
	switch children.GetTag() {
	case "add": // 添加成员
	case "remove": //删除成员

	}
}

// sendNotificationAck 发送ack
func (n *NotificationProcessor) sendNotificationAck(node newxxmp.Node) {

	attrId := node.GetAttributeByValue("id")
	attrTo := node.GetAttributeByValue("to")
	if attrTo == "" {
		attrTo = node.GetAttributeByValue("from")
	}
	attrType := node.GetAttributeByValue("type")
	attrParticipant := node.GetAttributeByValue("participant")
	ackNode := createAck(attrId, attrTo, attrType, ClassNotification, attrParticipant)
	n.SendBuilder(ackNode)
}
