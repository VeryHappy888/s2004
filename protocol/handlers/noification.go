package handlers

import (
	"errors"
	"strings"
	"ws-go/protocol/axolotl"
	"ws-go/protocol/entity"
	"ws-go/protocol/iface"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/stores"
)

const NotificationHandlerTag = "NotificationHandler"

func NewNotificationHandler(contactStores *stores.ContactStores, axolotl *axolotl.Manager, api iface.INodeApi) iface.Handler {
	handler := &NotificationHandler{
		contactStore: contactStores,
		Axolotl:      axolotl,
		NodeApi:      api,
		baseHandler:  newBaseHandler(NotificationHandlerTag, 10),
	}
	return handler
}

// NotificationHandle 对通知消息进行处理
type NotificationHandler struct {
	*baseHandler
	contactStore *stores.ContactStores
	NodeApi      iface.INodeApi
	Axolotl      *axolotl.Manager
}

func (n *NotificationHandler) AddHandleTask(i interface{}) error {
	if i == nil {
		return errors.New("Tasks cannot be empty")
	}
	if _, ok := i.(entity.Notification); !ok {
		return errors.New("Is not the handler's processing object")
	}
	// add  task
	n.Add(i)
	return nil
}

func (n *NotificationHandler) LoopQueue() {

	for {
		v := n.queue.Pop()
		if v == nil {
			if !n.queueClose {
				n.Close()
			}
			break
		} else {
			// chat message
			if notification, ok := v.(*entity.Notification); ok {
				n.handleNotification(notification)
			}
		}
	}

}

// handleNotification
func (n *NotificationHandler) handleNotification(notification *entity.Notification) {
	notifyType := notification.GetNotifyType()
	childrenTag := notification.NotifyChildrenTag()
	// notify type is encrypt
	if notifyType == "encrypt" && childrenTag == "identity" {
		from := notification.GetNode().GetAttributeByValue("from")
		n.encrypt(from)
	}
	// notify type is w:gp2
	if notifyType == "w:gp2" {
		n.wgp2(notification.GetNode())

	}
	// notify contacts
	if notifyType == "contacts" {
		n.contacts(notification.GetNode())
	}
}

func (n *NotificationHandler) contacts(node *newxxmp.Node) {
	tag := node.GetChildrenIndex(0).GetTag()
	// update
	if strings.Contains(tag, "update") {

	}

	// add
	if strings.Contains(tag, "add") {

	}
}
func (n *NotificationHandler) wgp2(node *newxxmp.Node) {
	tag := node.GetChildrenIndex(0).GetTag()
	gJid := node.GetAttributeByValue("from")

	// 创建群聊
	if strings.Contains(tag, "create") {
		groupNode := node.GetChildrenIndex(0).GetChildrenIndex(0)
		subject := groupNode.GetAttributeByValue("subject")
		for _, participantNode := range groupNode.GetChildren() {
			var admin int
			jid := participantNode.GetAttributeByValue("jid")
			if strings.Contains(participantNode.GetAttributeByValue("type"), "superadmin") {
				admin = 1
			}
			_ = n.contactStore.AddGroupParticipant(gJid, jid, admin)
		}
		// add group to  contact
		_ = n.contactStore.AddContact(gJid, subject, "", "")
	}
	// 添加群成员
	if strings.Contains(tag, "add") {

	}
	//设置管理员
	if strings.Contains(tag, "promote") {

	}
	// 退出群聊
	if strings.Contains(tag, "remove") {
		jid := node.GetChildrenIndex(0).GetChildrenIndex(0).GetAttributeByValue("jid")
		// del group participant
		_ = n.contactStore.DelGroupParticipant(gJid, jid)
	}

}

// encrypt
func (n *NotificationHandler) encrypt(from string) {
	_ = n.NodeApi.GetPreKeys(true, from)
}
