package entity

import (
	"encoding/json"
	"ws-go/protocol/waproto"
)

type decMsgContent struct {
	Msg *waproto.WAMessage
}

func (d *decMsgContent) SetContent(message *waproto.WAMessage) {
	d.Msg = message
}

func (d *decMsgContent) GetContent() *waproto.WAMessage {
	return d.Msg
}

type RespMessage struct {
	Message     interface{}
	Participant string
	From        string
	ContextType string
	Id          string
	T           string
}

type ChatMessage struct {
	*decMsgContent
	message     *waproto.Message
	encList     []*Enc
	participant string
	from        string
	contextType string
	id          string
	t           string
}

func (r *ChatMessage) SetMessage(message *waproto.Message) {
	r.message = message
}

func (r *ChatMessage) GetMessage() *waproto.Message {
	return r.message
}

func EmptyChatMessage() *ChatMessage {
	return &ChatMessage{
		decMsgContent: &decMsgContent{},
	}
}

func (r *ChatMessage) Participant() string {
	return r.participant
}

func (r *ChatMessage) SetParticipant(participant string) {
	r.participant = participant
}

func (r *ChatMessage) EncList() []*Enc {
	return r.encList
}

func (r *ChatMessage) SetEncList(encList []*Enc) {
	r.encList = encList
}

func (r *ChatMessage) T() string {
	return r.t
}

func (r *ChatMessage) SetT(t string) {
	r.t = t
}

func (r *ChatMessage) Id() string {
	return r.id
}

func (r *ChatMessage) SetId(id string) {
	r.id = id
}

func (r *ChatMessage) ContextType() string {
	return r.contextType
}

func (r *ChatMessage) SetContextType(contextType string) {
	r.contextType = contextType
}

func (r *ChatMessage) From() string {
	return r.from
}

func (r *ChatMessage) SetFrom(from string) {
	r.from = from
}

type PresenceResult struct {
	From   string
	Online string
	Last   string
}

type IqResult struct {
	ErrorEntity     *ErrorEntity
	PreKeys         []*USerPreKeys
	GroupInfo       *GroupInfo
	Invite          string
	USyncContacts   USyncContacts
	AddGroupMembers *AddGroup
	PictureInfo     *PictureInfo
	UserStatus      *UserStatus
	MediaConn       *MediaConn
	Qr              *Qr
	GroupAdmin      *GroupAdmin
	VerifiedName    *waproto.VerifiedName
	Categories      *Categories
}

func (i *IqResult) GetCategories() *Categories {
	return i.Categories
}

func (i *IqResult) GetGroupAdmin() *GroupAdmin {
	return i.GroupAdmin
}

func (i *IqResult) GetQr() *Qr {
	return i.Qr
}

func (i *IqResult) GetUserStatus() *UserStatus {
	return i.UserStatus
}
func (i *IqResult) GetMediaConn() *MediaConn {
	return i.MediaConn
}

func (i *IqResult) GetPictureInfo() *PictureInfo {
	return i.PictureInfo
}
func (i *IqResult) GetAddGroupResult() *AddGroup {
	return i.AddGroupMembers
}
func (i *IqResult) GetErrorEntityResult() *ErrorEntity {
	return i.ErrorEntity
}
func (i *IqResult) GetUSyncContacts() USyncContacts {
	return i.USyncContacts
}

func (i *IqResult) GetInviteCode() string {
	return i.Invite
}

func (i *IqResult) GetGroupInfo() *GroupInfo {
	return i.GroupInfo
}

// Get PreKeys
func (i *IqResult) GetPreKeys() []*USerPreKeys {
	return i.PreKeys
}

func (i *IqResult) String() string {
	d, err := json.Marshal(i)
	if err != nil {
		return err.Error()
	}
	return string(d)
}
