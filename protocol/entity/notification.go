package entity

import "ws-go/protocol/newxxmp"

// Notification
type Notification struct {
	notifyType        string
	notifyChildrenTag string
	node              *newxxmp.Node
}

func (n *Notification) GetNode() *newxxmp.Node {
	return n.node
}

func (n *Notification) NotifyChildrenTag() string {
	return n.notifyChildrenTag
}

func (n *Notification) GetNotifyType() string {
	return n.notifyType
}

//NewNotification
func NewNotification(nType string, cTag string) *Notification {
	return &Notification{
		notifyChildrenTag: cTag,
		notifyType:        nType,
	}
}
