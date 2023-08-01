package entity

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/gf/frame/g"
	"log"
	"strconv"
	"strings"
	"ws-go/libsignal/ecc"
	"ws-go/libsignal/keys/identity"
	"ws-go/libsignal/keys/prekey"
	"ws-go/libsignal/util/bytehelper"
	"ws-go/libsignal/util/optional"
)

type ParticipantAttr map[string]*g.Var
type GroupInfo struct {
	id           string
	creator      string
	creation     string
	subject      string
	sT           string
	sO           string
	participants map[string]ParticipantAttr
	groupId      string
}

func (g GroupInfo) Participants() map[string]ParticipantAttr {
	return g.participants
}

func (g GroupInfo) ST() string {
	return g.sT
}
func (g GroupInfo) Subject() string {
	return g.subject
}
func (g GroupInfo) Creation() string {
	return g.creation
}
func (g GroupInfo) Creator() string {
	return g.creator
}
func (g GroupInfo) Id() string {
	return g.id
}
func (g GroupInfo) SO() string {
	return g.sO
}
func (g GroupInfo) GroupId() string {
	return g.groupId
}

func NewGroupInfo(id, creator, creation, subject, st, so string, participants map[string]ParticipantAttr, groupId string) *GroupInfo {
	return &GroupInfo{
		id:           id,
		creator:      creator,
		creation:     creation,
		subject:      subject,
		sT:           st,
		sO:           so,
		participants: participants,
		groupId:      groupId,
	}
}

// UserPreKeys
type USerPreKeys struct {
	JID            string
	RegistrationId uint32
	KeyType        byte
	Identity       ecc.ECPublicKeyable
	Key            ecc.ECPublicKeyable
	SKey           ecc.ECPublicKeyable
	SKeyId         uint32
	KeyId          uint32
	Signature      []byte
}

// USerPreKeys create user pre keys
func NewUserPreKeys(jid string, regId, Identity, Key, SKey, SKeyId, KeyId, Signature []byte) (*USerPreKeys, error) {
	log.Println("jid", jid)
	log.Println("regId", regId)
	log.Println("Identity", hex.EncodeToString(Identity))
	log.Println("Key", hex.EncodeToString(Key))
	log.Println("Skey", hex.EncodeToString(SKey))
	log.Println("SkeyId", hex.EncodeToString(SKeyId))
	log.Println("keyId", hex.EncodeToString(KeyId))
	log.Println("signature", hex.EncodeToString(Signature))

	keyId, err := strconv.ParseUint(hex.EncodeToString(KeyId), 16, 0)
	if err != nil {
		return nil, err
	}

	identityKey, err := ecc.DecodePoint(append([]byte{5}, Identity...), 0)
	if err != nil {
		return nil, err
	}

	Keyable, err := ecc.DecodePoint(append([]byte{5}, Key...), 0)
	if err != nil {
		return nil, err
	}

	sKey, err := ecc.DecodePoint(append([]byte{5}, SKey...), 0)
	if err != nil {
		return nil, err
	}

	return &USerPreKeys{
		JID:            jid,
		RegistrationId: binary.BigEndian.Uint32(regId),
		KeyType:        5,
		Identity:       identityKey,
		Key:            Keyable,
		SKeyId:         binary.BigEndian.Uint32(append([]byte{0}, SKeyId...)),
		SKey:           sKey,
		Signature:      Signature,
		KeyId:          uint32(keyId),
	}, nil
}

// CreatePreKeyBundle
func (u *USerPreKeys) CreatePreKeyBundle() *prekey.Bundle {
	//registrationID, deviceID uint32, preKeyID *optional.Uint32, signedPreKeyID uint32,
	//	preKeyPublic, signedPreKeyPublic ecc.ECPublicKeyable, signedPreKeySig [64]byte,
	//	identityKey *identity.Key
	return prekey.NewBundle(
		u.RegistrationId,
		0,
		optional.NewOptionalUint32(u.KeyId),
		u.SKeyId,
		u.Key,
		u.SKey,
		bytehelper.SliceToArray64(u.Signature),
		identity.NewKey(u.Identity),
	)
}

// GetJid
func (u *USerPreKeys) GetJid() string {
	split := strings.Split(u.JID, "@")
	return split[0]
}

// Enc
type Enc struct {
	data    []byte
	encType string
	v       string
}

func (e Enc) V() string {
	return e.v
}

func (e Enc) EncType() string {
	return e.encType
}
func (e Enc) Data() []byte {
	return e.data
}

func NewEnc(data []byte, encType, v string) *Enc {
	return &Enc{
		data:    data,
		encType: encType,
		v:       v,
	}
}

type RetryInfo struct {
	error error
	msg   *ChatMessage
	count *gtype.Int32
}

func (r RetryInfo) Count() *gtype.Int32 {
	return r.count
}

func (r RetryInfo) Msg() *ChatMessage {
	return r.msg
}

func (r RetryInfo) Error() error {
	return r.error
}

func NewRetryInfo(err error, msg *ChatMessage) *RetryInfo {
	return &RetryInfo{
		error: err,
		msg:   msg,
		count: gtype.NewInt32(0),
	}
}

type USyncContacts []*USyncInfo

func (u *USyncContacts) AddContact(contacts *USyncInfo) {
	if *u == nil {
		*u = make([]*USyncInfo, 0)
	}
	*u = append(*u, contacts)
}

type USyncInfo struct {
	jid         string
	status      string
	contact     string
	contactType string
}

func (U USyncInfo) Jid() string {
	return U.jid
}

func (U USyncInfo) ContactType() string {
	return U.contactType
}

func (U USyncInfo) Contact() string {
	return U.contact
}

func (U USyncInfo) Status() string {
	return U.status
}

func NewUSyncInfo(jid, status, contact, contactType string) *USyncInfo {
	return &USyncInfo{
		jid:         jid,
		status:      status,
		contact:     contact,
		contactType: contactType,
	}
}

type ErrorEntity struct {
	code string
	text string
}

func (U ErrorEntity) Code() string {
	return U.code
}
func (U ErrorEntity) Text() string {
	return U.text
}
func NewErrorEntity(code, text string) *ErrorEntity {
	return &ErrorEntity{
		code: code,
		text: text,
	}
}

type AddGroup struct {
	groupId string
	members map[string]ParticipantAttr
}

func (a AddGroup) Members() map[string]ParticipantAttr {
	return a.members
}

func (a AddGroup) GroupId() string {
	return a.groupId
}

func NewAddGroup(g string, members map[string]ParticipantAttr) *AddGroup {
	return &AddGroup{
		groupId: g,
		members: members,
	}
}

type PictureInfo struct {
	form    string
	picture string
	base64  string
}

func (p PictureInfo) Base64() string {
	return p.base64
}

func (p PictureInfo) Picture() string {
	return p.picture
}

func (p PictureInfo) Form() string {
	return p.form
}

func NewPictureInfo(form, picture, base64 string) *PictureInfo {
	return &PictureInfo{
		form:    form,
		base64:  base64,
		picture: picture,
	}
}

type UserStatus struct {
	toWid     string
	signature string
	t         string
}

func (p UserStatus) ToWid() string {
	return p.toWid
}

func (p UserStatus) Signature() string {
	return p.signature
}
func (p UserStatus) T() string {
	return p.t
}
func NewUserStatus(toWid, signature, t string) *UserStatus {
	return &UserStatus{
		toWid:     toWid,
		signature: signature,
		t:         t,
	}
}

type Categories struct {
	Category    []*Category
	NotABizName string
	NotABizId   string
}

func NewCategories(Category []*Category, NotABizName, NotABizId string) *Categories {
	return &Categories{
		Category:    Category,
		NotABizId:   NotABizId,
		NotABizName: NotABizName,
	}
}

type Category struct {
	Id   string
	Name string
}
type GroupAdmin struct {
	Jid  string
	Type string
}

func NewGroupAdmin(jid string, types string) *GroupAdmin {
	return &GroupAdmin{
		Jid:  jid,
		Type: types,
	}
}

type Qr struct {
	code     string
	jid      string
	notify   string
	typeNode string
}

func (p Qr) Notify() string {
	return p.notify
}
func (p Qr) TypeNode() string {
	return p.typeNode
}
func (p Qr) Code() string {
	return p.code
}
func (p Qr) Jid() string {
	return p.jid
}
func NewQr(code, jid string, typeNode string, notify string) *Qr {
	return &Qr{
		jid:      jid,
		code:     code,
		typeNode: typeNode,
		notify:   notify,
	}
}

type MediaConn struct {
	hostname string
	auth     string
}

func (p MediaConn) HostName() string {
	return p.hostname
}

func (p MediaConn) Auth() string {
	return p.auth
}

func NewMediaConn(hostname string, auth string) *MediaConn {
	return &MediaConn{
		hostname: hostname,
		auth:     auth,
	}
}
