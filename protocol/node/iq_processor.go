package node

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gogf/gf/container/gmap"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/gf/container/gvar"
	"github.com/gogf/gf/os/gtimer"
	"github.com/gogf/guuid"
	"github.com/golang/protobuf/proto"
	"log"
	"strconv"
	"strings"
	"time"
	"ws-go/libsignal/keys/identity"
	"ws-go/libsignal/state/record"
	"ws-go/libsignal/util/bytehelper"
	entity "ws-go/protocol/entity"
	_interface "ws-go/protocol/iface"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/utils"
	promise "ws-go/protocol/utils/promise"
	"ws-go/protocol/waproto"
)

const (
	NodeIq            = "iq"
	NodeConfig        = "config"
	IQ_PREKEYS_RESULT = "iq_prekeys_result"
)

var (
	IdNotExistError = errors.New("id does not exist")
)

// IqNode
type IqNode struct {
	d []byte
	*newxxmp.Node
	id          int32 // iqId
	compression bool
	promise     *promise.Promise

	// result
	addGroup     *entity.AddGroup
	preKeys      []*entity.USerPreKeys
	groupInfo    *entity.GroupInfo
	syncContacts entity.USyncContacts
	pictureInfo  *entity.PictureInfo
	inviteCode   string
	userStatus   *entity.UserStatus
	mediaConn    *entity.MediaConn
	qr           *entity.Qr
	errorEntity  *entity.ErrorEntity
	groupAdmin   *entity.GroupAdmin
	verifiedName *waproto.VerifiedName
	categories   *entity.Categories
}

func (i *IqNode) GetIqId() int32 {
	return i.id
}

// SetNewListenHandler 用于回调 无阻塞
func (i *IqNode) SetNewListenHandler(handler *_interface.PromiseHandler) {
	if i.promise == nil && handler == nil {
		return
	}
	// 回调成功
	if handler.SuccessFunc != nil {
		i.promise.Then(func(data promise.Any) promise.Any {
			handler.SuccessFunc(data)
			return nil
		})
	}
	// 回调报错
	if handler.FailureFunc != nil {
		i.promise.Catch(func(err error) error {
			handler.FailureFunc(err)
			return err
		})
	}
}

// SetCallBack 用于回调 无阻塞
func (i *IqNode) SetListenHandler(success func(any promise.Any), failure func(err error)) {
	if i.promise == nil {
		return
	}
	// 回调成功
	if success != nil {
		i.promise.Then(func(data promise.Any) promise.Any {
			success(data)
			return nil
		})
	}
	// 回调报错
	if failure != nil {
		i.promise.Catch(func(err error) error {
			failure(err)
			return err
		})
	}
}

// GetResult waiting result return  thread blocking
func (i *IqNode) GetResult() (promise.Any, error) {
	if i.promise == nil {
		return nil, errors.New("not set promise")
	}
	return i.promise.Await()
}
func (i *IqNode) SetPromise(p *promise.Promise) {
	i.promise = p
}
func (i *IqNode) GetPromise() *promise.Promise {
	return i.promise
}

//Process 请求之后需要回调
func (i *IqNode) Process(node *newxxmp.Node) {
	if i.preKeys == nil {
		i.preKeys = make([]*entity.USerPreKeys, 0)
	}
	// 检查是否为nil
	if node.CheckNil() {
		return
	}

	iqType := node.GetAttribute("type").Value()
	switch iqType {
	case "result":
		i.handleResult(node)
		break
	case "error":
		i.handleError(node)
		break
	}

	// TODO 处理完成后 需要回调
	if i.promise != nil {
		i.promise.SuccessResolve(entity.IqResult{
			PreKeys:         i.preKeys,
			GroupInfo:       i.groupInfo,
			Invite:          i.inviteCode,
			USyncContacts:   i.syncContacts,
			AddGroupMembers: i.addGroup,
			PictureInfo:     i.pictureInfo,
			UserStatus:      i.userStatus,
			MediaConn:       i.mediaConn,
			Qr:              i.qr,
			ErrorEntity:     i.errorEntity,
			GroupAdmin:      i.groupAdmin,
			VerifiedName:    i.verifiedName,
			Categories:      i.categories,
		})
	}
}

// handleResult
func (i *IqNode) handleResult(node *newxxmp.Node) {
	if node == nil {
		return
	}
	// children
	childNode := node.GetChildrenIndex(0)
	if childNode == nil {
		log.Println(i.GetChildrenIndex(0).GetTag(), "result ->", node.GetString())
		return
	}
	//fmt.Println("getTag--->", childNode.GetTag())
	switch childNode.GetTag() {
	case "list":
		for _, n := range childNode.GetChildren() {
			if n.GetTag() == "user" {
				i.handleKeys(n)
			}
		}
	case "group":
		i.handleGroup(childNode)
	case "invite":
		i.handleInvite(childNode)
	case "add":
		i.handleAddGroup(node)
	case "usync":
		i.handleUSync(childNode)
	case "picture":
		i.handlePicture(node)
	case "status":
		i.handleGetStatus(node)
	case "media_conn":
		i.handleMediaConn(node)
	case "qr":
		i.handleQr(node)
	case "verified_name":
		i.handleVerifiedName(node)
	case "response":
		i.handleResponse(node)
	default:

	}
}

func (i *IqNode) handleError(node *newxxmp.Node) {
	qrNode := node.GetChildrenByTag("error")
	code := qrNode.GetAttributeByValue("code")
	text := qrNode.GetAttributeByValue("text")
	i.errorEntity = entity.NewErrorEntity(code, text)
}

//获取response数据
func (i *IqNode) handleResponse(node *newxxmp.Node) {
	responseNode := node.GetChildrenByTag("response")
	if responseNode != nil {
		categoriesNode := responseNode.GetChildrenByTag("categories")
		if categoriesNode != nil {
			listCategoriesNodes := categoriesNode.GetChildren()
			newList := make([]*entity.Category, 0)
			for i := 0; i < len(listCategoriesNodes); i++ {
				v := &entity.Category{
					Id:   listCategoriesNodes[i].GetAttributeByValue("id"),
					Name: string(listCategoriesNodes[i].GetData()),
				}
				newList = append(newList, v)
			}
			noTaBiz := responseNode.GetChildrenByTag("not_a_biz")
			bizCategoryNode := noTaBiz.GetChildrenByTag("category")
			id := bizCategoryNode.GetAttributeByValue("id")
			name := bizCategoryNode.GetData()
			i.categories = entity.NewCategories(newList, id, string(name))
		}
	}
}

// handlePicture
func (i *IqNode) handlePicture(node *newxxmp.Node) {
	from := node.GetAttributeByValue("from")
	pictureNode := node.GetChildrenByTag("picture")
	url := pictureNode.GetAttributeByValue("url")
	base64Str := ""
	data := pictureNode.Data
	if data != nil {
		base64Str = base64.StdEncoding.EncodeToString(pictureNode.GetData())
	}
	i.pictureInfo = entity.NewPictureInfo(from, url, base64Str)
}

// 状态需要回调
func (i *IqNode) handleGetStatus(node *newxxmp.Node) {
	statusNode := node.GetChildrenByTag("status")
	userNode := statusNode.GetChildrenByTag("user")
	jid := userNode.GetAttributeByValue("jid")
	t := userNode.GetAttributeByValue("t")
	signature := string(userNode.GetData())
	i.userStatus = entity.NewUserStatus(jid, signature, t)
}

//获取cdn
func (i *IqNode) handleMediaConn(node *newxxmp.Node) {
	mediaConNode := node.GetChildrenByTag("media_conn")
	auth := mediaConNode.GetAttributeByValue("auth")
	hostname := mediaConNode.GetChildrenIndex(0).GetAttributeByValue("hostname")
	i.mediaConn = entity.NewMediaConn(hostname, auth)
}

//获取二维码code
func (i *IqNode) handleQr(node *newxxmp.Node) {
	qrNode := node.GetChildrenByTag("qr")
	code := qrNode.GetAttributeByValue("code")
	jid := qrNode.GetAttributeByValue("jid")
	typeNode := qrNode.GetAttributeByValue("type")
	notify := qrNode.GetAttributeByValue("notify")
	i.qr = entity.NewQr(code, jid, typeNode, notify)
}

//获取verified数据
func (i *IqNode) handleVerifiedName(node *newxxmp.Node) {
	verifieldNode := node.GetChildrenByTag("verified_name")
	if verifieldNode.Data != nil {
		ps := &waproto.VerifiedName{}
		err := proto.Unmarshal(verifieldNode.GetData(), ps)
		if err != nil {
			log.Println("获取verified数据出现异常", err.Error())
		}
		i.verifiedName = ps
	}
}

// handleUSync
func (i *IqNode) handleUSync(node *newxxmp.Node) {
	//uSyncNode := node.GetChildrenIndex(0)
	listNode := node.GetChildrenByTag("list")
	usyncContacts := entity.USyncContacts{}
	for _, userNode := range listNode.GetChildren() {
		var status, ctype, contact string
		jid := userNode.GetAttributeByValue("jid")
		// status
		statusNode := userNode.GetChildrenByTag("status")
		if statusNode != nil {
			if statusNode.GetAttributeByValue("type") == "" {
				status = string(statusNode.GetData())
			} else {
				status = "fail"
			}
		}
		// contact
		contactNode := userNode.GetChildrenByTag("contact")
		if contactNode != nil {
			ctype = contactNode.GetAttributeByValue("type")
			contact = contactNode.GetAttributeByValue("contact")
		}
		usyncContacts.AddContact(entity.NewUSyncInfo(jid, status, ctype, contact))
	}
	i.syncContacts = usyncContacts
}

// handleAddGroup
func (i *IqNode) handleAddGroup(node *newxxmp.Node) {
	from := node.GetAttributeByValue("from")
	// add node
	addNode := node.GetChildrenIndex(0)
	participants := make(map[string]entity.ParticipantAttr)
	for _, n := range addNode.GetChildren() {
		participant := make(entity.ParticipantAttr)
		jid := n.GetAttributeByValue("jid")
		err := n.GetAttributeByValue("error")
		if err != "" {
			participant["error"] = gvar.New(err)
		}
		participants[jid] = participant
	}
	i.addGroup = entity.NewAddGroup(from, participants)
}

// handleInvite
func (i *IqNode) handleInvite(node *newxxmp.Node) {
	i.inviteCode = node.GetAttributeByValue("code")
}

// handleGroup
func (i *IqNode) handleGroup(node *newxxmp.Node) {
	if node == nil {
		return
	}
	// id
	attrId := node.GetAttributeByValue("id")
	// creator
	attrCreator := node.GetAttributeByValue("creator")
	// creation
	attrCreation := node.GetAttributeByValue("creation")
	//subject
	attrSubject := node.GetAttributeByValue("subject")
	// s_t
	attrSt := node.GetAttributeByValue("s_t")
	// s_o
	attrSo := node.GetAttributeByValue("s_o")

	groupId := node.GetAttributeByValue("jid")
	// participants
	participants := make(map[string]entity.ParticipantAttr)
	for _, child := range node.GetChildren() {
		attrJid := child.GetAttributeByValue("jid")
		if attrJid != "" {
			attr := make(entity.ParticipantAttr)
			attrErr := child.GetAttributeByValue("err")
			if attrErr != "" {
				attr["error"] = gvar.New(attrErr)
			}
			attrType := child.GetAttributeByValue("type")
			if attrType != "" {
				attr["type"] = gvar.New(attrType, true)
			}
			participants[attrJid] = attr
		}
	}
	i.groupInfo = entity.NewGroupInfo(
		attrId,
		attrCreator,
		attrCreation,
		attrSubject,
		attrSt,
		attrSo,
		participants,
		groupId)
}

// handlerKeys iq -> []list -> user
func (i *IqNode) handleKeys(node *newxxmp.Node) {
	if node == nil {
		return
	}

	var regId, identity, key, sKey, sKeyId, keyId, signature []byte

	// jid
	jid := node.GetAttribute("jid").Value()
	for _, user := range node.GetChildren() {
		switch user.GetTag() {
		case "registration":
			regId = user.GetData()
			log.Println("registration:", hex.EncodeToString(regId))
		case "type":
			break
		case "identity":
			identity = user.GetData()
			log.Println("identity:", hex.EncodeToString(identity))
		case "skey":
			sKeyId = user.GetChildrenByTag("id").GetData()
			log.Println("sKeyId:", hex.EncodeToString(sKeyId))

			sKey = user.GetChildrenByTag("value").GetData()
			log.Println("sKey:", hex.EncodeToString(sKey))

			signature = user.GetChildrenByTag("signature").GetData()
			log.Println("signature:", hex.EncodeToString(signature))
		case "key":
			keyId = user.GetChildrenByTag("id").GetData()
			key = user.GetChildrenByTag("value").GetData()
			log.Println("key:", hex.EncodeToString(key))
		}

		//preKeys = append(preKeys,perKey)
	}

	preKey, err := entity.NewUserPreKeys(jid, regId, identity, key, sKey, sKeyId, keyId, signature)
	if err != nil {
		// TODO notify error promise 发送错误
		return
	}
	i.preKeys = append(i.preKeys, preKey)
}

// Builder build node to ixxmp data
func (i *IqNode) Builder() ([]byte, error) {
	if len(i.d) > 0 {
		return i.d, nil
	}
	xxmpData := i.GetTokenArray().GetBytes(true)
	log.Println("builder node ", i.Node.GetString())
	return xxmpData, nil
}

// createIqUserKeys 创建同步Keys
func crateIqUserKeys(users []string, id gtype.Int32, reason bool) *IqNode {
	//<iq id="02" xmlns="encrypt" type="get" to="@s.whatsapp.net">
	//	<key>
	//		<user S="8618665179087@s.whatsapp.net"/>
	//	</key>
	//</iq>

	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqAttributes := make([]newxxmp.Attribute, 4)
	iqAttributes[0] = newxxmp.NewAttribute("id", id.String())
	iqAttributes[1] = newxxmp.NewAttribute("xmlns", "encrypt")
	iqAttributes[2] = newxxmp.NewAttribute("type", "get")
	iqAttributes[3] = newxxmp.NewAttribute("to", "@s.whatsapp.net")
	//uses
	usersNode := make([]*newxxmp.Node, 0)
	for _, user := range users {
		userAttribute := make([]newxxmp.Attribute, 0)
		if !strings.Contains(user, "@s.whatsapp.net") {
			userAttribute = append(userAttribute, newxxmp.NewAttribute("jid", user+"@s.whatsapp.net"))
		} else {
			userAttribute = append(userAttribute, newxxmp.NewAttribute("jid", user))
		}

		if reason {
			userAttribute = append(userAttribute, newxxmp.NewAttribute("reason", "identity"))
		}

		usersNode = append(usersNode, &newxxmp.Node{Tag: "user", Attributes: userAttribute})
	}

	iqNode := &newxxmp.Node{
		Tag:        "iq",
		Attributes: iqAttributes,
		Children: []*newxxmp.Node{
			{
				Tag:      "key",
				Children: usersNode,
			},
		},
		Data: nil,
	}
	i.Node = iqNode
	log.Println("crateIqUserKeys create new node ", i.Node.GetString(), "id->", id)
	return i
}

// createIqConfig send urn:xmpp:whatsapp:push config
func createIqConfig(id gtype.Int32) *IqNode {
	/*
		<iq id='1' xmlns='urn:xmpp:whatsapp:push' type='get' to='s.whatsapp.net'><config version='1'/></iq>
	*/

	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}

	urn_xmpp_whatsapp_push := &newxxmp.Node{
		Tag: "iq",
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("id", id.String()),
			newxxmp.NewAttribute("xmlns", "urn:xmpp:whatsapp:push"),
			newxxmp.NewAttribute("type", "get"),
			newxxmp.NewAttribute("to", "s.whatsapp.net"),
		},
		Children: []*newxxmp.Node{
			{
				Tag: "config",
				Attributes: []newxxmp.Attribute{
					newxxmp.NewAttribute("version", "1"),
				},
			},
		},
		Data: nil,
	}
	i.Node = urn_xmpp_whatsapp_push
	log.Println("crateIqUserKeys create new node ", i.Node.GetString(), "id->", id)
	return i
}

//createIqConfigOne
func createIqConfigOne(id gtype.Int32) *IqNode {
	/*
		<iq id='2' xmlns='w' type='get' to='s.whatsapp.net'><props protocol='2' hash=''/></iq>
	*/
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}

	urn_xmpp_whatsapp_push := &newxxmp.Node{
		Tag: "iq",
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("id", id.String()),
			newxxmp.NewAttribute("xmlns", "w"),
			newxxmp.NewAttribute("type", "get"),
			newxxmp.NewAttribute("to", "s.whatsapp.net"),
		},
		Children: []*newxxmp.Node{
			{
				Tag: "props",
				Attributes: []newxxmp.Attribute{
					newxxmp.NewAttribute("protocol", "2"),
					newxxmp.NewAttribute("hash", ""),
				},
			},
		},
		Data: nil,
	}
	i.Node = urn_xmpp_whatsapp_push
	return i
}

// createIqConfigTwo
func createIqConfigTwo(id gtype.Int32) *IqNode {
	/*
		<iq id='2' xmlns='w' type='get' to='s.whatsapp.net'><props protocol='2' hash=''/></iq>
	*/
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}

	urn_xmpp_whatsapp_push := &newxxmp.Node{
		Tag: "iq",
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("id", id.String()),
			newxxmp.NewAttribute("xmlns", "abt"),
			newxxmp.NewAttribute("type", "get"),
			newxxmp.NewAttribute("to", "s.whatsapp.net"),
		},
		Children: []*newxxmp.Node{
			{
				Tag: "props",
				Attributes: []newxxmp.Attribute{
					newxxmp.NewAttribute("protocol", "1"),
					newxxmp.NewAttribute("hash", ""),
				},
			},
		},
		Data: nil,
	}
	i.Node = urn_xmpp_whatsapp_push
	return i
}

// createIqActive
func createIqActive(id gtype.Int32) *IqNode {
	/*
		<iq id="2" xmlns="passive" type="set" to="@s.whatsapp.net">
		    <active/>
		</iq>
	*/
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "passive")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", "@s.whatsapp.net")
	// active node
	activeNode := newxxmp.EmptyNode("active")
	iqNode.Children.AddNode(activeNode)

	i.Node = iqNode
	return i
}

func createGetVerifiedName(id gtype.Int32, jid string) *IqNode {
	//<iq id='015' xmlns='w:biz' type='get'><verified_name jid='447466992083@s.whatsapp.net'/></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:biz")
	iqNode.Attributes.AddAttr("type", "get")
	verifiedNode := newxxmp.EmptyNode("verified_name")
	verifiedNode.Attributes.AddAttr("jid", jid)
	iqNode.Children.AddNode(verifiedNode)
	i.Node = iqNode
	return i
}

func createSendCategories(id gtype.Int32) *IqNode {
	//<iq to='s.whatsapp.net' type='get' id='0f' xmlns='fb:thrift_iq'><request type='catkit' op='typeahead' v='1'><query></query></request></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	iqNode.Attributes.AddAttr("xmlns", "fb:thrift_iq")
	iqNode.Attributes.AddAttr("type", "get")
	requestNode := newxxmp.EmptyNode("request")
	requestNode.Attributes.AddAttr("type", "catkit")
	requestNode.Attributes.AddAttr("op", "typeahead")
	requestNode.Attributes.AddAttr("v", "1")
	requestNode.Children.AddNode(newxxmp.EmptyNode("query"))
	iqNode.Children.AddNode(requestNode)
	i.Node = iqNode
	return i
}

//createBusinessProfile
func createBusinessProfile(id gtype.Int32, categoryId string) *IqNode {
	//<iq id='011' xmlns='w:biz' type='set'><business_profile v='116'><categories><category id='1223524174334504'/></categories></business_profile></iq>
	//<iq id='010' xmlns='w:biz' type='set'><business_profile v='116' mutation_type='delta'><categories><category id='2250'/></categories></business_profile></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:biz")
	iqNode.Attributes.AddAttr("type", "set")
	requestNode := newxxmp.EmptyNode("business_profile")
	requestNode.Attributes.AddAttr("v", "116")
	requestNode.Attributes.AddAttr("mutation_type", "delta")
	categoriesNode := newxxmp.EmptyNode("categories")
	categoryNode := newxxmp.EmptyNode("category")
	categoryNode.Attributes.AddAttr("id", categoryId)
	categoriesNode.Children.AddNode(categoryNode)
	requestNode.Children.AddNode(categoriesNode)
	iqNode.Children.AddNode(requestNode)
	i.Node = iqNode
	return i
}

// createBusinessProfileTow
func createBusinessProfileTow(id gtype.Int32, jid string) *IqNode {
	//<iq id='014' xmlns='w:biz' type='get'><business_profile v='116'><profile jid='6283166028332@s.whatsapp.net'/></business_profile></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:biz")
	iqNode.Attributes.AddAttr("type", "get")
	requestNode := newxxmp.EmptyNode("business_profile")
	requestNode.Attributes.AddAttr("v", "116")
	categoriesNode := newxxmp.EmptyNode("profile")
	categoriesNode.Attributes.AddAttr("jid", jid)
	requestNode.Children.AddNode(categoriesNode)
	iqNode.Children.AddNode(requestNode)
	i.Node = iqNode
	return i
}

// createIqPing
func createIqPing(id gtype.Int32) *IqNode {
	//<iq id="2" xmlns="w:p" type="get" to="@s.whatsapp.net">
	//    <ping/>
	//</iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// node
	pingNode := &newxxmp.Node{
		Tag: NodeIq,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("id", id.String()),
			newxxmp.NewAttribute("xmlns", "w:p"),
			newxxmp.NewAttribute("type", "get"),
			newxxmp.NewAttribute("to", "s.whatsapp.net"),
		},
		Children: []*newxxmp.Node{
			{Tag: "ping"},
		},
	}
	i.Node = pingNode
	return i
}

// createPreKeys
func createPreKeys(key *record.PreKey) *newxxmp.Node {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("createPreKeys panic: %v\n", r)
		}
	}()
	// key node
	keyNode := newxxmp.EmptyNode("key")
	idNode := newxxmp.EmptyNode("id", utils.PutInt24(int(key.ID().Value)))
	bytes := key.KeyPair().PublicKey().Serialize()
	//log.Println("key:",hex.EncodeToString(bytes),key.KeyPair().PublicKey().Serialize()[1:32])
	valueNode := newxxmp.EmptyNode("value", bytes[1:])
	// set node
	keyNode.Children.AddNode(idNode)
	keyNode.Children.AddNode(valueNode)
	return keyNode
}

// createSignedPreKeys
func createSignedPreKeys(skey *record.SignedPreKey) *newxxmp.Node {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("createSignedPreKeys panic: %v\n", r)
		}
	}()
	sKeyNode := newxxmp.EmptyNode("skey")
	// id
	idNode := newxxmp.EmptyNode("id", utils.PutInt24(int(skey.ID())))
	sKeyNode.Children.AddNode(idNode)
	// value
	valueNode := newxxmp.EmptyNode("value", skey.KeyPair().PublicKey().Serialize()[1:])
	sKeyNode.Children.AddNode(valueNode)
	// value
	signatureNode := newxxmp.EmptyNode("signature", bytehelper.ArrayToSlice64(skey.Signature()))
	sKeyNode.Children.AddNode(signatureNode)

	return sKeyNode
}

// createIqPreKeyListNode
func createIqPreKeyListNode(preKeys []*record.PreKey) *newxxmp.Node {
	// list node
	listNode := newxxmp.EmptyNode("list")
	for _, key := range preKeys {
		// set node to key node
		listNode.Children.AddNode(createPreKeys(key))
	}
	return listNode
}

// createIqSetEncryptKeys 上传  identity keys
func createIqSetEncryptKeys(id gtype.Int32, preKeys []*record.PreKey, skey *record.SignedPreKey, identityKey identity.Key, regId uint32) *IqNode {
	/*
		<iq id="3" xmlns="encrypt" type="set" to="@s.whatsapp.net">
		    <identity>6b8289e7a3a6cda9ffb5a132c8a79f9ded99419034db058f7f87a04b48f1f811</identity>
		    <registration>fc0426936cd0</registration>
		    <type>fc0105</type>
		    <list>
		        <key>
		            <id>fc037a904f</id>
		            <value>fc207e36b51bd3a25c8b4cae6e975abacfa52c6fbb2c9e9b033fd9482b2853ae1922</value>
		        </key>
		    </list>
		    <skey>
		        <id>fc03000000</id>
		        <value>fc2048266d76f392ff7469316c7c6fce603fb6424825ea8762e0f8ac7276f3f36a0e</value>
		        <signature>fc40bf0fe96dae238036c140eb9d68b8a05814bad89ddc65373e33c9d87d5efdb681cff2e3c6fbd1588b1750669f2338312afb82e82c0e0029795f05b20efa33290d</signature>
		    </skey>
		</iq>
	*/
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "encrypt")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", "@s.whatsapp.net")
	// identity node
	identityNode := newxxmp.EmptyNode("identity")
	identityNode.SetData(identityKey.PublicKey().Serialize()[1:])
	// set identity node to iq node
	iqNode.Children.AddNode(identityNode)
	// registration node
	registrationNode := newxxmp.EmptyNode("registration")
	// uint32 to bytes
	regIdData := make([]byte, 4)
	binary.BigEndian.PutUint32(regIdData, regId)
	registrationNode.SetData(regIdData)
	// set registration node to iq node
	iqNode.Children.AddNode(registrationNode)
	// type node
	typeNode := newxxmp.EmptyNode("type", []byte{0x05})
	iqNode.Children.AddNode(typeNode)
	// set key list node
	iqNode.Children.AddNode(createIqPreKeyListNode(preKeys))
	// set skey node
	iqNode.Children.AddNode(createSignedPreKeys(skey))

	i.Node = iqNode
	return i
}

// createIqUSync 同步联系人
func createIqUSync(id gtype.Int32, contacts []string) *IqNode {
	/*
			扫码同步
			<iq
		    xmlns='usync' id='0f' type='get'>
		    <usync sid='sync_sid_delta_9c4d2691-119f-46b1-a54c-edbe427d798e' index='0' last='true' mode='delta' context='interactive'>
		        <query>
		            <contact/>
		            <status/>
		            <business>
		                <verified_name/>
		                <profile v='116'/>
		            </business>
		            <devices version='2'/>
		        </query>
		        <list>
		        </list>
		    </usync>
		</iq>
	*/
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// new empty node
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("xmlns", "usync")
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("type", "get")
	// usync node
	usyncNode := newxxmp.EmptyNode("usync")
	usyncNode.Attributes.AddAttr("sid", "sync_sid_delta_"+guuid.New().String())
	usyncNode.Attributes.AddAttr("index", "0")
	usyncNode.Attributes.AddAttr("last", "true")
	usyncNode.Attributes.AddAttr("mode", "delta")
	usyncNode.Attributes.AddAttr("context", "interactive")
	// query node
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Children.AddNode(newxxmp.EmptyNode("contact"))
	queryNode.Children.AddNode(newxxmp.EmptyNode("status"))
	// business node
	/*businessNode := newxxmp.EmptyNode("business")
	businessNode.Children.AddNode(newxxmp.EmptyNode("verified_name"))
	businessNode.Children.AddNode(newxxmp.EmptyNode("profile", newxxmp.NewAttribute("v", "116")))
	// set business node to  query node
	queryNode.Children.AddNode(businessNode)*/
	// set devices version=2
	devicesNode := newxxmp.EmptyNode("devices")
	devicesNode.Attributes.AddAttr("version", "2")
	queryNode.Children.AddNode(devicesNode)
	// set query node to usync node
	usyncNode.Children.AddNode(queryNode)
	// list node
	listNode := newxxmp.EmptyNode("list")
	for _, contact := range contacts {
		userNode := newxxmp.EmptyNode("user")
		contactNode := newxxmp.EmptyNode("contact")
		if strings.Index(contact, "+") == -1 {
			contact = "+" + contact
		}
		fmt.Println(contact)
		contactNode.SetData([]byte(contact))
		userNode.Children.AddNode(contactNode)
		listNode.Children.AddNode(userNode)
	}
	usyncNode.Children.AddNode(listNode)
	iqNode.Children.AddNode(usyncNode)
	i.Node = iqNode
	return i
}

// createIqUSyncAdd 扫码后第一次同步
func createIqUSyncAdd(id gtype.Int32, contacts []string) *IqNode {
	/*<iq
	    xmlns='usync' id='01e' type='get'>
	    <usync sid='sync_sid_query_0ad2cd71-180b-4f16-8560-503bfc6e1419' index='0' last='true' mode='query' context='add'>
	        <query>
	            <contact/>
	            <status/>
	            <business>
	                <verified_name/>
	                <profile v='116'/>
	            </business>
	            <picture type='preview'/>
	        </query>
	        <list>
	            <user jid='19048786157@s.whatsapp.net'></user>
	        </list>
	    </usync>
	</iq>*/
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// new empty node
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("xmlns", "usync")
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("type", "get")
	// usync node
	usyncNode := newxxmp.EmptyNode("usync")
	usyncNode.Attributes.AddAttr("sid", "sync_sid_query_"+guuid.New().String())
	usyncNode.Attributes.AddAttr("index", "0")
	usyncNode.Attributes.AddAttr("last", "true")
	usyncNode.Attributes.AddAttr("mode", "query")
	usyncNode.Attributes.AddAttr("context", "add")
	// query node
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Children.AddNode(newxxmp.EmptyNode("contact"))
	queryNode.Children.AddNode(newxxmp.EmptyNode("status"))
	// business node
	businessNode := newxxmp.EmptyNode("business")
	businessNode.Children.AddNode(newxxmp.EmptyNode("verified_name"))
	businessNode.Children.AddNode(newxxmp.EmptyNode("profile", newxxmp.NewAttribute("v", "116")))
	// set business node to  query node
	queryNode.Children.AddNode(businessNode)
	pictureNode := newxxmp.EmptyNode("picture")
	pictureNode.Attributes.AddAttr("type", "preview")
	queryNode.Children.AddNode(pictureNode)
	// set query node to usync node
	usyncNode.Children.AddNode(queryNode)
	// list node
	listNode := newxxmp.EmptyNode("list")
	for _, contact := range contacts {
		userNode := newxxmp.EmptyNode("user")
		userNode.Attributes.AddAttr("jid", NewJid(contact).Jid())
		listNode.Children.AddNode(userNode)
	}
	usyncNode.Children.AddNode(listNode)
	iqNode.Children.AddNode(usyncNode)
	i.Node = iqNode
	return i
}

//createIqUSyncInteractive -扫码后第二次同步
func createIqUSyncInteractive(id gtype.Int32, contacts []string) *IqNode {
	/*<iq
	    xmlns='usync' id='01f' type='get'>
	    <usync sid='sync_sid_delta_82ee3aff-81f0-41be-b65d-1566b9d118de' index='0' last='true' mode='delta' context='interactive'>
	        <query>
	            <contact/>
	            <status/>
	            <business>
	                <verified_name/>
	                <profile v='116'/>
	            </business>
	            <devices version='2'/>
	        </query>
	        <list>
	            <user>
	                <contact>+19048786157</contact>
	            </user>
	        </list>
	    </usync>
	</iq>*/
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// new empty node
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("xmlns", "usync")
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("type", "get")
	// usync node
	usyncNode := newxxmp.EmptyNode("usync")
	usyncNode.Attributes.AddAttr("sid", "sync_sid_delta_"+guuid.New().String())
	usyncNode.Attributes.AddAttr("index", "0")
	usyncNode.Attributes.AddAttr("last", "true")
	usyncNode.Attributes.AddAttr("mode", "delta")
	usyncNode.Attributes.AddAttr("context", "interactive")
	// query node
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Children.AddNode(newxxmp.EmptyNode("contact"))
	queryNode.Children.AddNode(newxxmp.EmptyNode("status"))
	// business node
	businessNode := newxxmp.EmptyNode("business")
	businessNode.Children.AddNode(newxxmp.EmptyNode("verified_name"))
	businessNode.Children.AddNode(newxxmp.EmptyNode("profile", newxxmp.NewAttribute("v", "116")))
	// set business node to  query node
	queryNode.Children.AddNode(businessNode)
	pictureNode := newxxmp.EmptyNode("devices")
	pictureNode.Attributes.AddAttr("version", "2")
	queryNode.Children.AddNode(pictureNode)
	// set query node to usync node
	usyncNode.Children.AddNode(queryNode)
	// list node
	listNode := newxxmp.EmptyNode("list")
	for _, contact := range contacts {
		userNode := newxxmp.EmptyNode("user")
		contactNode := newxxmp.EmptyNode("contact")
		phone := "+" + strings.ReplaceAll(contact, "@s.whatsapp.net", "")
		contactNode.SetData([]byte(phone))
		userNode.Children.AddNode(contactNode)
		listNode.Children.AddNode(userNode)
	}
	usyncNode.Children.AddNode(listNode)
	iqNode.Children.AddNode(usyncNode)
	i.Node = iqNode
	return i
}

// createIqUSyncSyncAddOneContacts  添加单个联系人
func createIqUSyncSyncAddOneContacts(id gtype.Int32, contacts []string) *IqNode {
	/*
			<iq
		    xmlns='usync' id='05' type='get'>
		    <usync sid='sync_sid_delta_f267ee49-d9ee-42c7-95c6-d39bb351a5a9' index='0' last='true' mode='delta' context='interactive'>
		        <query>
		            <contact/>
		            <status/>
		            <business>
		                <verified_name/>
		                <profile v='116'/>
		            </business>
		            <devices version='2'/>
		        </query>
		        <list>
		            <user>
		                <contact>+16815159081</contact>
		            </user>
		        </list>
		    </usync>
		</iq>
	*/
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// new empty node
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("xmlns", "usync")
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("type", "get")
	// usync node
	usyncNode := newxxmp.EmptyNode("usync")
	usyncNode.Attributes.AddAttr("sid", "sync_sid_delta_"+guuid.New().String())
	usyncNode.Attributes.AddAttr("index", "0")
	usyncNode.Attributes.AddAttr("last", "true")
	usyncNode.Attributes.AddAttr("mode", "delta")
	usyncNode.Attributes.AddAttr("context", "interactive")
	// query node
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Children.AddNode(newxxmp.EmptyNode("contact"))
	queryNode.Children.AddNode(newxxmp.EmptyNode("status"))
	// business node
	businessNode := newxxmp.EmptyNode("business")
	businessNode.Children.AddNode(newxxmp.EmptyNode("verified_name"))
	businessNode.Children.AddNode(newxxmp.EmptyNode("profile", newxxmp.NewAttribute("v", "116")))
	// set business node to  query node
	queryNode.Children.AddNode(businessNode)
	pictureNode := newxxmp.EmptyNode("devices")
	pictureNode.Attributes.AddAttr("version", "2")
	queryNode.Children.AddNode(pictureNode)
	// set query node to usync node
	usyncNode.Children.AddNode(queryNode)
	// list node
	listNode := newxxmp.EmptyNode("list")
	for _, contact := range contacts {
		userNode := newxxmp.EmptyNode("user")
		contactNode := newxxmp.EmptyNode("contact")
		phone := "+" + strings.ReplaceAll(contact, "@s.whatsapp.net", "")
		contactNode.SetData([]byte(phone))
		userNode.Children.AddNode(contactNode)
		listNode.Children.AddNode(userNode)
	}
	usyncNode.Children.AddNode(listNode)
	iqNode.Children.AddNode(usyncNode)
	i.Node = iqNode
	return i
}

// createIqUSyncSyncAddScanContacts
func createIqUSyncSyncAddScanContacts(id gtype.Int32, contacts []string) *IqNode {
	//<iq xmlns='usync' id='07' type='get'>
	//	<usync sid='sync_sid_query_d684e03c-5ed6-4489-8bff-1e20d0676059' index='0' last='true' mode='query' context='interactive'>
	//			<query><contact/><status/><business><verified_name/><profile v='116'/></business><picture type='preview'/></query>
	//			<list><user><contact>+19342074269</contact></user></list>
	//  </usync>
	//</iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// new empty node
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("xmlns", "usync")
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("type", "get")
	// usync node
	usyncNode := newxxmp.EmptyNode("usync")
	usyncNode.Attributes.AddAttr("sid", "sync_sid_delta_"+guuid.New().String())
	usyncNode.Attributes.AddAttr("index", "0")
	usyncNode.Attributes.AddAttr("last", "true")
	usyncNode.Attributes.AddAttr("mode", "query")
	usyncNode.Attributes.AddAttr("context", "interactive")
	// query node
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Children.AddNode(newxxmp.EmptyNode("contact"))
	queryNode.Children.AddNode(newxxmp.EmptyNode("status"))
	// business node
	businessNode := newxxmp.EmptyNode("business")
	businessNode.Children.AddNode(newxxmp.EmptyNode("verified_name"))
	businessNode.Children.AddNode(newxxmp.EmptyNode("profile", newxxmp.NewAttribute("v", "116")))
	// set business node to  query node
	queryNode.Children.AddNode(businessNode)
	pictureNode := newxxmp.EmptyNode("picture")
	pictureNode.Attributes.AddAttr("type", "preview")
	queryNode.Children.AddNode(pictureNode)
	// set query node to usync node
	usyncNode.Children.AddNode(queryNode)
	// list node
	listNode := newxxmp.EmptyNode("list")
	for _, contact := range contacts {
		userNode := newxxmp.EmptyNode("user")
		contactNode := newxxmp.EmptyNode("contact")
		phone := "+" + strings.ReplaceAll(contact, "@s.whatsapp.net", "")
		contactNode.SetData([]byte(phone))
		userNode.Children.AddNode(contactNode)
		listNode.Children.AddNode(userNode)
	}
	usyncNode.Children.AddNode(listNode)
	iqNode.Children.AddNode(usyncNode)
	i.Node = iqNode
	return i
}

// createIqAddGroup 邀请成员
func createIqAddGroup(id gtype.Int32, to string, participants ...string) *IqNode {
	/*
		<iq id="7" xmlns="w:g2" type="set" to="6283827948009-1618231574@g.us">
		    <add>
		        <participant jid="886933638868@s.whatsapp.net"/>
		    </add>
		</iq>
	*/

	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", (&JId{S: to}).GroupId())
	// add node
	addNode := newxxmp.EmptyNode("add")
	for _, participant := range participants {
		participantNode := newxxmp.EmptyNode("participant")
		participantNode.Attributes.AddAttr("jid", (&JId{S: participant}).Jid())
		addNode.Children.AddNode(participantNode)
	}
	iqNode.Children.AddNode(addNode)
	i.Node = iqNode
	return i
}

// createIqInvite 获取&请邀请链接
func createIqInvite(id gtype.Int32, to, code string) *IqNode {
	/*
		<iq id="02" xmlns="w:g2" type="get" to="6283827948009-1618227309@g.us">
		    <invite/>
		</iq>
	*/
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("to", to)
	inviteNode := newxxmp.EmptyNode("invite")
	if code != "" {
		//iq id='04' xmlns='w:g2' type='set' to='g.us'><invite code='IAlVfXsZJW13nKtyieWNQ3'/></iq>
		iqNode.Attributes.AddAttr("type", "set")
		inviteNode.Attributes.AddAttr("code", code)
	} else {
		iqNode.Attributes.AddAttr("type", "get")
	}
	// set invite node to iq node
	iqNode.Children.AddNode(inviteNode)
	i.Node = iqNode
	return i
}

// createIqWg2Query 获取群成员
func createIqWg2Query(id gtype.Int32, groupId string) *IqNode {
	/*
		<iq id="3" xmlns="w:g2" type="get" to="6283827948009-1617871241@g.us">
		    <query request="interactive"/>
		</iq>
	*/
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "get")
	iqNode.Attributes.AddAttr("to", groupId)
	// query node
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Attributes.AddAttr("request", "interactive")
	// set query node
	iqNode.Children.AddNode(queryNode)
	i.Node = iqNode
	return i
}

// createIqGetGroupCode 获取二维码
func createIqGetGroupCode(id gtype.Int32, groupId string) *IqNode {
	//<iq id='06' xmlns='w:g2' type='get' to='85366311809-1623558808@g.us'><invite/></iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "get")
	iqNode.Attributes.AddAttr("to", groupId)
	invite := newxxmp.EmptyNode("invite")
	iqNode.Children.AddNode(invite)
	i.Node = iqNode
	return i
}

// 设置群管理
func createIqSetGroupAdmin(id gtype.Int32, groupId, toWid string) *IqNode {
	//<iq id='6' xmlns='w:g2' type='set' to='85366311809-1623558808@g.us'>
	//	<promote><participant jid='8613538240895@s.whatsapp.net'/></promote>
	//</iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", groupId)
	promoteNode := newxxmp.EmptyNode("promote")
	participantNode := newxxmp.EmptyNode("participant")
	participantNode.Attributes.AddAttr("jid", toWid)
	promoteNode.Children.AddNode(participantNode)
	iqNode.Children.AddNode(promoteNode)
	i.Node = iqNode
	return i
}

//createBuildEncryptNode 发消息会用到
func createBuildEncryptNode(id gtype.Int32, u string) *IqNode {
	//<iq id='03' xmlns='encrypt' type='get' to='s.whatsapp.net'><key><user jid='601164346429.0:0@s.whatsapp.net'/></key></iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "encrypt")
	iqNode.Attributes.AddAttr("type", "get")
	iqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	// active node
	keyNode := newxxmp.EmptyNode("key")
	userNode := newxxmp.EmptyNode("user")
	jid := strings.ReplaceAll(u, "@s.whatsapp.net", ".0:0@s.whatsapp.net")
	userNode.Attributes.AddAttr("jid", jid)
	keyNode.Children.AddNode(userNode)
	iqNode.Children.AddNode(keyNode)
	i.Node = iqNode
	return i
}

//取消群管理
func createIqDemoteGroupAdmin(id gtype.Int32, groupId, toWid string) *IqNode {
	//<iq id='5' xmlns='w:g2' type='set' to='85366311809-1623558808@g.us'>
	//	<demote><participant jid='8613538240895@s.whatsapp.net'/></demote>
	//	</iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", groupId)
	promoteNode := newxxmp.EmptyNode("demote")
	participantNode := newxxmp.EmptyNode("participant")
	participantNode.Attributes.AddAttr("jid", toWid)
	promoteNode.Children.AddNode(participantNode)
	iqNode.Children.AddNode(promoteNode)
	i.Node = iqNode
	return i
}

//退出群组
func createIqLeaveGroup(id gtype.Int32, groupId string) *IqNode {
	//<iq id='5' xmlns='w:g2' type='set' to='g.us'><leave><group id='85366311809-1623558808@g.us'/></leave></iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", "g.us")
	leaveNode := newxxmp.EmptyNode("leave")
	groupNode := newxxmp.EmptyNode("group")
	groupNode.Attributes.AddAttr("id", groupId)
	leaveNode.Children.AddNode(groupNode)
	iqNode.Children.AddNode(leaveNode)
	i.Node = iqNode
	return i
}

//createPresencesSubscribeNew
func createPresencesSubscribeNew(id gtype.Int32, u string) *PresenceNode {
	//<presence type="subscribe" to="xxxxx@s.whatsapp.net"/>
	p := &PresenceNode{id: u, BaseNode: NewBaseNode()}
	// create node
	n := &newxxmp.Node{
		Tag: NodePresence,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("type", "subscribe"),
			newxxmp.NewAttribute("to", u),
		},
	}
	p.Node = n
	return p
}

//createIqGroupDesc 设置群组描述
func createIqGroupDesc(id gtype.Int32, groupId string, desc string) *IqNode {
	//<iq id='06' xmlns='w:g2' type='set' to='85366311809-1624079613@g.us'>
	//<description id='2D012316694A0684A4C75588F8B7C8F0'><body>1111111111\U0001f600</body></description></iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", groupId)
	descriptionNode := newxxmp.EmptyNode("description")
	descriptionNode.Attributes.AddAttr("id", "")
	//descriptionNode
	bodyNode := newxxmp.EmptyNode("body")
	bodyNode.SetData([]byte(desc))
	descriptionNode.Children.AddNode(bodyNode)
	iqNode.Children.AddNode(descriptionNode)
	i.Node = iqNode
	return i
}

// createIqGroup 创建群聊
func createIqGroup(id gtype.Int32, u, subject string, participants []string) *IqNode {
	/*
		<iq xmlns="w:g2" id="01" type="set" to="@g.us">
		    <create subject="Hu" key="6283827948009-bc310b899631417283e97452f1076ec6@temp">
		        <participant S="886926932491@s.whatsapp.net"/>
		        <participant S="886955754531@s.whatsapp.net"/>
		        <participant S="84878185730@s.whatsapp.net"/>
		    </create>
		</iq>
	*/
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	key := fmt.Sprintf("%s-%s@temp", u, strings.ReplaceAll(guuid.New().String(), "-", ""))
	log.Println("createIqGroup key:", key)
	// participants
	participantsNode := make([]*newxxmp.Node, 0)
	for _, participant := range participants {
		if !strings.Contains(participant, "@s.whatsapp.net") {
			participant = participant + "@s.whatsapp.net"
		}
		// create participants node
		pNode := &newxxmp.Node{
			Tag: "participant",
			Attributes: []newxxmp.Attribute{
				newxxmp.NewAttribute("jid", participant),
			},
		}
		participantsNode = append(participantsNode, pNode)
	}
	// create iq
	n := &newxxmp.Node{
		Tag: NodeIq,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("xmlns", "w:g2"),
			newxxmp.NewAttribute("id", id.String()),
			newxxmp.NewAttribute("type", "set"),
			newxxmp.NewAttribute("to", "@g.us"),
		},
		Children: []*newxxmp.Node{
			{
				Tag: "create",
				Attributes: []newxxmp.Attribute{
					newxxmp.NewAttribute("subject", subject),
					newxxmp.NewAttribute("key", key),
				},
				Children: participantsNode,
			},
		},
	}
	i.Node = n
	return i
}

// createIqGroupMember 获取群成员
func createIqGroupMember(id gtype.Int32, u string) *IqNode {
	//<iq id='2' xmlns='w:g2' type='get' to='919999904379-1624957782@g.us'><query request='interactive'/></iq>
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:g2")
	iqNode.Attributes.AddAttr("to", u)
	iqNode.Attributes.AddAttr("type", "get")
	queryNode := newxxmp.EmptyNode("query")
	queryNode.Attributes.AddAttr("request", "interactive")
	iqNode.Children.AddNode(queryNode)
	i.Node = iqNode
	return i
}

// createIqSetProfilePicture 设置用户头像
func createIqSetProfilePicture(id gtype.Int32, pictureData []byte, to string) *IqNode {
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:profile:picture")
	iqNode.Attributes.AddAttr("to", to)
	iqNode.Attributes.AddAttr("type", "set")
	// picture node
	pictureNode := newxxmp.EmptyNode("picture")
	pictureNode.Attributes.AddAttr("type", "image")
	pictureNode.SetData(pictureData)
	// set picture node to iq node
	iqNode.Children.AddNode(pictureNode)

	i.Node = iqNode
	return i
}

// 获取cdn
func createIqMediaCon(id gtype.Int32) *IqNode {
	// default promise 超时100秒
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:m")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	mediaNode := newxxmp.EmptyNode("media_conn")
	iqNode.Children.AddNode(mediaNode)
	i.Node = iqNode
	return i
}

// createIqGetQr 获取二维码
func createIqGetQr(id gtype.Int32) *IqNode {
	//<iq id='02' xmlns='w:qr' type='set'><qr type='contact' action='get'/></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:qr")
	iqNode.Attributes.AddAttr("type", "set")
	qr := newxxmp.EmptyNode("qr")
	qr.Attributes.AddAttr("type", "contact")
	qr.Attributes.AddAttr("action", "get")
	iqNode.Children.AddNode(qr)
	i.Node = iqNode
	return i
}

// createIqRevokeQr 重置二维码
func createIqRevokeQr(id gtype.Int32) *IqNode {
	//<iq id='05' xmlns='w:qr' type='set'><qr type='contact' action='revoke'/></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:qr")
	iqNode.Attributes.AddAttr("type", "set")
	qr := newxxmp.EmptyNode("qr")
	qr.Attributes.AddAttr("type", "contact")
	qr.Attributes.AddAttr("action", "revoke")
	iqNode.Children.AddNode(qr)
	i.Node = iqNode
	return i
}

// createIqScanCode 扫码二维码
func createIqScanCode(id gtype.Int32, code string, opCode int32) *IqNode {
	//<iq id='03' xmlns='w:qr' type='get'><qr code='3OLFTKU2ZTIYN1'/></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	if opCode == 0 {
		// iq node
		iqNode := newxxmp.EmptyNode(NodeIq)
		iqNode.Attributes.AddAttr("id", id.String())
		iqNode.Attributes.AddAttr("xmlns", "w:qr")
		iqNode.Attributes.AddAttr("type", "get")
		qr := newxxmp.EmptyNode("qr")
		qr.Attributes.AddAttr("code", code)
		iqNode.Children.AddNode(qr)
		i.Node = iqNode
		return i
	} else {
		//<iq id='02' xmlns='w:g2' type='get' to='g.us'><invite code='IAlVfXsZJW13nKtyieWNQ3'/></iq>
		iqNode := newxxmp.EmptyNode(NodeIq)
		iqNode.Attributes.AddAttr("id", id.String())
		iqNode.Attributes.AddAttr("xmlns", "w:g2")
		iqNode.Attributes.AddAttr("type", "get")
		iqNode.Attributes.AddAttr("to", "g.us")
		inviteNode := newxxmp.EmptyNode("invite")
		inviteNode.Attributes.AddAttr("code", code)
		iqNode.Children.AddNode(inviteNode)
		i.Node = iqNode
		return i
	}
}

//createIqNickName
func createIqNickName(id gtype.Int32, name string) *IqNode {
	//<presence type='available' name='000'/>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	presenceNode := newxxmp.EmptyNode("presence")
	presenceNode.Attributes.AddAttr("type", "available")
	presenceNode.Attributes.AddAttr("name", name)
	i.Node = presenceNode
	return i
}

// createIqGetPicture 获取头像
func createIqGetPicture(id gtype.Int32, u string) *IqNode {
	/*
			<iq id="02" xmlns="w:profile:picture" to="886933638868@s.whatsapp.net" type="get">
		    	<picture query="url" type="image"/>
			</iq>

			result
			<iq from="886933638868@s.whatsapp.net" type="result" id="02">
		   	 	<picture id="1445084059" type="image" url="https://pps.whatsapp.net/v/t61.24694-24/55964022_277198593191373_3871628458580770816_n.jpg?ccb=11-4&oh=1e13b946655722e0111f419988ebed93&oe=608CA872" direct_path="/v/t61.24694-24/55964022_277198593191373_3871628458580770816_n.jpg?ccb=11-4&oh=1e13b946655722e0111f419988ebed93&oe=608CA872"/>
			</iq>
	*/

	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:profile:picture")
	iqNode.Attributes.AddAttr("to", u)
	iqNode.Attributes.AddAttr("type", "get")

	pictureNode := newxxmp.EmptyNode("picture")
	pictureNode.Attributes.AddAttr("query", "url")
	pictureNode.Attributes.AddAttr("type", "image")
	// set picture node to iq node
	iqNode.Children.AddNode(pictureNode)

	i.Node = iqNode
	return i
}

// createIqGetPreview 获取头像
func createIqGetPreview(id gtype.Int32, u string) *IqNode {
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "w:profile:picture")
	iqNode.Attributes.AddAttr("to", u)
	iqNode.Attributes.AddAttr("type", "get")
	pictureNode := newxxmp.EmptyNode("picture")
	pictureNode.Attributes.AddAttr("type", "preview")
	// set picture node to iq node
	iqNode.Children.AddNode(pictureNode)
	i.Node = iqNode
	return i
}

//上传个人签名
func createIqState(id gtype.Int32, contact string) *IqNode {
	//<iq id='?' xmlns='status' type='set' to='s.whatsapp.net'><status>"+context+"</status></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "status")
	iqNode.Attributes.AddAttr("type", "set")
	iqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	statusNode := newxxmp.EmptyNode("status")
	statusNode.SetData([]byte(contact))
	iqNode.Children.AddNode(statusNode)
	i.Node = iqNode
	return i
}

//获取个性签名
func createIqGetState(id gtype.Int32, u string) *IqNode {
	//<iq id='00' xmlns='status' type='get' to='s.whatsapp.net'><status><user jid='85366311809@s.whatsapp.net'/></status></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "status")
	iqNode.Attributes.AddAttr("type", "get")
	iqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	statusNode := newxxmp.EmptyNode("status")
	userNode := newxxmp.EmptyNode("user")
	userNode.Attributes.AddAttr("jid", u)
	statusNode.Children.AddNode(userNode)
	iqNode.Children.AddNode(statusNode)
	i.Node = iqNode
	return i
}

// createIq2Fa 两步验证
func createIq2Fa(id gtype.Int32, code, email string) *IqNode {
	//<iq to='s.whatsapp.net' id='5' xmlns='urn:xmpp:whatsapp:account' type='set'><2fa><code>671013</code><email></email></2fa></iq>
	i := &IqNode{id: id.Val(), promise: promise.New(nil)}
	// iq node
	iqNode := newxxmp.EmptyNode(NodeIq)
	iqNode.Attributes.AddAttr("to", "s.whatsapp.net")
	iqNode.Attributes.AddAttr("id", id.String())
	iqNode.Attributes.AddAttr("xmlns", "urn:xmpp:whatsapp:account")
	iqNode.Attributes.AddAttr("type", "set")

	fa2Node := newxxmp.EmptyNode("2fa")
	// code node
	codeNode := newxxmp.EmptyNode("code", []byte(code))
	fa2Node.Children.AddNode(codeNode)
	// email node
	emailNode := newxxmp.EmptyNode("email")
	if email != "" {
		emailNode = newxxmp.EmptyNode("email", []byte(email))
	}
	fa2Node.Children.AddNode(emailNode)
	iqNode.Children.AddNode(fa2Node)
	i.Node = iqNode
	return i
}

func NewIqProcessor() *IqProcessor {
	return &IqProcessor{
		_iqId:     gtype.NewInt32(0),
		_nodeList: gmap.NewIntAnyMap(true),
	}
}

// IqProcessor
type IqProcessor struct {
	_iqId     *gtype.Int32
	_nodeList *gmap.IntAnyMap
}

// reset
func (i *IqProcessor) reset() {
	//i._iqId.Set(0)
	i._nodeList.Clear()
}

// iqId
func (i *IqProcessor) iqId() gtype.Int32 {
	defer i._iqId.Add(1)
	return *i._iqId
}

// GetResult 可以重新定义超时时间 如果设置等待时间，回调函数将失效
func (i *IqProcessor) GetResult(id int, waitTime ...time.Duration) (promise.Any, error) {
	var _waitTime time.Duration
	value := i._nodeList.Get(id)
	if iPromise, ok := value.(_interface.IPromise); ok {
		if len(waitTime) > 0 {
			_waitTime = waitTime[0]
			t := gtype.NewInt32(int32(id))
			iPromise.SetPromise(i.catchTimeOut(*t, _waitTime))
			return iPromise.GetResult()
		} else {
			return iPromise.GetResult()
		}
	}
	return nil, IdNotExistError
}

// SaveNode
func (i *IqProcessor) SaveNode(id gtype.Int32, any interface{}) {
	i._nodeList.Set(int(id.Val()), any)
}

func (i *IqProcessor) SetNodeTimeOutRemove(
	id gtype.Int32, node interface{}, interval time.Duration, callBack func()) {
	if interval == 0 {
		// default 10 second
		interval = time.Second * 10
	}
	// add to list
	i._nodeList.Set(int(id.Val()), node)
	// add timer
	gtimer.AddOnce(interval, func() {
		if i.RemoveNode(int(id.Val())) != nil {
			// 移除成功回调
			callBack()
		}
	})
}

// RemoveNode
func (i *IqProcessor) RemoveNode(id int) interface{} {
	return i._nodeList.Remove(id)
}

// An error occurred to remove the node from the list
// 使用默认的promise 前提是promise被初始化 一旦发送错误就从列表删除节点
// 建议使用这个
func (i *IqProcessor) catchRemoveNode(promise *promise.Promise, id gtype.Int32) *promise.Promise {
	catch := promise.Catch(func(err error) error {
		if i.RemoveNode(int(id.Val())) == nil {
			return nil
		}
		log.Println("catchRemoveNode error", err, "remove node id->", id.String())
		return errors.New(err.Error() + "id -> " + id.String())
	})

	promise = catch

	return catch
}

// catchTimeOut
//当超时,从列表移除节点 和上面 的catchRemoveNode 不同 catchTimeOut 创建了个新的 context timeout
// 可以重新定义超时时间
func (i *IqProcessor) catchTimeOut(id gtype.Int32, duration time.Duration) *promise.Promise {
	return promise.New(func(resolve func(promise.Any), reject func(error)) {
		timeout, _ := context.WithTimeout(context.Background(), duration)
		<-timeout.Done()
		if i.RemoveNode(int(id.Val())) == nil {
			return
		}
		log.Println("p2 catchTimeOut error", "remove node id->", id.String())
		reject(errors.New("run time our!" + " id -> " + id.String()))
	}).Catch(func(err error) error {
		// time out remove node

		return err
	})
}

// Handle 处理 Tag Iq相关 Node
func (i *IqProcessor) Handle(node *newxxmp.Node) error {
	if node == nil {
		// TODO 返回错误吗
		return nil
	}
	if node.GetTag() != NodeIq {
		//TODO 不属于这个处理器的
		return errors.New("")
	}
	idNode := node.GetAttribute("id")
	if idNode == nil {
		return errors.New("id is nil")
	}
	iqIdStr := idNode.Value()
	if iqIdStr == "" {
		return errors.New("iqIdStr is null")
	}
	iqId, err := strconv.Atoi(iqIdStr)
	if err != nil {
		return err
	}
	// 尝试从列表中删除
	item := i._nodeList.Remove(iqId)
	if item != nil {
		if iqNode, ok := item.(*IqNode); ok {
			iqNode.Process(node)
		}
	} else {
		//TODO 处理其他不存在列表的
	}
	return nil
}

// BuilderIqUserKeys
func (i *IqProcessor) BuilderIqUserKeys(u []string, reason bool) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = crateIqUserKeys(u, iqId, reason)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	//// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*5, func() {
		build.promise.Reject(fmt.Errorf("iq user keys time out id : %d", iqId.Val()))
	})
	return build
}

//BuilderIqConfig send xmlns for urn:xmpp:whatsapp:push
func (i *IqProcessor) BuilderIqConfig() (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqConfig(iqId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq usync contact time out id:%d", iqId.Val()))
	})
	return build
}

// BuilderIqConfigOne
func (i *IqProcessor) BuilderIqConfigOne() (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqConfigOne(iqId)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuilderIqConfigOne time out id:%d", iqId.Val()))
	})
	return build
}

// BuilderIqConfigTwo
func (i *IqProcessor) BuilderIqConfigTwo() (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqConfigTwo(iqId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq usync contact time out id:%d", iqId.Val()))
	})
	return build
}

//BuilderIqPing
func (i *IqProcessor) BuilderIqPing() (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqPing(iqId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq usync contact time out id:%d", iqId.Val()))
	})
	return build
}

//BuilderGetVerifiedName
func (i *IqProcessor) BuilderGetVerifiedName(jid string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createGetVerifiedName(iqId, jid)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuilderGetVerifiedName time out id:%d", iqId.Val()))
	})
	return build
}

//BuilderSendCategories
func (i *IqProcessor) BuilderSendCategories() (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createSendCategories(iqId)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq  BuilderSendCategories time out id:%d", iqId.Val()))
	})
	return build
}

//BuilderBusinessProfile
func (i *IqProcessor) BuilderBusinessProfile(categoryId string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createBusinessProfile(iqId, categoryId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuilderBusinessProfile time out id:%d", iqId.Val()))
	})
	return build
}

//BuilderBusinessProfileTow
func (i *IqProcessor) BuilderBusinessProfileTow(u string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createBusinessProfileTow(iqId, NewJid(u).Jid())
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*10))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq usync contact time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqCreateGroup
func (i *IqProcessor) BuildIqCreateGroup(u, subject string, participants []string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqGroup(iqId, u, subject, participants)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateGroup time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqCreateGroupMember
func (i *IqProcessor) BuildIqCreateGroupMember(u string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqGroupMember(iqId, u)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateGroupMember time out id:%d", iqId.Val()))
	})
	return build
}

//BuildIqGetGroupCode
func (i *IqProcessor) BuildIqGetGroupCode(u, groupId string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqGetGroupCode(iqId, groupId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateGroupMember time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqSetGroupAdmin
func (i *IqProcessor) BuildIqSetGroupAdmin(u, groupId, toWid string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqSetGroupAdmin(iqId, groupId, toWid)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqSetGroupAdmin time out id:%d", iqId.Val()))
	})
	return build
}

//BuildEncrypt
func (i *IqProcessor) BuildEncrypt(u string) (build *IqNode) {
	iqId := i.iqId()
	subscribeNode := createBuildEncryptNode(iqId, u)
	return subscribeNode
}

// BuildIqSetGroupAdmin
func (i *IqProcessor) BuildIqSetDemoteGroupAdmin(u, groupId, toWid string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqDemoteGroupAdmin(iqId, groupId, toWid)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqSetGroupAdmin time out id:%d", iqId.Val()))
	})
	return build
}

//BuildIqLogOutGroup
func (i *IqProcessor) BuildIqLogOutGroup(u, groupId string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqLeaveGroup(iqId, groupId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqLogOutGroup time out id:%d", iqId.Val()))
	})
	return build
}

// BuildPresencesSubscribeNew
func (i *IqProcessor) BuildPresencesSubscribeNew(u string) (build *PresenceNode) {
	iqId := i.iqId()
	// create
	build = createPresencesSubscribeNew(iqId, u)
	build.SetPromise(i.catchTimeOut(iqId, time.Second*5))
	// save node to list
	i.SaveNode(iqId, build)
	return build
}

//BuildIqGroupDesc 设置群描述
func (i *IqProcessor) BuildIqGroupDesc(u, groupId string, desc string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqGroupDesc(iqId, groupId, desc)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqLogOutGroup time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqWg2Query 获取群成员
func (i *IqProcessor) BuildIqWg2Query(groupId string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqWg2Query(iqId, groupId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqWg2Query time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqAddGroup 邀请群成员
func (i *IqProcessor) BuildIqAddGroup(groupId string, participants ...string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqAddGroup(iqId, groupId, participants...)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqWg2Query time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqUSyncContact 同步联系人
func (i *IqProcessor) BuildIqUSyncContact(contacts []string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqUSync(iqId, contacts)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqWg2Query time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqUSyncContactAdd
func (i *IqProcessor) BuildIqUSyncContactAdd(contacts []string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqUSyncAdd(iqId, contacts)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq usync contact time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqUSyncContactInteractive
func (i *IqProcessor) BuildIqUSyncContactInteractive(contacts []string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqUSyncInteractive(iqId, contacts)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqWg2Query time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqUSyncSyncAddOneContacts
func (i *IqProcessor) BuildIqUSyncSyncAddOneContacts(contacts []string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqUSyncSyncAddOneContacts(iqId, contacts)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqUSyncSyncAddOneContacts time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqUSyncSyncAddScanContacts
func (i *IqProcessor) BuildIqUSyncSyncAddScanContacts(contacts []string) (build *IqNode) {
	iqId := i.iqId()
	// create
	build = createIqUSyncSyncAddScanContacts(iqId, contacts)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqWg2Query time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqActive
func (i IqProcessor) BuildIqActive() (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqActive(iqId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqActive time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqIqSetEncryptKeys 上传812 对
func (i IqProcessor) BuildIqIqSetEncryptKeys(preKeys []*record.PreKey, skey *record.SignedPreKey, identityKey identity.Key, regId uint32) *IqNode {
	iqId := i.iqId()
	keys := createIqSetEncryptKeys(iqId, preKeys, skey, identityKey, regId)
	//keys.d,_ = hex.DecodeString("00f80a1104491ba6052d07fa0003f805f8029bfc207ddee872c8270012f1aa280458f1a85eb20c794e1284771781a7e8a13c43953ff8029afc045a586b5af80205fc0105f8025cf9032cf8028df802f80204fc03693b51f80228fc20ef6e02220b53f6d7c3d746810fae4929d94bd2819b7037ee9a94b02891dbfa4af8028df802f80204fc03693b46f80228fc20bf42a133f24ba17462e039deac26391e3e06e02f8e9b1881fa3f5742fd43a952f8028df802f80204fc03693b5cf80228fc20b8aa31ab6731e308112fecda2eacc52ba9b0b915f5dd2da282bed675267fac45f8028df802f80204fc03693b4af80228fc20edfe1d0156f6b30432c14b9828a7d7a227d19f76b2e72f968b72a41e44a7d724f8028df802f80204fc03693b48f80228fc208800af15c522f06845a06ddd6829975ea084b5464399d19cde7282fe2eed8958f8028df802f80204fc03693b45f80228fc2097fb514c341d3627adb01afedfba402f93e942b2047d9b106bac3280af04853ef8028df802f80204fc03693b36f80228fc206a2d19612a67015aee729caf6ad7cfde1a137bf1ec0e2925458d478322fae23df8028df802f80204fc03693b5ef80228fc2071a3be4c45e47926a867a1094617d8258c90db1379a9eaebe27556609c97c547f8028df802f80204fc03693b35f80228fc20ab95b34b3b8d4c1bd5577adfd8ea452df42a074fbdf96c66197ee8d6571ffb6af8028df802f80204fc03693b33f80228fc2039d06007f4ec5f1c24e77708b72f92085c4e238c58a8b96eb8d66bc73f77454cf8028df802f80204fc03693b63f80228fc20dce0db8470bf44565f1592afea095c122f6997e0e8bc39ac9f0f6080ba88e558f8028df802f80204fc03693b53f80228fc20b595e8497574c8ba8d626c0c15a5d68ec264336246f84f774acd05c6e2f13418f8028df802f80204fc03693b4df80228fc20280522d8f5e0b87926701ca51278e2b0a8996322316e80fd69f944db91d04949f8028df802f80204fc03693b47f80228fc201317cc3f455a87403a9478266ce24a216a144a2766a60ccc97f4d9cc4749194ef8028df802f80204fc03693b39f80228fc205d6fb2d18a14f3baa54dac9c9a563736b818b915545f1a626dbd42f013d5fe48f8028df802f80204fc03693b5bf80228fc20acf9ae57936ce4856276c01372c8f403ed1bbc915736ad357994a44227b53d0df8028df802f80204fc03693b4bf80228fc20607d48ec02f4e3d82c184dbdf5883654d31b8e1806b6567e56812df4dd478b27f8028df802f80204fc03693b37f80228fc20110d5e29c2669338f083ee5364b6a585551e166590c0fe5928a9602bed2a8326f8028df802f80204fc03693b5af80228fc20530fa767c23f1297f56614e7dac7e8c81e328adfc1068a3229a3b9d7fcb75c06f8028df802f80204fc03693b55f80228fc20601c8702212df02e89864e36cfbfb23d20cb677faf48eb6ab157832a1900be5cf8028df802f80204fc03693b41f80228fc20a5678c9994b279cf2f588160c38b5d217de8b6479bc3b7954f708a54a1d70c11f8028df802f80204fc03693b49f80228fc20d2a3bb39f48b5da15a032b544fb80487534ac6e6d94a32b6675220a8ef587b5bf8028df802f80204fc03693b43f80228fc207847bde0b087c23e8927ae46a1aa5a7f852d406bd91390c02ed34cc51d48b001f8028df802f80204fc03693b3bf80228fc203f500dd1db8987b030bd591b9fc901353b779d23e27b96ccdd4c80b347d98544f8028df802f80204fc03693b52f80228fc203e721a74c3b0954789aecbb56579efe1a3b6ce7317ff83c9e3662605bbbf897ff8028df802f80204fc03693b5df80228fc20f96b9fe2f12461e3e871acb9788d6eafbe7ffb2f913c198e82c004e56619e73cf8028df802f80204fc03693b38f80228fc20879aa840a1e62e0d076d2f9862823bf459facfc121e47436f736b18dff3fc634f8028df802f80204fc03693b61f80228fc2083d99a3b1cfca0d1272ef195755bb23ecd3854a2855ca38e313fbec78fa7dd08f8028df802f80204fc03693b34f80228fc20eb81bd59d42234ff61318904369f5de575abec6d80df2a2d7a0131076d90da7af8028df802f80204fc03693b57f80228fc20b0f9c2d3107c59b168da58e4f4fce19871f725665711ea55c6706e5e417af556f8028df802f80204fc03693b58f80228fc20c76632dabc5145f3b83922c758af9114967518e1ce13128efa621b26d58df52ef8028df802f80204fc03693b3cf80228fc20f3af5df1b4585373d4b59784a2e04acc36b5472705ab2355efd33b79177b5c5df8028df802f80204fc03693b60f80228fc201371914851abbc7471a6fb666cec0ec144e8027c22f27198c687eb67fe8e7410f8028df802f80204fc03693b62f80228fc208e59b2bd9dc7523e37dedeef65fe2489b54ff535654e8b77a408f838c85a7734f8028df802f80204fc03693b64f80228fc204c82dfae9730e9bc06fba77c3b6d6fd8337ecbf65964e7dc964770ae5b281e0bf8028df802f80204fc03693b4ff80228fc20e44fe30b43735b5b75c89c41eceffbcb80fb7c128337df24f38f188c9dd16902f8028df802f80204fc03693b3af80228fc209fc88ca8e5e72b5db7959a213061b2c78e8101accedeadec4a8cabfb56884366f8028df802f80204fc03693b5ff80228fc20e827f8348c8a66eca1535bfb0fe522c5f9cb195136602b4208b0a3becbacdc6bf8028df802f80204fc03693b59f80228fc2085ea7baef71f68ec493fd0fc3bf39ce62d1c960112d58c1b963919121b68b238f8028df802f80204fc03693b54f80228fc204991919710726c719d8b334707ce51abfafc7a7f008377d854b361e3c8eed249f8028df802f80204fc03693b4cf80228fc209db267668fcdf42de8e63530ea9b705b8b0f58272e864c690054c3ffaba8f93ef8028df802f80204fc03693b44f80228fc20e8041a91988963087a004a0016fe850c383498516a136ddd548d03e4e9ed9612f8028df802f80204fc03693b3ff80228fc20031138bae197ffe94757c6384377fa2125a001b361ebb23dd5816d630912886df8028df802f80204fc03693b4ef80228fc20ab1f0917aa36658c3d70dad6be3d25dc67562272c351ba7198586bba2a2d3935f8028df802f80204fc03693b56f80228fc2045f08674264eefaf7efe7bacbb22284a6c41a70f296605a18ce7055482a74766f8028df802f80204fc03693b40f80228fc209c0d81d628025e489fe62e1b57b9b8c88fd70813eb6b976658d848fa6d335f44f8028df802f80204fc03693b50f80228fc2003b7ed0a1bb82f986960b68c727aff23a9e499ede64dcd33e20baca57f575a53f8028df802f80204fc03693b3ef80228fc20d0a6fd0a0eab4004944ce54fda7558d74e6612d0737adf371eef25a367f75940f8028df802f80204fc03693b3df80228fc2095ab7fb6e51d927335941a1f9f513699587f9e618a40ad97cd2ef2021465b316f8028df802f80204fc03693b42f80228fc20db9a7d0230895f8853f911cf3181c4147ec915bcbb11584f7bcb7792a3c3ed7df8028df802f80204fc03693b83f80228fc2077c78138065e0b4be5626366bfe6a61e0dce606cdc6b426f6a85741fcc792c28f8028df802f80204fc03693b6df80228fc207e514df2be1df515262f99b1d331400f9196ec9d15e41efa793a605db6b24a08f8028df802f80204fc03693b8ef80228fc20f5cb070f8ed73f09782cbbe7d9dbc772cea889a70063a3b5ad2e515b0b5b8107f8028df802f80204fc03693b80f80228fc204fbb3a77c12ecd8b3a538b939369d8a3696fd775be2f03b03047e3d24c0d1e75f8028df802f80204fc03693b97f80228fc203d97684c3388eacc6d67aaa052a407ee6f85dfb62642a327328d05ba14590049f8028df802f80204fc03693b7ff80228fc205b1ae12f4afa42fff8952955eff72155be6848d9d5d1f32430bde76cc19a222cf8028df802f80204fc03693b6ff80228fc2076b933454b13ea75dba5b761a2efc6ccf376cb05d83a8561e90439b2d450d471f8028df802f80204fc03693b7bf80228fc208d65d8b083c44bce9daf2bdaf5c89711816285e73acac2ee457b1887de49d20cf8028df802f80204fc03693b96f80228fc20ef426ddca99d6a320d8c31d74c9bbd72cd5beef3fecd236a4c101cb37cb7220df8028df802f80204fc03693b73f80228fc20ec24dace6e5364738fa51bff9ec6b3389df5be2c64ae90e10adf0c79ee2c531bf8028df802f80204fc03693b92f80228fc20ca75f8db9d9f45df5b367939ed04d6c3e7aae71c50da01329999b3ff14319e09f8028df802f80204fc03693b71f80228fc2021393e4618f82f53002ddad2d7d5724345d5cec5602829de1305c9b3d9347612f8028df802f80204fc03693b66f80228fc203c3bf6b82f06c7c4ab21dec664d7d860747385905973d923de9e7dabf0d7975bf8028df802f80204fc03693b79f80228fc2067df96864558075061cf35c6874d6b7d93d6f04eb45561684bdc39598afac036f8028df802f80204fc03693b6cf80228fc203e1c1d4b8b361d50331357cadab0ed52cc678c0244fcdb3657971cb2f5d65735f8028df802f80204fc03693b90f80228fc207fe2b39c59eb57b5901af8228e90c789e2592eafe0d325aeb01be9e12cedfa29f8028df802f80204fc03693b8af80228fc201481421d59945b0056075362022393ece02de623d4de25eb17f32b1172200125f8028df802f80204fc03693b70f80228fc20a35d62ce89017c3f5f27bcfea6204c3d8b3501eb4c6e8b9e00fe942a9954413bf8028df802f80204fc03693b86f80228fc203afdb1bca1827d9fe12fc37766be1e5842582bdfc037dce1a7370e95121a4157f8028df802f80204fc03693b69f80228fc203c395d98363840faac35f19b2bce729d9b6c3e145afe03e32fb0a363460a8948f8028df802f80204fc03693b8ff80228fc20fbddda38193b2245ab8484a75c382ed2586c9031f09733832e7e7e6709fd491bf8028df802f80204fc03693b85f80228fc20fbca16b5bd1b30ac41147ef99e0026973737a2bccbe61e72027be6bcaf516668f8028df802f80204fc03693b7ef80228fc20bd4e8e85d9672d8c9f040a5d0d459fb4d0decc846ec9bab266a672073d626a03f8028df802f80204fc03693b94f80228fc20f6a7a119626eb5570751425cf5b1d4f4d94eb58c3a4cdd3965cdcc9eae4d545af8028df802f80204fc03693b7af80228fc205e3076006f3368d881ce76def7e18d1e48a3f99d55d937b29a8e4201fd70313bf8028df802f80204fc03693b72f80228fc2031cd1ad2475753876c14642755ceced9ebabd9e56b32ccb5c8ef8e0411225729f8028df802f80204fc03693b74f80228fc207a8d5e6bbbefc6fd25b911dee86dba58359947d8c914ec86b0552225b1c6b76af8028df802f80204fc03693b7df80228fc2021ee9a0697f3594c0eb7a9678a991c728afc42feade0b101581e403e12e0ad61f8028df802f80204fc03693b87f80228fc20630805838d9279ea977ca92422824307311adcc15007dd28e28cfcf24e90135bf8028df802f80204fc03693b7cf80228fc20a7bb6b17f976d1678ab0376cb97ddf9fba1d29a3cb39aa26d2c1c5517e258e16f8028df802f80204fc03693b6ef80228fc20d934d3cc3bb5bdeb2760d7b30d1016060c2984b23450d70f60d11b57732fd133f8028df802f80204fc03693b84f80228fc20fce1a6636763e023201b34cd4ff85a6e9e1892afc11570ad36eef80c12cc731ff8028df802f80204fc03693b82f80228fc20b29824f48a43beec00da100be36356fc42f8743a63352e4e87afb9beccf85f65f8028df802f80204fc03693b93f80228fc20fb0e2663b7d2c9868c66b0e577ce5616d7133778dfadce0a233bad26b4b2ac77f8028df802f80204fc03693b89f80228fc20d46a991b3c65465c3dfc70325d20a647e9c322e240633f64a9675b7489769f3bf8028df802f80204fc03693b75f80228fc20817432f67a6b9a1129e95d8f25dd76bb77d50f7585ad15f487ba887e90e65379f8028df802f80204fc03693b88f80228fc208fa1c7435633b9f761da92510fbec1d05c363b7d909d28a21009ba90d4528b08f8028df802f80204fc03693b95f80228fc200328575c76d1b6e150b6faa3617fd28c2487ee4728582bec50aa0828699a015bf8028df802f80204fc03693b68f80228fc2035a6d56efa86b0bd8253aa8a36301de70141898192ecf57644250cd147b8f327f8028df802f80204fc03693b77f80228fc20ba522a1dc984aae5c1e32aa556c47151329745bceba00da40e7e90f81f2c9f42f8028df802f80204fc03693b8cf80228fc2030ea3e0b338f9bcd479de644fcf5c45a94fc32f28fdcf167ed5fab9b9d17124ef8028df802f80204fc03693b81f80228fc20f9855b30ad74448a7c43c8710f1ae3525a85e67114c00f35cd9d9c648f9f8e6ef8028df802f80204fc03693b6af80228fc2036082cfe54ccbee2ed4432d7b17f71da702ca70b527c0f74e3734ec33a310507f8028df802f80204fc03693b78f80228fc20158077c61159501ddda4b11f11b6b0ab2376072a16c893041d557912e469bf69f8028df802f80204fc03693b76f80228fc2095fb81ae26150c521b9d12c6b5648491a58e754d68c94080a4234a165304fa5ef8028df802f80204fc03693b8bf80228fc20e566d77d632e066ce11ce63c7630d2377f035f279d81e43fc4e808f3b13e8157f8028df802f80204fc03693b67f80228fc20c2c4353075d67f17a4493f11fd00b5a568a1717574a287cb8ec31a548bd0e277f8028df802f80204fc03693b91f80228fc2091ba9012778ebd8267a1300c97e270ed179256245b140fb9b17a1219e4ebeb71f8028df802f80204fc03693b8df80228fc2077217112d73728408028268c1c1c22d7e5906decd303d8882b131464a18f8a53f8028df802f80204fc03693b6bf80228fc20d114d3279e41c758bac0c365e3c43f4813ae8dd5760debae76a45d3d8032c41af8028df802f80204fc03693bcaf80228fc20eaa6a228becf1b473390dc5b976b3d35980aae38aae35c771a378f36eba77261f8028df802f80204fc03693b9af80228fc20777919d25b8408faf7f5fe33af3ec9ff5530616bb8932bcbc123563fb370310df8028df802f80204fc03693ba4f80228fc200ee18f3ed4b192dda5ed031645f1563a7f9a8c6999195bd8634fc13825820a43f8028df802f80204fc03693bb7f80228fc209bb7f39b8de8979ac2a852130df2b3de0119a2f9d41ebfa14abf657e0256170bf8028df802f80204fc03693bc8f80228fc202df1038eed94ab4de5eaa594a462667d11ee669bd148be4cb11ffd6ce7aca97ef8028df802f80204fc03693bb2f80228fc202a2fa592eda6c352417c883f973bf1bf5b8587d464b949c3350039ff2038d169f8028df802f80204fc03693ba6f80228fc20d2c45f82511210973ec33d37ea8143f3fddb6d4d87670bae7ca4adedddedf053f8028df802f80204fc03693badf80228fc206565c62d99ddd0fc71998e76f43917e515c9fc99928be865e1b3af658a91662bf8028df802f80204fc03693bb3f80228fc2081faf272b805653cf50c3912ea239ed59d33c8d9d42164b875be85ee6c57cd3ef8028df802f80204fc03693ba0f80228fc204d2266520eae6b257b0a9f42694f21569807b52b49a336f02b64e8556762272af8028df802f80204fc03693ba1f80228fc2021e2ce242c5671bf18d0a0fbb616cc6b1ece072446bcf60f21eac5d56e5cbd35f8028df802f80204fc03693bb5f80228fc200441b644a35a9664e37a75dc14c45cc68230bb7109aa68c08699cf0aea884552f8028df802f80204fc03693bb8f80228fc200a6864cbfa79200d99cb8847acf3e5e107680d194e13fd92a73fe5beecafc414f8028df802f80204fc03693ba8f80228fc20cedd4f4c975d38acf1cd40a63617160a6eaa8c2c46b6a52bf447e1513646b13df8028df802f80204fc03693ba5f80228fc201ad893011e60ac7a7fae696b709e0d76d94df0200c0df901b35150255c29be5af8028df802f80204fc03693ba3f80228fc20aa00b76b533ed079f3e0fd901e4d8cf7bbdccfafa427b5d9448fd1891e301a43f8028df802f80204fc03693bb9f80228fc206ffc453473ed4063430061c9f5e093000200f548ba94abb8acc33d8d5ae91f35f8028df802f80204fc03693b99f80228fc202f4f408673dffe094dcc6c8738ad4f14004d542860e5e6182f56ed09ca1ab701f8028df802f80204fc03693bc2f80228fc20564a2af7f0ff0ebec0dc94eea7fe5337170681f9f16a6ddf721035ccac2d9341f8028df802f80204fc03693bb1f80228fc20ce4a64ae4376ef03e24f453b187e75db3cf02d72a3a33a82a625812678c21b2df8028df802f80204fc03693bc0f80228fc204ffd8f83e20130288be2f6d17b3924d9bcd23917d9671ab7b6575dffc48fc07bf8028df802f80204fc03693bc1f80228fc20f0fc387b4de741314e8e7a8cde04fc41d82260ad7ea7615949551f198481746ef8028df802f80204fc03693b9cf80228fc2037a93845eb8e893c81811ec0de9f72fba89430e5ab700c1874a94da8dbfdc304f8028df802f80204fc03693bbdf80228fc201085229ae5a463c4227d09e40b78a69a7b67bd78c8b74295bfcd46e0622a7b7cf8028df802f80204fc03693bacf80228fc20befc365c3ac8ebf447fd3d3df36f4d86bb5600e118f1d7740873833092340544f8028df802f80204fc03693babf80228fc20f1ff8fe82ce8a0f4a5ed46a504ccb4b8eb859e005be76980dadf339277a57f38f8028df802f80204fc03693bbcf80228fc204423c9ef4614df27947562be2a1565e4fd0668bb243f85ab51e9b6388f911a69f8028df802f80204fc03693bc7f80228fc202d3b5a5f9761e515b758f3affbfa4af5a66d7768df4f7ff6aca7bb2ab1ba7549f8028df802f80204fc03693baff80228fc20ce4a22885f909294f13797739d3c0c5ac8cbc61da7b6a1d4b088833408a6716df8028df802f80204fc03693bb0f80228fc20f9f4e775569b6b25069e3fa292d58ad40be761e639d412c58ef00ec39334fc60f8028df802f80204fc03693bbff80228fc204eb31b1dfc612026066ab7c4e0bc9457cf4866dee2ecb54b16504841ff6cde65f8028df802f80204fc03693bc3f80228fc2052efaedb935f575b570b320a396009788a984c346f08f839afa824d926451e79f8028df802f80204fc03693bb4f80228fc20bff66a80b165f2ec4e8f4479b5a37e4eaed46bb6b3814cb2d551431677bbb750f8028df802f80204fc03693b9ff80228fc20b53ecb7155d32b343f53e28174579927b172b48074bc9bd0c5c4816ed7b9400ff8028df802f80204fc03693ba2f80228fc209636ea6875a36076a6cb8ab9d533c6ccaba137bc7b29775bf4e383b54d819b7cf8028df802f80204fc03693bbbf80228fc20766b13a51f5f8c77f3b0b134ea18f5b3c541f8ff5541b2bedaa1fa3746fd3225f8028df802f80204fc03693baef80228fc20cf548fc6c30bd206cda52bfe1c072f6e89377712babccc7ef7e3463041a70f05f8028df802f80204fc03693bc4f80228fc200d4674759b4368658c79ce468b35628a9ca1954a387be5deb0b5c5b7d959180bf8028df802f80204fc03693bc6f80228fc2096d4c58f27f8f6a0afe927c70889defe16a8dc146a77b68aaba3b7c7c1a9c673f8028df802f80204fc03693b9df80228fc209621afa9c70ec3b038c6e4c3ce03e31ee4b87c3324e5fae3f7e986eaf1745d74f8028df802f80204fc03693b9ef80228fc20299a8a50d30624d320fe67b9f3a3a1491ca39fbcbb866b21b9586802aaf73941f8028df802f80204fc03693ba9f80228fc20ced3ebb923293195f795b3838c3e63b85a347c52f92c8231213addf838b35a69f8028df802f80204fc03693bbef80228fc207c9aabf1fbb8b45fd257a5a89399b10650dcb866aac6a619b94814dbf7863b75f8028df802f80204fc03693bc5f80228fc208cc811692f23e96d515afaeced7772c60fd2ec110f56fec67bce72d6162a8273f8028df802f80204fc03693baaf80228fc20d5be1f360772a6cadb7a0a210af3db925077768601303abd56a3d5e2a9483a06f8028df802f80204fc03693b9bf80228fc2098dac1ab4d26ec357e0c57335e3876f315ae5f0fc116570a5bdf53e888f74b01f8028df802f80204fc03693bb6f80228fc207a8e2e8fad5febb09e8e1690fb7ba584bb3cc30fb7767b8a97d48a955afbfd12f8028df802f80204fc03693bbaf80228fc204642c2f578fa37cbd21cd5dc0b5abfba159ff742aa6c4dc8c39ff0b51794cd18f8028df802f80204fc03693ba7f80228fc200cd954191b12bd9a15f7724cddefda4f2b6d157be1a21f385497dbd598573467f8028df802f80204fc03693bc9f80228fc20e2f1c76a5a2794c53fdc1a463c49cd4b336bcc8f81eda8a16334caa307cf020cf8028df802f80204fc03693bd6f80228fc20e04d5a1f4f51ae43aafda1f89405ecd824d66c874cfeb953501a11fa204d0257f8028df802f80204fc03693bdaf80228fc209669190a087e271c3ffc9e055fbc9ae25622078764d8ba1d528afeb642277d47f8028df802f80204fc03693bddf80228fc20edd253fc78f8aac6d07ca07bf4ea97cfe494ebe668f2e08ca59850985c4eb601f8028df802f80204fc03693be9f80228fc20273eba39b861cf4a4f64f52eb70e3b05e6d43e511a1f1aadc68e2d4305cd235df8028df802f80204fc03693be2f80228fc20d4c85046bd1d0d8a1f4284366a991bb38d1c0dd1968641150d931943dec19518f8028df802f80204fc03693bfcf80228fc20ec2a67dd4dfb01453e2b4e168bcbc06f3be64f74c9d515ca5fd8d12b570c5967f8028df802f80204fc03693be8f80228fc204009e56f74a987ed86453cd1028b4a39ec83c39743a89aa41f31069b353eac5cf8028df802f80204fc03693bf9f80228fc20e599b2eed296b4dd26c182c680478b05d090744d6195126c5b73634b5b886676f8028df802f80204fc03693bedf80228fc206785b69bd2981b3cd6a37673144a6096d42748b3cc9de98262e7e3aea83d5c76f8028df802f80204fc03693bfbf80228fc20fa2208b1a0561948fd3e39c817ad6148971cda8139ac3d64835bce0d9c0fde76f8028df802f80204fc03693beef80228fc203d467fbe5393893f6d3302d0376941319c45966d19c8531e11ad8fe2ce8e9153f8028df802f80204fc03693bf0f80228fc20989c6ac3d340dcf80ffad074e75587e03dc74f6685c9e09042eb5315dd46ee1cf8028df802f80204fc03693bcff80228fc20e4dfafaf58c7968f7f2222b634d4eb87f14713cc9e231624e55f5f476f66e66af8028df802f80204fc03693be0f80228fc20e0ee6f956bcaa09f3be0d47d4aa1f84187fa7c451ff78759d3093b3791644e0bf8028df802f80204fc03693bcdf80228fc20dc7935785dc38a52bdfe8c898d9a1abff5ebc55aa9c294163ac55430e9215165f8028df802f80204fc03693be7f80228fc20119bf4b595b18ae17a731c3588a6de2811782de13cd7387e0994944c9f103956f8028df802f80204fc03693be3f80228fc20a0ab70476b91cec487615a50c6471480e49a0a43e68486f518e4e0fac85f337ff8028df802f80204fc03693bf5f80228fc20624b5730133b85d8fd432eb43529fed486769b4a3987a1905a3a8fe67aa42946f8028df802f80204fc03693bfdf80228fc204cda0fc0fe47c68a0d6be690fee544f3911c6febf6147669793da4f6eaf0e70cf8028df802f80204fc03693bd9f80228fc206748543b7799c9654b9e536958e5de4da3c4b1eda1d099f73f01e18adf312904f8028df802f80204fc03693bccf80228fc2022557019c4182cf521fc2600781bb3359ce4ce138e591f6c477fa68b7a82084cf8028df802f80204fc03693bcef80228fc20d2ae5fc442a4bdd9cf32195a3f4a5d97f2ceab09be53e5f2195cab0111f9c447f8028df802f80204fc03693bd2f80228fc20bad6db4f1a4704fd45cbd6198fa3332ce7d75e569df95c45678732af873ab83ff8028df802f80204fc03693bd1f80228fc20b19d7177ff402febd4cebd872970d7278a6aa3572524e2cb0321b10ca1db561ff8028df802f80204fc03693bf4f80228fc2090d5b68481184e0b4bd06814402cec172b5af2b111d319dbebac589962a49e78f8028df802f80204fc03693bdef80228fc209f99d8b3dac31213e543383da47f94cb6daf65046142e37e1ce92d19e3d9730cf8028df802f80204fc03693bebf80228fc20cac581f17a3298580a21b9d3adfe3a9da9d76796430976581f236d956e9da43cf8028df802f80204fc03693bf2f80228fc20caf168b99f0d29349799a17aed4e22293b808e2978b9e482684adb0bb533f654f8028df802f80204fc03693bf1f80228fc20c8be191b66daeb1d3e6cbca6177a2da1885b4832f5b6ce3751bf10b51dd9bc58f8028df802f80204fc03693be4f80228fc20faf3e50ca8394f25f7784f32b83069eb1d69523f957723ab88e65663f1ee0e64f8028df802f80204fc03693bdbf80228fc2030f982aea71bc6051c0b9995ac384709f775e7c4e1c01c107cb96f10d29b896df8028df802f80204fc03693bdcf80228fc20b4b2cbc7480d38a4b6595628a535db5058ad9c8dc85a09aa74bc33634a838520f8028df802f80204fc03693bd7f80228fc201593822852e10aceb65b61f4544ad1feeab986924318f6aa0a7bfb20eee6f279f8028df802f80204fc03693be6f80228fc20150f4cc5760f9c28fbb708b2a256313e7fbfaa8bb8d7d1a3a368c56252e00d6bf8028df802f80204fc03693bdff80228fc2099f396972e49e4ab72af0badde1fb31fd8262b4b2ade0baad0848b68e3c3e460f8028df802f80204fc03693be1f80228fc20d34dbed0fad93b10a9045db093f3839671ccf3e2d8bc857bdbbce0f1f20ef143f8028df802f80204fc03693bf8f80228fc207fdce54f04f58bc577bd216d8231757a763887becd5ecae5c6e50043c2f93a77f8028df802f80204fc03693beff80228fc20c8d1b29b7792fd68da7be10b4752a91fd05fefbffca1b55c9184b04590dea040f8028df802f80204fc03693bd4f80228fc205579d5019f24e221f1a106498e7725e5ac58bf532cdc6b7af2d9a90add195b3cf8028df802f80204fc03693bf6f80228fc20d1fbada8bdc76c49df2f4f2d5026669f670df1d99722a99f0e82c2a1c3bd9e2ef8028df802f80204fc03693beaf80228fc20928aa337e1afb788661c36e5b557bf5fe60ec37e645aed0e070c76aead7a431ef8028df802f80204fc03693becf80228fc2064426bd67b271e9831405537bad6399a957b245439e4895215caa858c7c4136cf8028df802f80204fc03693be5f80228fc20eb621b8eaffb1233e2c0a2efc2b33edd5b765ebebaf8f30fe1a567a121c2a901f8028df802f80204fc03693bd8f80228fc200e1a8dcb442d1973ec2756864796272521c4b45161d817fe22e9a1f55840be59f8028df802f80204fc03693bf7f80228fc20c1cee98f66c6f8073509abe27b1a753f8cf59b9148e29c32f41b9cf7946a6f76f8028df802f80204fc03693bd3f80228fc2045fc8453c05e74b84bc6d8a97949464b70f21eae176b622ed80ce58cf4ee1675f8028df802f80204fc03693bf3f80228fc207199b22fd7df5c0ef912f4b0e5f43f97ef3e05d568cf378154daedd8ccc55000f8028df802f80204fc03693bd5f80228fc2090b2c2890cfe521e1a0bd048c1741a70469a386c98119f582ee2216eb9e1bd0ff8028df802f80204fc03693bd0f80228fc20b014c0f44de5cb51644ff870ddecf3f2e5814844a81ccbc6b9727b75e63e5136f8028df802f80204fc03693bfaf80228fc20eb00162de8e8f6891a5841dea27c3bd1545562ce09f08655040104e5c76b7e46f8028df802f80204fc03693c1af80228fc2073e5198d84a0d6e47ea22f0c3a0d931f86510cc79a25ee721872e5562a0d4336f8028df802f80204fc03693c0bf80228fc20c4731c55820970ddcce154baf31168bbb67be2cfa5b619e9a0680f2cf46c230cf8028df802f80204fc03693c2ff80228fc20a8288fb72590a491476f1ccba4f474180b0f2264ba8a640febcd68a4ed058675f8028df802f80204fc03693c23f80228fc209f6b4ed64c7d5bae3862de728b8d3e46e9f25279464f1e51882f603a283e7830f8028df802f80204fc03693c06f80228fc2070a545108c14ccd98dd7d914a80caf8f22b8c1e6efc63b932d9c66452eef3816f8028df802f80204fc03693c28f80228fc201743de83cf8edd62f948cdd7760ea4f0493b483823a02e897ddb41700af05909f8028df802f80204fc03693c27f80228fc205755c7c6f50660720cb0a179a0bc22fb22c4a74e6ee5377e3d9332146f759a6cf8028df802f80204fc03693c07f80228fc20dbcf74a3b43d1b0a14496859cc22edd7310014830bc30d03936979ef5aae5b46f8028df802f80204fc03693c1cf80228fc20eec23a9231b03d839fb906ca03bd0e4e2c39ef04d74bc8a2309d92d8062e9212f8028df802f80204fc03693c03f80228fc203b601c2ea8f18ed44fc5f216ad62ffc67911658c11d141648aa8ff1b31263f55f8028df802f80204fc03693c11f80228fc20088378099c3e3d65785ef55460b4c32337a50058f13c4b4519072c942332f701f8028df802f80204fc03693c1bf80228fc207af9fd1c2faf083e1984a47022958dbf2cf629cc4addafe2950586bf13186167f8028df802f80204fc03693c2af80228fc20cb9c2b56b77f5a347d28d14607aa32ddfc0b2ef58a4c21d2d4ab16b920198601f8028df802f80204fc03693c17f80228fc200fe20d5aacfc108cfb733ef2671320ea677eac867a425bfa17880fbfbf4aef63f8028df802f80204fc03693c14f80228fc20fc7da81e65284da2d44de751cc2482139998c5b188e9866f21f27fd7e44abc49f8028df802f80204fc03693c16f80228fc20531acbadd5b3c5b26cd8113af4b6a298f72682f6767431431fc425b6fea9ed4df8028df802f80204fc03693c10f80228fc2033e2f7395eca107e4048bd5a8b47e75ac4274aaa1e2e1d7094eeb475bbb70432f8028df802f80204fc03693c12f80228fc205c6562e409b8b09f30692b5b9a15baff46f06ebd9baaaf838ece0f64023f085cf8028df802f80204fc03693c20f80228fc209ed78078271c28ff168f95288253c77f9e8b0dd3cc337709b19bf6dc6afe2329f8028df802f80204fc03693c29f80228fc203cd5ac4b237a96035cfbdd99cd4c1586588c5abff39314d62de8af72cbcb692ff8028df802f80204fc03693c2ef80228fc20b0f8fe5cdb3af6707a2d389f19d9af5b0788cb2cfa86a3ddfa793616ca5e6a39f8028df802f80204fc03693c2df80228fc200c44bfe3dd605dfbbb46f4b9d225ed57c812f4038ffbc2f16f64aa0c713f420df8028df802f80204fc03693c05f80228fc206bed7955021b15f7f5fe4536d74532d42c5887fdf7573a3fccfaa240ed140e45f8028df802f80204fc03693c26f80228fc2035fe0841ce6c5640e1c0c36d6d7462280cb9bf3dc15f0db716563e0b3617165bf8028df802f80204fc03693c21f80228fc2080f94b9289d050b28b74df3502e1e4c4726e734844ba3dc646af277a1f0e5d7af8028df802f80204fc03693c18f80228fc208a974e33d606ccfc6d7211cc33b4c5d7a02e91f02fb4b1da47d4140da6c64e7df8028df802f80204fc03693c04f80228fc20f211945586aa046b0569b15d7ddf13858297d4e3892b060fc39e456353f92a7df8028df802f80204fc03693c02f80228fc2007baa9981c7afdf2f017a91ad31e866c0995ea7b71bc6e68fb23e3737712b041f8028df802f80204fc03693c09f80228fc200d76f59fae86fb8af7eee7654045d2fe28427e944a91b58f3b713fa04ee96634f8028df802f80204fc03693c19f80228fc20630ad2b930af2582776a93d82781b7378ba33e8d223b879782aad00aad2d9152f8028df802f80204fc03693c13f80228fc20ff056afd379e8fdbcb24d69a27aed5c7061843c2e141ea65b31bf66bba00713bf8028df802f80204fc03693c08f80228fc206ce1964aeec1cc8473772f0638a2377ca7ec2988031a4c8626f39e9e628d4e1ff8028df802f80204fc03693bfff80228fc20aa247a5155f88d0468a07f7dcaaa525336d641b308bff9d3222ae763baa7b449f8028df802f80204fc03693c24f80228fc20bf3f72cff1e3154ead49cc378ceb970dce96e49da83fa3d7c9cf717d6d809854f8028df802f80204fc03693c0af80228fc20d96311b57aa0c674a9b51bbb1b7888ebe0ca70fe8357a70c3db2cba0231b1b07f8028df802f80204fc03693c15f80228fc201809fa333f3f477a7f6903bbdd8293a5d814468f522778a8efcc8ea739534f38f8028df802f80204fc03693c2bf80228fc207e34c08579ac11e4e6352bb32d64d8af128470dfedc16b4d3115da7d6b5c0a1ff8028df802f80204fc03693c1ef80228fc20756b3b1fd03c861a8113af8ffcc7bd4f052ffa57accc86b5a97ecf46fcdb9443f8028df802f80204fc03693c0df80228fc2037158a09c7b49d13c98862b015f52bd762ef1d744d58bfc69fb711211ba6ad5af8028df802f80204fc03693c25f80228fc2017d5f6676680e6f28b29bd2803402d5410fd0e8390cc9289720d89038fed2808f8028df802f80204fc03693c01f80228fc2007828a39ef9f359bdabd74dae679cb828fdb0a2ad88bb90c7d0fcb850e52701bf8028df802f80204fc03693c1df80228fc20687b39589b8d454b447beb4e04915363b2357f65c47d050003b939ec0fe7ca73f8028df802f80204fc03693c0ff80228fc2042e93748b8e9beb52adcaca243a0fb0be56d63e616ca65a3ed16b8773e1e5d02f8028df802f80204fc03693c30f80228fc20cac5da92a6cd064ccc0f9b1c49c45acd5d92c3525d7da3b5c77671fb7b15576af8028df802f80204fc03693c22f80228fc20784faaa63916f8034d208315d9e4e057c9a107abd963f6c163e4fc60ed850e14f8028df802f80204fc03693c1ff80228fc20809cf66eba63d8d187eda16270449402ebed12b2ef65f17556e164776537f719f8028df802f80204fc03693c2cf80228fc203744aa8ed20d837d706f94bdd258df7d1903e2d223bef982c276f3a71c3dd46bf8028df802f80204fc03693c0cf80228fc20a7654c30929eb0e04c82a295b1c331b4d9e43fa814deef6bbaf0c36989284905f8028df802f80204fc03693c00f80228fc20710671a638ee1aee87a09b36b5b4e26588c0329b72e59a187f2e08d29743a965f8028df802f80204fc03693c0ef80228fc2090a3f3ca5eb4766931af31132b23aa0f9b1ee04121c577aab762a3670324d813f8028df802f80204fc03693c5df80228fc20cbe53168a75c1802429f678b10c3dcb52c7a2cf5627364ff369f3e055a619f53f8028df802f80204fc03693c33f80228fc20428c464436e68e2e913a6ee0828bdec5ad6081548c4731714e3034884b5be84bf8028df802f80204fc03693c43f80228fc202959c00d05e05aee452c9f0bcf5cffbcf71c098e556ed7969977b5b865022600f8028df802f80204fc03693c3df80228fc2018d3cfa1cf8e6dd2ff26c681d226cacf387c7d8a962687c5af12724d525dfc1df8028df802f80204fc03693c3cf80228fc20019d93c97281b2df10709aff39766cba9e2094d7e20fd7b4c672ad11443a3552f8028df802f80204fc03693c48f80228fc209fb7a4119bed09317ce22e9a2db22cf4b2075923031001072c3dea0c7859b436f8028df802f80204fc03693c36f80228fc20ebc4a46ea4ac307d27657ae26d6a5774f941afaabf855c9f6e864b31f3d3c55ef8028df802f80204fc03693c3ef80228fc20c5fa2c071b9be62580893f3022520842b86ef953fce6fa57fd6f90c0c80b477bf8028df802f80204fc03693c5bf80228fc202196f25c1bee0a8a0728a4428ba1febcb1646cef93f41626c02601f9e30dee5cf8028df802f80204fc03693c61f80228fc2054b8d2ad940975d16650e2b790f5f28b5b725070af8b79e332a4cf07eeaf1822f8028df802f80204fc03693c4af80228fc2017c52977244b40120875cf921d1ec48c0165f759fe7675beaeb6a205a180eb18f8028df802f80204fc03693c4cf80228fc2070a483bcda9d8b929f409c15d1e6f8aaaecf1bcae8d44b43d33252075e5b8704f8028df802f80204fc03693c56f80228fc20e0fd9c70b49a14789cb21e76015d669cb231d8eb64b2df32f00c75bf6342c60ef8028df802f80204fc03693c3bf80228fc20de5d71157fead637b3ea2a9b5a261910bba0f966ab5360c92ae7c0a204b2f954f8028df802f80204fc03693c32f80228fc207ccb7e5e6e65fb73c67f960a183a9dd193cf4ff8d4b43954be5a73c00056477ff8028df802f80204fc03693c5ff80228fc207d752abb5fc5b03e860967699ecc58d83625b2cee19d7933c164b14b107ca515f8028df802f80204fc03693c53f80228fc20fcd6ff42f1e23c3cff305d773efdd2462fb577e3ade483ca75973a981b0d164ef8028df802f80204fc03693c51f80228fc20e5973b104b65228a983135fa77cb6eea35da19028e539e8c6e540c96c1be8b7ff8028df802f80204fc03693c54f80228fc20861918be55696fa960d2b7b63236c29fe2d116d5834a9dfb20af576c1636891ff8028df802f80204fc03693c37f80228fc209384314e7fc10018b04d07f0e2400bcc71cef12de0c859a57626673be411c476f8028df802f80204fc03693c3af80228fc204e107ea758f5db9fd23b67b28a8bd3f294204b54484dcfbe3318969e290d2267f8028df802f80204fc03693c63f80228fc20512c913a31501e2d8ffccb313d4f3ea07687e9f3ec2bfd7779a475b26fd5826df8028df802f80204fc03693c52f80228fc20847d7dbd732085d8a6aff3b0b821989ff2bcf4a0b4ce8d5fdfef6b907f809411f8028df802f80204fc03693c55f80228fc2037633c6f2afddf9063a625f0c6c66fd585e10c2f64364fcb66fb7c07553fd933f8028df802f80204fc03693c50f80228fc20ef8b246f513be417f504000c7c5706e2af9c6044df33124b8d2485daf625a80af8028df802f80204fc03693c40f80228fc20b1dda19a5c992d4f205f44a54e501f96fd6bfc3ac051f10ae5c4a0c8b3e25e6ff8028df802f80204fc03693c5cf80228fc20329fad292f78772ec040ed8bcdc52b9edd0d4898af5dc890afe3c76ffdea0024f8028df802f80204fc03693c4bf80228fc2014f14d0c7d7f00bf4ffd1197f573b5676f9c4d1c83d2dc5a75c45370d8ae7222f8028df802f80204fc03693c46f80228fc2064d6df0b353cc4d0feb76d34966ba84144cdb93b656abe2768ca00d5d765cf37f8028df802f80204fc03693c3ff80228fc2089d3e094e9ccdc6a6aaf07a7771ce5f0e5a3dcaa5efe862741fb98fb41c93e28f8028df802f80204fc03693c39f80228fc20b57fe9f018165b7fcfa1b55d616424476ac2375df51365158d8062793d151820f8028df802f80204fc03693c45f80228fc208d9367b455bac68dafa7dc4f28813ebe21f4f770b8610542b9da612bfd2bcb06f8028df802f80204fc03693c35f80228fc20d85dfe4e8277847734b97b363c1c30cd32b9ea63e2642affa67f29a50409cf78f8028df802f80204fc03693c60f80228fc2098911a82885840c3c1d2a469bb2eda3dc31ffb2a5514cb91a2ccf70f25a35937f8028df802f80204fc03693c34f80228fc205aa5bd8172301662e87fe53fd5cbbb2d6d10477b4002a33bac50612ef1096023f8028df802f80204fc03693c58f80228fc20398a7d6def8f835c3edb452134e4b1e79e1bc769df555ef3e3eca5022cf83a37f8028df802f80204fc03693c57f80228fc20cd3667dc8015f93e5b4b447922a3878f695ec00c3e288eadf51f4f426750eb31f8028df802f80204fc03693c5af80228fc20ae08cfc83d2c80f2ba857a6e8bbfa01efc91e6f55147a86f0eff4170be13f50ff8028df802f80204fc03693c62f80228fc2043d9c7a6e8bd8b811cb324e0dcf6e1d129a42c503c5bac5e6c8856483bbf2b52f8028df802f80204fc03693c47f80228fc20d479e552c0b3948c07a3cc3dabb1026581f29d072312b8c993172d8df8b8b052f8028df802f80204fc03693c42f80228fc20f9639dbc7ddaff2915863329ade64259a41219ec53199e2eec5746270bf11d6df8028df802f80204fc03693c5ef80228fc20caa4086e84a283ae7ca835ff4c72476dad21b190d1a486e48416c12362eea936f8028df802f80204fc03693c4ff80228fc20003f51673c6f0762156338eb4d1df0babc960233caa010a1d1f31c9d80031f74f8028df802f80204fc03693c4df80228fc2024c410953fec1b277440122869209d2bbf2ec98c2a87f569c1ad5102f963b145f8028df802f80204fc03693c41f80228fc20314c189e62264146eca01daf55cff79d135836cb8f4c0f51164d3603d030772cf8028df802f80204fc03693c4ef80228fc2093e840c6eed19756ca3c202ee20ee2bbb9d2ae31cbe78ea5f0194d4182879574f8028df802f80204fc03693c44f80228fc20b78679d2a3ab270e1b10458c03bd428880c489979fec4fb418a32e3a6ad51648f8028df802f80204fc03693c59f80228fc20da2802d4d988fea060ecb301c1ca519a86a68169daa0ee05a7bc757468236823f8028df802f80204fc03693c38f80228fc209316825da07953e352027631ffd60b13b5e050d553958aeddba539dfb999401bf8028df802f80204fc03693c49f80228fc20d3517bf114a4099ce4c5947c7775f813b0ab7ca0b6671ab9b3ad47a0b13e960ff8028df802f80204fc03693c89f80228fc20879d8320c8191c291dc08524168c811cf059106f191553fed5225897d7423532f8028df802f80204fc03693c70f80228fc20faa1f6fefc8114953fb0f85b8b07e35a9f67685fd0c9cc4db575af9e608baf5ff8028df802f80204fc03693c96f80228fc20fe37448271763019f11bb47273e425bc5a34db2d9267d0e731b391c8b20c1e38f8028df802f80204fc03693c72f80228fc20568fbb3abadba81fe7a01517b5161f663e84146a59c03b11835b4877a474be29f8028df802f80204fc03693c81f80228fc20ddfe459d6dbf2a7d83906035e93904fb6ab9581b2c489d657df1ba1929faa336f8028df802f80204fc03693c88f80228fc202f0b0ac20f017d9faf16bda8ff2ce8ce2a1c01a51107ab0c710c78947e093052f8028df802f80204fc03693c71f80228fc20e6eb9be6a4fec7bb3ddcb59c4de9cc18371406f21c8015a3ddac1333c2f3d05af8028df802f80204fc03693c79f80228fc200b5abcca6d885511fd2d7d8d9641b1dee656fecd652c7d1827125e4a87a8c306f8028df802f80204fc03693c87f80228fc20b67a41f5365ea03c17d94a614a23ebb3cc516cd982fa5c5f0c0ac1731610bb4bf8028df802f80204fc03693c74f80228fc209749e51f340c14297de7d93b434c50a8e05955a1a70ba80405d001d19cf8d958f8028df802f80204fc03693c94f80228fc20c2e631d2bd3d806b8b1ee6b75a3579ebf5d3c8844aa5f4b93d62e31fd6e6423ef8028df802f80204fc03693c6ef80228fc201005bec5c947dbf0730be95d90b81ae0f5f76d8806c6ac15c703a86d2e45f959f8028df802f80204fc03693c8df80228fc20bf1facd93d92427c819f2ad1d7270c5efa94fac1ee00c60c61323470c4566711f8028df802f80204fc03693c92f80228fc207be07a3157e8762066860697c4808255523cbef3c7b679b252a5b6bde8ef0241f8028df802f80204fc03693c84f80228fc20f04ac6f663e8c8e3ce1b20cfd010120edfcc78d8194cbf9d90ab4152e0d5ca30f8028df802f80204fc03693c77f80228fc208630286e9811a3a553db9c2c1cd482e9049200d920cff9fa305682aac1b6697cf8028df802f80204fc03693c67f80228fc20ec88b8ba663aaaea259a1d19448bf39db59e460c1a53e2755e6503efa1d70173f8028df802f80204fc03693c85f80228fc204a143c7851e2a14fe726e92adf53b4db3809b7f7f8c45515463349cff2f34825f8028df802f80204fc03693c78f80228fc202f57de40622f91b7f180a724619cfb89c9c9bb228f87f2e0dc9bfabdb1607b66f8028df802f80204fc03693c82f80228fc2023d1f66faaa948320fbf0854198db480f4244f11ec49e3967e2924e6b52c3a76f8028df802f80204fc03693c90f80228fc2071da09535342de69b9fff5b80b21b381fab277da75055f05ddfd4a0b9d93f562f8028df802f80204fc03693c93f80228fc20758ceb25ae0782c00d6c2da03ff20588fd6cefa9c918c0602b0be158dce0cb0df8028df802f80204fc03693c6af80228fc2097a6f441f96eae67e42d4357cc49df2f173a869903fd7119035c62e3fb9b8c15f8028df802f80204fc03693c7df80228fc201303dee1314c4aa671f4f94d5aa5a15aef5aee27198b7e281103f16dde710f56f8028df802f80204fc03693c86f80228fc20549a73a72b7e88b7c8c06ce51bc5243f9c4b136586ffc278399034aeaa550c0bf8028df802f80204fc03693c91f80228fc20c8ae7b28823eb5491384be4e40e1f7c673d1aad8ee764ed1df7247ebc9bcc217f8028df802f80204fc03693c76f80228fc203ffa4cbb830b0de590a73761bfab3edbd129c96e311cabda20c2c734652d3e19f8028df802f80204fc03693c8af80228fc204cc7b149bd5e69f3d7f7076bccc1bc7be9397dd65a800bc9e1bb99ab36d45914f8028df802f80204fc03693c8ef80228fc20a325c1757673ae3ee4e3256af3edb7e6b1d907b8b4586a9f26ff7ed6188e5f72f8028df802f80204fc03693c68f80228fc2052a7e143b4090ad4084d309bb689ecf45fb5640ee2c9267f87ad4e3796da5441f8028df802f80204fc03693c7ef80228fc203a1471ae929a125977d649177ff91b68df67477c59a0fcab045c704a44a56762f8028df802f80204fc03693c73f80228fc209ebdcb3e0da4d766e6e0ca13e98c596b8fe38e04a8532c10ced9351cad74ad6ff8028df802f80204fc03693c69f80228fc205de0f71936f85b43afb4a792d6e07c6137814d3d8b4f127e482f05754b679359f8028df802f80204fc03693c6bf80228fc20cae8760db39cc9186588cacb817168b016ff4f0bb72f96c0c27c9e2b3d10b507f8028df802f80204fc03693c8ff80228fc203d103fbb2ed780daf21b1d6d7b086a6776a0ffd23be9e38b40e89af0ab2dd94cf8028df802f80204fc03693c8cf80228fc20b74c1ae4b81cf3fe945480f6bf155dcd2525891a3b34761d18072ae4de534121f8028df802f80204fc03693c66f80228fc2074f63db90379fc996cea877e759f83b12aa26c0147f444ad1259019b140f3f0df8028df802f80204fc03693c7cf80228fc2084770e26273a468fef3ef586a6deecbc492155b77fd3f64b3bfb9d470d9e007cf8028df802f80204fc03693c6ff80228fc2060b43fb9d6828c1a190370e72493c9db5e0d7917ce9115f987f6daaa3fc48347f8028df802f80204fc03693c6cf80228fc20c3e59b43482113a00a6ff354b9054d286c6dbbc9f051798568bbc1af80b90379f8028df802f80204fc03693c7ff80228fc2050e1d5069d3b2391e590421eaa33d548467ecd7340cb3d42f52b9f6654bd1469f8028df802f80204fc03693c7af80228fc20cf73c7e47738ffa6b042796586eab97573bf8667ff2b6e73541b6e5bb618d027f8028df802f80204fc03693c6df80228fc202d88ea7a00b970b1ba03392b6e70338a652d6f15e1761167acc5ceaf359acd61f8028df802f80204fc03693c7bf80228fc20fd6608b78651412f39310342161d302a8705485b5ca80c63960d93a838da8e77f8028df802f80204fc03693c65f80228fc20f8d98e96dc933a74bdb2481e965c995426302f63bab0827afa699a2fc5caa818f8028df802f80204fc03693c80f80228fc201a7172c1573fbaaf090ec0c2541603a198f209e6d33caf2fbb8d6fd8c6b93c23f8028df802f80204fc03693c83f80228fc2042e9ce1c111a183546d138320b908793dc243994b71484ccc937704905bd1c5df8028df802f80204fc03693c95f80228fc2081ea0a7d0d39536c04f97dbf776064b7ea5161f559ea6cf57d922e009271d251f8028df802f80204fc03693c8bf80228fc207326430c24c9a60151c525e42f3ff410f054a4994425d1ff2011261e028f5a14f8028df802f80204fc03693c75f80228fc20842c8bc7c9c496fdc07298241b5fa2b57610571cbc5266aedb675f6dd097f36ff8028df802f80204fc03693cc7f80228fc206602f1dbcd5fef5d5d4a5f0f25c36e207e9156e7309ae3a5d10ce49e35fc4273f8028df802f80204fc03693ca3f80228fc2038881d7d96b8da3eaa1533e9b86692a2ef19eb36719b4f485ac96ee01daf0678f8028df802f80204fc03693ca0f80228fc2024d019bf44a7c2f1e41f86cb1f3ffa5047e1107876124af95fe9c1ef3d76ef2bf8028df802f80204fc03693cb1f80228fc207c200f4ca148684d37e4f39526f0d1f01c4cd6b2176bf0d4c8a6094c9744bf4cf8028df802f80204fc03693cb7f80228fc2072f3b38c6e6c7693085a45d28c1a08f11a79cd88b3350d367f79f130d1f87066f8028df802f80204fc03693cbdf80228fc2076b708cd80a55c060268046c3bd798e20b670381272d0255ee74d8c58c9a3802f8028df802f80204fc03693caff80228fc20b9aff47ec035101ba00f3ae61c81e9cdf790678c312852b35ceff68132992923f8028df802f80204fc03693cb9f80228fc20b63fa6918cd7eac6369d30df7defa2aac0830f832305ff689cff0071dc610c62f8028df802f80204fc03693cbcf80228fc2027cbe83410baf5ef7b7fae7ee18a72811cf1beaa3a9bf725d9543148bc57505af8028df802f80204fc03693ca9f80228fc206cda0c09313f22481d2b9f7bd448fa5af42786778695b151783c8b2a168f8b0df8028df802f80204fc03693ca6f80228fc20d9eb85c46044765037d923ae2320f3f49de28e0d62879d5d85c7eb5011e7b720f8028df802f80204fc03693cb4f80228fc20fa1382095473dc316496c02d0008354488e089930fa6b4827b576fd69af9cf0af8028df802f80204fc03693c9bf80228fc207928563ad6d92eed787a20f5bbdde564d5a0878b24e94bba238e1bb57bd8ef38f8028df802f80204fc03693caef80228fc200859b19f3000bf11fedc65f110232d47521b762afe3c1d09aede2846df9c116ff8028df802f80204fc03693ca7f80228fc20cced1ec56758a6bd7e9bcea4c3f18fd055e9aeb6f392a9bf73f43db591bf5758f8028df802f80204fc03693cb2f80228fc2089ba92f5b76b8252c594d2a528245ae65d2dec4c01e7c4aa4a788fc16af00226f8028df802f80204fc03693c9cf80228fc20d2f7176aee1814dd4dcd91a42b42a9850cd53422ef369750346c95445f9de64df8028df802f80204fc03693caaf80228fc20ff09690ba451436308720d8c619cd3fea4c20bbcdc737efa0a3f7829fd506102f8028df802f80204fc03693cbef80228fc203d9517a77da0b573f6e28c23b00b72ada482ffc6822b67f322080fe4fc0ab61cf8028df802f80204fc03693cc4f80228fc20a1dcea2871af749101dd5a37b0c1a559061ba4685c9db4e6e8b985c7a9082d7cf8028df802f80204fc03693c99f80228fc20749f93fee3f57e192828ef1dd953461c216ad812103cc4f213108007daca4746f8028df802f80204fc03693cb8f80228fc202dbde96a3a2e2dfb35fc32a32534486736047db43ac0c3cf2cd97be148c76040f8028df802f80204fc03693c9ef80228fc205b023cd3da682c6fab6ce63bfeca26d79ec8c00e991f8d3a4627a1a208158069f8028df802f80204fc03693ca8f80228fc2028e1f0d642703ba837edefbda4ce6dfabd358fe9092da1f354857e9b9b99f44ef8028df802f80204fc03693cc9f80228fc204ee66221028a1ab41b541710adeba1eeb5a9a1748902552291b74db457664e4ef8028df802f80204fc03693cbaf80228fc207f0185459a3195ce7223846908e708394ef6ca92c6b5c779d0ec997d5d65001df8028df802f80204fc03693ca4f80228fc20b1002cbee2d6fbae273b9d58622f029f4c08f8a8c72b79796e1e7fefa2377101f8028df802f80204fc03693cc6f80228fc2058e9ca6c9dac6f4c5dd81bf9fc389cdbd75a7bfd1acbf42e4b8188a3ba20d22cf8028df802f80204fc03693cabf80228fc2067cf7dabeb90dca1ee2df94fd0298415d2156191dcfbb0170e5bfc4d7c8d296bf8028df802f80204fc03693cc0f80228fc20a45445215e66e517c21776931f94ede8fcede191d17e73cbb321683d8022a709f8028df802f80204fc03693cbff80228fc2070dcf6eaf5d888d3e97930a764bf60b59922565ce4c8eedde5487db13288555af8028df802f80204fc03693ca2f80228fc20647730e72454414324fbe607c214eb77533662bf58fce885ead92315c691fb7ff8028df802f80204fc03693c9ff80228fc20ff3a747938edd0c1f7fc32235fc486080cefb21ee1254833ea10b9460382d734f8028df802f80204fc03693cb3f80228fc201a096788c70aa35af550b6a4f477134b1adca522e41d0d164bebb14a7f93da6cf8028df802f80204fc03693c9df80228fc203adf75575d35719c0927601c4127331608cb7f1caa28afb6f417d47a3bfdb634f8028df802f80204fc03693cb6f80228fc20eae68065e40d6b5e5f360b03730c8663a28bd8851e1d325858bd4a25769f1a3af8028df802f80204fc03693cadf80228fc208cc987d7b6cb9d153048ed2092881b3d574146dfd0b1d6ae576a4b3f294c3667f8028df802f80204fc03693cc3f80228fc2023d5d6e0335b8adae06f31d165fcc75174f618e8b673328bf3670159cc38ef6ff8028df802f80204fc03693cc2f80228fc20e6ae922dbc07a5e2ce7f38e73790c299780c4fd6c5e39700968d64cdbfc0cf22f8028df802f80204fc03693cc1f80228fc20332a68b6c70eb574a1767c623daefb53cc60de7a6d4680277cde16103a2f692af8028df802f80204fc03693ca5f80228fc2043e0e9ccc8dc82dfc1da7bae3ab8ce4f693801501349820621bc49ef568a0d3ef8028df802f80204fc03693ca1f80228fc201f22372d4fd7fa2590fbc069345c02e82c47cc7fd1ec36dca887dc069348453af8028df802f80204fc03693cc5f80228fc20759df3cd0221c4f28d2daf7e08626fa3453560857c30972e21bacccae501f97cf8028df802f80204fc03693cbbf80228fc2029c1defa200976d2a919284006891e64b84899bd2ad884e058da67e06ee1eb1af8028df802f80204fc03693c9af80228fc2029d138c1d6ca64a0cc3a7508135357c52c4f09d8a4dee5766130e1cc6fd9de67f8028df802f80204fc03693c98f80228fc20acf0e1b7e1d0cb509dfb83f0fdefdb9e5d9a5b5866a0e40eeb04197aee7bd039f8028df802f80204fc03693cb0f80228fc20029a8349c3e9450bd0783eeaf833766c669246d606fbcce59103cdabb6ddb13ff8028df802f80204fc03693cc8f80228fc2087635739a211af917bea80fc738348648e82edea16e9a51ded8941eb9796ed0df8028df802f80204fc03693cacf80228fc20ea2ef18d841950cd362e641e06e51fcac47848aa7ced54d1dbb974ae71bd9257f8028df802f80204fc03693cb5f80228fc20ff7ee98c4ce66df1d9e80e36abb7e70045d3abcf2b046fef080d607e0ba5152cf8028df802f80204fc03693cd4f80228fc20da1d514f5c500314674934f95169947255903800c2a52c047a364e9a914d7c17f8028df802f80204fc03693cd8f80228fc2023ea994e88d8a38e250991506bcec04cbf50f809f23326a1352e5462a8cfad1bf8028df802f80204fc03693ce5f80228fc208986331155566e11582c3c9b7adb1f7e82e76d6f66390a1a45ef310540114b7ef8028df802f80204fc03693cf5f80228fc209b4f856bdc4e1e13dafeddeb8aa66c2abfd6a71ff6a8a606772c4b7b1e91c07df8028df802f80204fc03693ccef80228fc2083f784088a11ee357daa4e3e21186241616de78007086a26a9ea0eab30382c36f8028df802f80204fc03693cd2f80228fc20976158a8cb89a987a69480f2bc5e6d046645321be114bf8e892431b0a550d00cf8028df802f80204fc03693cdef80228fc2034d4366506c87e98eb89edb20043a0826583e6c559520f83f324687ff1ed5068f8028df802f80204fc03693ce4f80228fc20fab9d6d222ebdf4ce9ed39689e257b1f5e97ba3c91ff40af8e14c0083d143055f8028df802f80204fc03693cf1f80228fc2061df95fdc3b8d3f45733dc7274d57c0c5963e92c37a67b8dec7fc436a090d830f8028df802f80204fc03693cf6f80228fc20f89e88d5dac0b65a78a06ae3c92ee20b11ec61d35605db9d3eaa33b5e6d4dc41f8028df802f80204fc03693cf0f80228fc201cf14e1ba122b7cd84d1bc7ffa7ecb44b9386cfb7766b3ba56421837321e0c1ef8028df802f80204fc03693cdcf80228fc202f36c14f8d8ec0a95e5d0e55135a69aebd7486fc72c62df80a502707522c6e28f8028df802f80204fc03693cd9f80228fc20e12e1c0676e9aec421b374268131b1396ab84dacbe501674d5ccef149e51ce23f8028df802f80204fc03693ceaf80228fc20954acb48b1ffe44f1bdc702a2ee289c7654f8570dfe55cc93865b81366ac2d1bf8028df802f80204fc03693ccff80228fc20e709de5654fe9c543a6c95d327793425e40307d2e3049c770436e7320c28f634f8028df802f80204fc03693cddf80228fc2077e03a289576344aa6acd6d1087b3d284cdc5331c5f4d723019dd998433e7366f8028df802f80204fc03693ce8f80228fc20242861f6c7dd31ccf4e3df119654cd2512cde847e511f8b958233a4bed7f7a49f8028df802f80204fc03693ceef80228fc20c35a4b289dbb5634ad28036c406418cfefb615c6971c8b8576db248bc8e46935f8028df802f80204fc03693ccbf80228fc209df68717ff24732a4c63d888678055a2ed1afa14d661e149ac918b56c3d3a056f8028df802f80204fc03693cd1f80228fc20c86c51d802a8fb7e22910a5a9f36c18f5f00a7bd483ff156bf9dcd05dec8385bf8028df802f80204fc03693ce6f80228fc206e62f1dec3a81aa87451a7410edb20304e8fef3cfb32c2a481338b7fde8eb869f8028df802f80204fc03693ceff80228fc2015b6e076ccd8755aabaf3030aebd0a4633170058d01aa9aa12fe9330e0b80f09f8028df802f80204fc03693ccdf80228fc20656412539d52da0dd59da4405522dc594dc17dbe80c832358c4e3579a4b36710f8028df802f80204fc03693cedf80228fc20e751e1f77a56bd464c21d65b8eed79db241ec46c41802e9822181a1eb958d346f8028df802f80204fc03693cd0f80228fc20731ed64d37df422f3950c92cc8ce5a0544d99da05f2c48a865426c8c41307e10f8028df802f80204fc03693cd5f80228fc201be02539bd69b619ce80e602f655257eca5021a46b43396418e7188cb7fc1644f8028df802f80204fc03693cecf80228fc2067f869d4d70627c79f13d194ddb5333f10466ff9f9a79a5e9fb75508d1fab96ef8028df802f80204fc03693cf9f80228fc202b07b30aedf840579609bd2adabca7478fbd88318d8e60dea11b8c000e94ea2bf8028df802f80204fc03693cebf80228fc2071e4488f8ad19a011adb8a175c9af9c4c6247af4259a039cdee4d69ebdff7a12f8028df802f80204fc03693cd7f80228fc20e73f166533fca9862c2768349bb9392e7e41387458517cab0c762fa44018dd60f8028df802f80204fc03693cfcf80228fc20e4180381b22b9480a829072e129d061a76a3f35c95376355e64ff28164d57f56f8028df802f80204fc03693cf8f80228fc20ac86cb96acd064b8a5757c20b69b3b70493c445e28561b1b8a480efba70a0210f8028df802f80204fc03693ce1f80228fc20505959fda8a7c1bde33eaede77b7f4cf10a50ea3e5a581bd31c700e720c89704f8028df802f80204fc03693cdaf80228fc208e9a0f9dcee699cdd45bfd411b3fe23daafff132eec257fd52f33a10b468cb27f8028df802f80204fc03693cdff80228fc2017b402f48aef7de492c05eb2851848a20a2a54c278cfdd63fb2d275564571d55f8028df802f80204fc03693cf2f80228fc20c2c08232b51b384937d7a00e0fddc0dc2145eee7f448e5a13ee75004f0a99830f8028df802f80204fc03693cf7f80228fc202e8d12cdddc0ae72697572b3b89684c2afd49fe363f3d113842f4c4c5cfc6920f8028df802f80204fc03693cfaf80228fc2024204befdf7a07d3aabb55c44e8280d0df325d933496df46c92dd869378efc7ef8028df802f80204fc03693cd6f80228fc2085d24f02c5e4fd36cdb16a939c0856e30d116b8541e13ef09b228e8114f2eb65f8028df802f80204fc03693cfbf80228fc20f257bfa5afd3e4881bb07022d40e925a46287c843b10611b15fdc9cb56c44576f8028df802f80204fc03693ce9f80228fc20cd94d694bfd8596e37b6e37d8c242ebfcb902b814ab4747d84381daafb61002df8028df802f80204fc03693ce3f80228fc20ecd11a3cc8079cc6ea1815bda31d8216e1ee31b9054cbbbfde56c0b94aefe134f8028df802f80204fc03693cd3f80228fc2097cbe018f51b102c377c4f68d95b9c88a245f2c3d3b184ffc0bbf91bcb4a2217f8028df802f80204fc03693cccf80228fc200cabfe52245b20dfd44bfca033b96e25aedd909a6affb57ebcf6b0e4f0149826f8028df802f80204fc03693cf3f80228fc20d5cf6e311dbc549925cb3acb92bc228af22bb400b318b8798730c995bc0d6e46f8028df802f80204fc03693ce7f80228fc20cadaf766690d7025b3df72b73f1b9df3761a4f01051d50c7212c445d1452f439f8028df802f80204fc03693ce2f80228fc200098139b0bdfd6140a283c3e448d3e5a02b5cf73dd7043325940df058cfca00ef8028df802f80204fc03693ce0f80228fc2033fd2bc414bd0f8102a5caab7ff0fa58cf67ef082c56bbc331100ec7f0e4ec56f8028df802f80204fc03693cf4f80228fc202222901cd1d9d1f641bfeaa469a2694086593134af651fba0c0429d20ec0e006f8028df802f80204fc03693cdbf80228fc203ab4c19470c20f6c5d18ce82531444c7f65a54b2ddb48cf38134c27f25357359f8028df802f80204fc03693d19f80228fc20059d5efbc6f20f5b2e26d54651b36893666d4221897a04ae3508f2646c812526f8028df802f80204fc03693d12f80228fc20325fc157c778185f65d6c3c109b4ac20b4cc67b4e2f40f374fea6008ed792029f8028df802f80204fc03693d0cf80228fc202093b6d8950797ae96b56c2210ad3bebc8b75ff12c96925f2bbffed1cbbc8425f8028df802f80204fc03693d10f80228fc20a4a2761ad7c026bf02c86d393cbc6b511a41f53e6a8ac82ef8da188492974b0ef8028df802f80204fc03693d25f80228fc206c82d61171731cbc7bd4cd32082010589b50f1fb085cf56b07ff8f5c6ee19467f8028df802f80204fc03693d16f80228fc20d93890248265127c52736d5325fec9fe87fbed9b8760ce0bdc0812d64d109f65f8028df802f80204fc03693d2cf80228fc2063b8a1e3590dad47ac78a710ad74918b27c2fe7f386ea703e7034d4e3750d419f8028df802f80204fc03693d29f80228fc2069889a3b275e28e7124a3bf4e2984216410f08e96c9865ffac9ef97055178170f8028df802f80204fc03693d0bf80228fc202d5ab068ffc3817dc84d7b0a5eb404eceb94175edb32ce6479a74a69a4859b3af8028df802f80204fc03693d17f80228fc20d66cffbdd7d8f37ac155207003419b3de07582f27937f566af578388d8dd0e28f8028df802f80204fc03693d05f80228fc201f16d2191bb4f60603589effb19c4d54d3a989a1c001e6982df3fb76ae7dc940f8028df802f80204fc03693d14f80228fc20f067b95600721cb4fab2e5a0f180e7bc1df105ab425483bf7db5854e79c92624f8028df802f80204fc03693d1ef80228fc20f47b88b171b784108959687df4f187bf067a464374da5b7f9572a241ed087204f8028df802f80204fc03693d04f80228fc201a3f087755f5983fd44092dc1929cb9ec50e2b1d02d998aae895a675b87a841df8028df802f80204fc03693d13f80228fc20c620abe40ef0174b3233aba4703cc1eeedc6ca70802e7eb9ac9dac47f2673d7cf8028df802f80204fc03693d0ff80228fc20f6d3d019b287038aa4aaa078d17f696173a04faff6b32dfda0007f2391f0d078f8028df802f80204fc03693d2af80228fc205c65fa847e730506e6fd25b8b3195f0b66211d953c0c2eb46c5b1f4bad948043f8028df802f80204fc03693cfef80228fc20c59d62f302dcd2745dcbe596ebc437cd2eaae7354fc000adec1d26c7836e1e0df8028df802f80204fc03693d1bf80228fc2016444f1f473d8743977c6b3afa7ab5013aca6f3ad563597eaaaa044d5f6f2323f8028df802f80204fc03693d0df80228fc208582b9197ac9799294ad3577b8ff955f9fd7337337e73e4b7c02b300c269814cf8028df802f80204fc03693d24f80228fc203e63bfa38c5326f1591ab62e0e295524fd145499a5298dbe9bce98bad3fc9308f8028df802f80204fc03693d15f80228fc2060a12219765effeec1d354379c0dc1c28db6da81aa73582082184938f9295d0bf8028df802f80204fc03693d08f80228fc209b41cf7cd723a2ae9fb8bbdf7ef79479e86d6e2d55b086d1bcf7069ab7f43537f8028df802f80204fc03693d2bf80228fc20f8d2913cd042e672407a702a0ce2dd6ba2e0b820f06fa8c544c346e8d6d27d39f8028df802f80204fc03693d03f80228fc20540e41482bd0448af66b5edb06ca9b14abfddf0c158faa4d32ed78b0c7edb635f8028df802f80204fc03693d2df80228fc202ebb19d5bd61fed5d9a906fb2dda5ca54d4029b5a73dd30f63211e90156f1470f8028df802f80204fc03693d01f80228fc208a6c2de9b06f259a0658be444ce301c459c6fa3a49d6e9b5438043a5274d7743f8028df802f80204fc03693d23f80228fc20aed2af8b649273e9b1dc8b543414bc42246c69185b24d1fd1e22fa4791be360cf8028df802f80204fc03693cfff80228fc205ccebf48f787dbd487b1142541e53a185358e9c387c1600a8ca4040f4378a661f8028df802f80204fc03693d22f80228fc202e1c06717c62e5647b1502d9047fd57d895f2861e784953c92c5a91c8f4a157bf8028df802f80204fc03693d00f80228fc20a395d46acd1d8a9191892088adc611092db120946dab8944bd786a9f7c916e72f8028df802f80204fc03693d2ff80228fc2073fcc9a3dad570688b300e715cf2258c535f8a6163e7e2eec81bd68c58228b31f8028df802f80204fc03693d20f80228fc204795e7edeb0b89b55dac49de0a0cc6c4d3534af6695051420c0179d4a66b6943f8028df802f80204fc03693d18f80228fc20c82c004265e0b227fa3dcec5760b00fa2182a87c130d2f136e1685953fa8b403f8028df802f80204fc03693d1df80228fc202573c492e863c869aaf89dc8efc9f0e8a356b0c6f79b467e9ac5962607ca237df8028df802f80204fc03693d07f80228fc20ce8f769c9fa38038fc9ae993f3ac43907227b268731481f16a660e8fd9510e5ff8028df802f80204fc03693d11f80228fc20765c8af7046d4c5a9d33b11eb55475ca8821ad93a9b6f62de721143b6590a447f8028df802f80204fc03693d26f80228fc2040c016044e49867a3f9336d6be7509528005f2c1e710f244510480e741fdcd72f8028df802f80204fc03693d06f80228fc20926bf8f38903208488a972f1e2aecafe4ccc035cffc0905fa819cc77d8ced459f8028df802f80204fc03693d21f80228fc20fe66a19ee010d88c6a231e32880d7ed1c79f851f650c2da434145b6538aeb24ef8028df802f80204fc03693d09f80228fc20c17cb488d44638c5e68b46a1a822d12587806b65c8bf3d287f4d440adbea6267f8028df802f80204fc03693d27f80228fc20ca2ae6149735a18b551c3b7809266e8268f3eaed5deb246289ab897485acc071f8028df802f80204fc03693d0af80228fc20a185d3187c8cdc5ab2897dc5565bc636bb32abfc7b5b94b39b56a364b9b9477df8028df802f80204fc03693d2ef80228fc20c9583a853f797c4c43d0e4f95b6f1ed3d804e841751c9adad6d01afed2dfe062f8028df802f80204fc03693d1af80228fc20a992a534c116077db5513c5feb5618b1f3f553b13f325bd8452cb375e414574ff8028df802f80204fc03693d28f80228fc208e368b7824d596b1cff9b3f6c0861aca8faa03a572043950ea069694847f2223f8028df802f80204fc03693d02f80228fc200d9d77c09bcb9ea923e97eeaafef1f71ad2e1500f6ad8b9caa6e3d7af75a1462f8028df802f80204fc03693d1ff80228fc20f9b1b04b02eac47043533b1efc6c39d806149fd5e7abf22f0a61a8616a7b735ef8028df802f80204fc03693d0ef80228fc20c8dd17de91ec8420a6fda32706df5f21f513ac11e7225b7178ba71aa32a89449f8028df802f80204fc03693d1cf80228fc20684009cd6e71a28141feea75ff6b5a6c18180c5397c97c6f1aa343bb5e709666f8028df802f80204fc03693d4af80228fc20f13086bc698781e64956f083084bce61b12e6f7b0378e73fe78a45d3a4296f2bf8028df802f80204fc03693d58f80228fc20d9c41dec4626786e39cff2684424ac35403a23aaa414e1453268e8676bf58314f8028df802f80204fc03693d5bf80228fc20edccc14d3f4ee8c4935615ee48a3a5f159cd5497d690bb12d64a654618020235f8028df802f80204fc03693d34f80228fc200a96a28406a6d32afe9b72df9b336f0aad8289ba2082cea9eb05f43da67b205cf8028df802f80204fc03693d48f80228fc200d7521a921e348327a8221deed716da2b947a4bed9259c52cc77566022538963f8028df802f80204fc03693d3ef80228fc20d7d1cecf695a7df85ffa9406e55b8a7910cc3524cc86716742b310c2e30eaa39f8028df802f80204fc03693d43f80228fc20c51a2ecc7045392ee3c1b33a5b6953dfba041b981693a878af23b9e15f423f29f8028df802f80204fc03693d59f80228fc20a21397abcc0cafcbf332861dfba39de9a5f3b29ca130df095f48473f52929163f8028df802f80204fc03693d56f80228fc2031dd091ce6ef2eb3eca3c12aa035729bbf1e9c1026e73da9b9f9c5f83a4c775bf8028df802f80204fc03693d3ff80228fc20c37db266198240727b2e162d97b6c86239b343ae4b358c8c0246e0e45240b24ff8028df802f80204fc03693d31f80228fc20e0f218c90ad5dd48f25694ad12954a1a24fedf22f40934f22c00266ed5b25426f8028df802f80204fc03693d39f80228fc209fb2886227826b7d6fd05e7a105afa7bae5a289143f1da2a297221b658cbb008f8028df802f80204fc03693d57f80228fc20fdc35126bf6d5b5786dbae03d0cccfeb284687db505df8eace08e414bb6ee072f8028df802f80204fc03693d5af80228fc20ce3e21887c1aae1df3d009aa4ef45bba9e03ac684e4672d3b27374fb6d228616f8028df802f80204fc03693d4ef80228fc200bff4d63844073bc04f55f58badeb5ab6f64aade752ad5edf94579914ae39714f8028df802f80204fc03693d4ff80228fc20e9f8a7475a0efc8658a65c0bf14a33f470c135422f649cf7d61ff93896107d08f8028df802f80204fc03693d45f80228fc20864ef2c5f8ab88249f35e9714aecbc6504d8ff04b4e2e56714b01047594a1a3bf8028df802f80204fc03693d36f80228fc2016bdfbf500079103f5ed0f84459e8112b79c92c5f299945d69bde4c1ea059c2df8028df802f80204fc03693d5ef80228fc20bfbe82aeaba945bb1cb9e00f5753942d43271870bd32caea970edeb02d271b67f8028df802f80204fc03693d5cf80228fc20b82615a7488f5e4626a6caeda4ea38d70f89d6b49e6c301d725fe6fc37995c76f8028df802f80204fc03693d60f80228fc2052700ed8ad5d5c2069268535409fcf26befc4d4d866c27a295d2f336ca963c4bf8028df802f80204fc03693d50f80228fc206b2316bc71f78e99ad8d39489e3692b3dff08f812f6638b400a9f1976750f534f8028df802f80204fc03693d61f80228fc20e5144ed2df8f8e3555ee01bc15d73ffd74a14673f49898848fd611489e1f9f3bf8028df802f80204fc03693d3bf80228fc20603eae82eff49e1ae3606330374f3afc6f9e51072f36b8666b50fac3aa9d445ef8028df802f80204fc03693d51f80228fc20d39c9793fdbb2d38c27468d001cfb9aa07c15f56987d19522d1995a82a1cae51f8028df802f80204fc03693d54f80228fc20302c7b4ca284b14ca0198e19341aee7bd3f5862db01c0977fc7c88846946080af8028df802f80204fc03693d32f80228fc20ff354460ffcc275595bf8a28c7408852610cbc55c7f7a4dec5b6e0db137d4f02f8028df802f80204fc03693d3cf80228fc20321993fa57f442bc89d1fdbb648cd4c36948a82cf0d3ba5d5ab5f3f1675f7949f8028df802f80204fc03693d44f80228fc203dcf07aa5eae254c0863463592050da2a0fb620d5d6549c32aa548e0e41dcb74f8028df802f80204fc03693d47f80228fc20c332a1b4235708e1195921d81322a9d8f166ff25445628477cca97c5ffc5ba4af8028df802f80204fc03693d4bf80228fc20a6ceea386f859f4f55041f6f88e90076866ec6d7998a4c9ec4f5a5d8da3bdd15f8028df802f80204fc03693d3df80228fc206e91adea7fda65199c9d4986a6256fed8d763ad2eb5ac6e0f9fe7a427d9adc16f8028df802f80204fc03693d4df80228fc209bfcb569e9b06e390c65b4bae542045184dc9345a1ea8254e3db5841e28a5641f8028df802f80204fc03693d33f80228fc204236cbebcb1898c6d8dafa1d956c39bc9b802b168f28a22f155ee71ed9ca3c4df8028df802f80204fc03693d52f80228fc20c1074911be0267aaef73f089305e3744eee9ad4c0713a8648ceca499f2ada93ef8028df802f80204fc03693d35f80228fc2014ccea340bb986fb394b2bf3b7e0be466f90663398e6f05a490003886c276004f8028df802f80204fc03693d5ff80228fc203a869de8a61ef5a0620b15c3f8749eafb6082d8ea497fef024c17875fee9b417f8028df802f80204fc03693d49f80228fc20b45f2471277c9423e09ee0304774b6b75f88db6fea3c0ae86b144a7ad8577811f8028df802f80204fc03693d40f80228fc20d2810a1985f33fff192ccd3ab0d778a5c009589a07b913fe937f26259dd1ef37f8028df802f80204fc03693d41f80228fc206a340e7e52fd9beafbc575c8819d6ca34b8bb7137c0fd12aeedbcd8755d5136df8028df802f80204fc03693d53f80228fc20ca1bfedf6b3a6aad7c928152fd2ce55c72f37bb97c97751c550d94d492857147f8028df802f80204fc03693d46f80228fc2082847af3309944605c88fb28fa1df424f1ee1ee715345ccb93244b5a09fdc069f8028df802f80204fc03693d55f80228fc2033f597b2a8209ca330337c2c02652c3fe3fc67d7a87a3229fd48a6a1e6ec6316f8028df802f80204fc03693d42f80228fc20152263d32cfb69531372c2944ef8888007cd9a683ff1af31097af751eb30a208f8028df802f80204fc03693d38f80228fc20585c6ffdf4d520e26622a638cfbe5346fa61d8bee9f45e7b05b681cf6e260f70f8028df802f80204fc03693d37f80228fc205405f8681fdc1dd73a578d23714fd5bed08625bb8b8b543198779ee7d527817df8028df802f80204fc03693d62f80228fc206c073f3b50edb6202e5f38f78c7c12912ae7c17cca9e591ede59ac240337df24f8028df802f80204fc03693d4cf80228fc2088373442c8a298a9b3885ee37a84bd1479a95d0c1631687b04c9b7f383c6d63df8028df802f80204fc03693d3af80228fc20488bce29933f9ec3d2e7c825d42b384dacf458683a6cc71b6cb9dac5bdddcb7df8028df802f80204fc03693d5df80228fc2055a15be0fed65d12843d933fcd27ccddce56ccb0bb0ffdd0a75d770add4ed056f8028df802f80204fc03693d8cf80228fc2088da38670be2b908aecb44bbd4371f9845c7811b2fb67ee738774d5a90739419f8028df802f80204fc03693d7ff80228fc20422b3d773f91503851b96ef64c5f99cf181b030320349fa74ce470597fd37a5df8028df802f80204fc03693d7bf80228fc209c30ad258923e3a3d7a8e279d8261732d22a5945dc34c02cc2d921e0dd269761f8028df802f80204fc03693d8bf80228fc20dfb32ae7912887f315292decc9e6554bef08d14088444519fb87f0207512d92ef8028df802f80204fc03693d74f80228fc20e064907af70c34b8cc46270e47521a441a1216626536d11f42b73abb18159616f8028df802f80204fc03693d76f80228fc20d35b89abef59359db740f1fa7ea7523bfcd86a42f7cd8428e4e90f81d3693878f8028df802f80204fc03693d7df80228fc205e0a4d94e086f3a1b90b676b0ce172ba442c40ab9411bd389497bb43e7c9b346f8028df802f80204fc03693d8df80228fc2030b9c3f90138378886ced4286570c8a29ac2b1d84dee304a7d85d502e5072830f8028df802f80204fc03693d78f80228fc2050f726361f1e71846e0227b753da06243298737c792484be81f59aa430a11924f8028df802f80204fc03693d70f80228fc202cb97679f2e52487a2e519968f494dc1d9cd03bdaf843d15fc2df815c4fcbe6bf8028df802f80204fc03693d69f80228fc2062789817f4495376c67c2a59f96d15f3a1b1f96dd8b3d03633c7afdf2f191329f8028df802f80204fc03693d89f80228fc20b1d93db551ace62979aa7dc2b3da440960e81af0b163626d1cf8c5bea9db9309f8028df802f80204fc03693d6bf80228fc206e9c4c578b7a7970cf1eb8110559a6f6d0879ee8edb01480bf2ffe35929aea2bf8028df802f80204fc03693d6ff80228fc20a054337410abf0c640dc1a8ff12e8be1371597add25c431598f39d3599a3ea17f8028df802f80204fc03693d7cf80228fc206f89b94750053ce75b173301757dd56bff498cf118af40f25fe34ddfb39e0c2bf8028df802f80204fc03693d85f80228fc2039020444445f10c1414825043f582bbfd68c55add7c57320b2e9d0acc18e331bf8028df802f80204fc03693d66f80228fc20a7cea01ba621f78d4505502e7d57f29084bf28f1176e9c0a33963e9d15deb57df8028df802f80204fc03693d87f80228fc20a244cf1b9115ada146113f1c6d501c184b3a92bac27f7aa08ce674a49d87c127f8028df802f80204fc03693d80f80228fc20c8d10439b06d56bb4fd01cad77a6722711a410f7eed31570a1792798a5d8ef7af8028df802f80204fc03693d65f80228fc20cd73948836746c31b453656d4956c3ba123d6f3581da9542e4ba6f74d035b15bf8028df802f80204fc03693d6af80228fc20953ede6792f797889ebf322e0927ffe97cc1acb33f20e93fa3c52728cd206d41f8028df802f80204fc03693d93f80228fc20882e1a7f3d44f3437c960f92f0dfd8581699a4f1d72d724076f120c54adada34f8028df802f80204fc03693d68f80228fc20c82342226154c9f6db02549dec7fad544ab6b5e0ec06f8d4091a7090073c8f51f8028df802f80204fc03693d88f80228fc200d14982365e27d8e94ae69f6e846d7aecc3d05ff4b0596ebe01c9eba0e270d42f8028df802f80204fc03693d64f80228fc20a125c49b2bcf61fa7d66eaa3b69fc01dd7f76d5f31e6fa7c176d07edb843874af8028df802f80204fc03693d86f80228fc203bb7c80119037db46579139f48720c27482ae8a26cbe539660618d2b4ee42850f8028df802f80204fc03693d90f80228fc2088a0fd734a6f45b60f11d551c5023138f698f3352f163a57bd80ca09680f8259f8028df802f80204fc03693d82f80228fc201e2b61aec0390d7b838c7174a316e0db8dda59da260b3f767fd6459de3177535f8028df802f80204fc03693d77f80228fc207af266dea5e48876b79d57649e418bbd5a81ba82e155b0031018382db038944ef8028df802f80204fc03693d72f80228fc2060d1d8ba2a0c5f5531664d3d1eb24fb79f28034c95897b6cf6315c1def6e437bf8028df802f80204fc03693d75f80228fc208c127c07954afd2416784f395e3700aa9de270a1335fff783b5dc6b80bb5e415f8028df802f80204fc03693d6cf80228fc20de7e78b2279df254b6ca00aa02790b8e50a75472987caa6b0ae122d5e7f9715af8028df802f80204fc03693d95f80228fc20d7d4491be1e57246b403bcdb63f155032eab563479dc9c57c427e6d10c79f40df8028df802f80204fc03693d7ef80228fc20fd1839a05750e512fead6a93782e2f66e823e6c2f4dc9363c1300df0434c374df8028df802f80204fc03693d91f80228fc205b6382ddda48df2c222ab870ab3cfa1b1c3c8e1ddee4ff9b1d652b5578178725f8028df802f80204fc03693d6df80228fc2031be944ffecbd5d54bd9009c62933c0a3ceb4dfd80b2b9f77bb7db0eb4f68216f8028df802f80204fc03693d8ff80228fc20df85d0c9df421bd3bfc3271231c1bf2b9ec193abeacd1e197b0cfe0931a7c46cf8028df802f80204fc03693d6ef80228fc2036e3cca4048c3cd37f9bb8e65d45049a6b856c934bf04cd26df3fc19cd14727cf8028df802f80204fc03693d8af80228fc20f89cb31512b12df093c7bb575b2f9c3a427700072db0605b8c47f0d2933ce009f8028df802f80204fc03693d79f80228fc20047836d6d1eb16fdbfb66206863f9658c0fa5f034a7a4f957f07214d9fc7006bf8028df802f80204fc03693d94f80228fc207744df48ce6e64c217c2b7a7b3befb8205fd96f642f99cd0ca5ce93bc1821c12f8028df802f80204fc03693d71f80228fc200d410bc02c846fd6aa5a021e17611524c417f20d920891ca9c86ab590910f757f8028df802f80204fc03693d81f80228fc20f29b24199f0abda6ca32b8495d72faff76106ec5dd0f534497ff491a21546b4af8028df802f80204fc03693d67f80228fc206bc7965945e888aa500c817bcd0f49cdb1bca9f07a047c420b302476d65fe773f8028df802f80204fc03693d83f80228fc20ee8cf9c8676afeb182e301796f5dab97ebe3996a5dcf5bf2fdfc84260cd36e53f8028df802f80204fc03693d73f80228fc201fa077defdcf7980bba550161e087e2fc3d42a2983215c771f2286542ca58d1cf8028df802f80204fc03693d8ef80228fc2090a75e615a3fc8e952d7df2d931cd6c5740b1c16514b1aaf6322f483b578ef5bf8028df802f80204fc03693d7af80228fc208475c73ea6c057e7cf8709b458c420ba26a98419c8c30160a8f7c9d3c27e7e64f8028df802f80204fc03693d84f80228fc2000ced9fc6609347929b89d58e42db7a5c09361c260abd7fb0868f00a81ba7469f8028df802f80204fc03693d92f80228fc206c947db9e85028a02459de941b50d2cf87678620b6b1798163f543820eec701cf8028df802f80204fc03693dc8f80228fc2039592f28427dc6a984ff1a62624dc0a0f5bacc69cb9d65e93b63e883af5c6542f8028df802f80204fc03693db4f80228fc20a5ba7062c18bbd2b7949cd07e800c7fa5c758b84c827a949f5c50ba15114aa17f8028df802f80204fc03693dadf80228fc200a816be8f57f4b97e5bb5ccb8d1d6bfcee5007667bc1c97b6e86d612958dcd14f8028df802f80204fc03693da4f80228fc2033402f95767f3ec82d1fc54e98869c2e069616b3f53a004517e193b4a5f0fe61f8028df802f80204fc03693db7f80228fc2041c48e43fe965bc646ee98ad445920782a3e020fd6e698ea64b1aee154aa3d1ef8028df802f80204fc03693da5f80228fc2010e6fd0183a767283de0c7c60daf0e583fece69bacd1fa59b1352345d3deb774f8028df802f80204fc03693da1f80228fc20288a47931febf563a3d300a380907bbfecb478d5ebc7ab445177e5f5d9ae2117f8028df802f80204fc03693db5f80228fc20567ccf350a682b1dffa62bff4bc42011be75b9d6697fefe632541e922e95802ff8028df802f80204fc03693dbaf80228fc205bdefe9eb96925ed8f76034c47b7e41e5aa8507cab5f3e0b93946391c5d8993ff8028df802f80204fc03693dc0f80228fc20263c6df24e6a99acc48312788aa2046450e4ff538452b59c04a77a9a7e326c7af8028df802f80204fc03693d9df80228fc200c1a5a6d3acd25dc2a215a496a7b6e4bc030e660f7b6eea023edac9b95b49463f8028df802f80204fc03693d99f80228fc20397db3485b6aec18f26574c943d8830f388acd4af4b60caef3583ea1065faf33f8028df802f80204fc03693dabf80228fc201c47d370fc339554e950d8328232f2ed74726a0ba5de71c35f5470dbda84af0cf8028df802f80204fc03693db8f80228fc20e837e6baab1c1415f509d5dba936cd8d334edd93c914c046dbc25b68f992ab6af8028df802f80204fc03693daff80228fc20ab37ddb61a011b94525a253d718e1c9dcb8424345d4c122d1c65af99a7017878f8028df802f80204fc03693d98f80228fc20652a612acf3efec3fcb984ea15a3fc134a38d82e2c29bc15e9a25c889aa3412af8028df802f80204fc03693d97f80228fc201e22a9989ff7b3b9271684f22ee1429945e020d9e29a9cbdcd6edc893ba4d37ef8028df802f80204fc03693dc5f80228fc20541db886036fd136cadb8aa2a69b72ee46a9976e46395e1b2426fcadcc281550f8028df802f80204fc03693dc1f80228fc20302f8d1325c116a83bf0faeb21e524e687262c1c4dc4c215c700b4e773a77557f8028df802f80204fc03693db6f80228fc209435f8e971d3101f9621758ebd0ba8303f929a8a492ec28ceca95535a52b9e28f8028df802f80204fc03693dc4f80228fc20b4c0953280b624ddd1cdd074878856d12fb94e4d83eb5382fd88955799867a35f8028df802f80204fc03693dc2f80228fc2019b5b69e4c472fc8a6369b29abc8d021b47cd074f61bae87bbe6fdb0f6ecd76df8028df802f80204fc03693d9ef80228fc20ac842398bd05602e65c3cf996a1ecb08dbf60af5ae50f3f0f9d64c1b0c3e3b44f8028df802f80204fc03693dacf80228fc205b23bc7b929ed9e394ef7e9c07e2d4bf6067ed40e47b94c22b436f8534e3f768f8028df802f80204fc03693da7f80228fc207d2de87f12be5b16f556da83b351d5faaf083d5e3614c978be548855beed9540f8028df802f80204fc03693da3f80228fc20238ca185be3ffecd4a7bfabfa9200f5b535e8cabe0372d45122eb9870303500ef8028df802f80204fc03693da9f80228fc20de8f7ce6c63a19a711f21b3072d07509d788126ebe722e979ff9c7a46f3e8238f8028df802f80204fc03693dc7f80228fc208ecb5ce441fceae8224a54765b79068a7eea475aa433cec2c74b247b2a45b432f8028df802f80204fc03693dbbf80228fc20bb1cd483eeb3c664a3fb7df8db9bd27334657b79b3333cc41a51c8492733782bf8028df802f80204fc03693da6f80228fc207e12d3badbc7423cfce8baca0310ea882e6653c2c5dead313a41428b50299f1ff8028df802f80204fc03693da8f80228fc20bc78c5ea4560273f64c552ac6bef07e15fca771d7cc4ed6a85ba2a067d0e6f31f8028df802f80204fc03693d9bf80228fc20923830815cbf6e23784f9cbde27fba5a5006a9f7d7ee2f16d3dcb920065db46ef8028df802f80204fc03693db2f80228fc2014b95bbf2d63deeb99f45fea36de412e7db21c5fb8424ce1b4ba797bb924e340f8028df802f80204fc03693daef80228fc20ddb2d590a3c41e8e5cb281f8427682d3ec2fec29cd980e14cc7e077b604efc3ff8028df802f80204fc03693dc3f80228fc20b2fa6c2bf0567a3a24ee5a3830d59da314b89ab03f7dd3ac7b8e245b2c31c633f8028df802f80204fc03693d9ff80228fc206b33fa75f64c9ad7a539e6cdd20b0338a33b67af6c7013b9fe688c3c550bde19f8028df802f80204fc03693d9af80228fc209bc0b147b35c68eb2e1ff350c42c6a98acf58d9dc7da575fa7a2e2c331105a21f8028df802f80204fc03693dbef80228fc20d4e74527689dd054b627860a0e4dcaa6b644b265bc29390fb9f4da201b81f51df8028df802f80204fc03693dbff80228fc207e528e597f7a2696ed1e7c00055895f6d601e6e44d0a6f851de1f2b5d2a6bb7cf8028df802f80204fc03693db9f80228fc2041196b9aaa239523a53ab1079a43c99d9b0b021384b0cca4286234766c124723f8028df802f80204fc03693db1f80228fc20f1992bdef7bd6f3014dcd7c3a511bdde2248ae4f6a2d835779dfdccb02c8a237f8028df802f80204fc03693daaf80228fc20f70a6255df0fbae29cb00fadaeb7c3c0686289a51a027a272a24894364208e5bf8028df802f80204fc03693da2f80228fc205ae17b81ad999b80d3b76d37a8d01b6e792c974480e9309060d0651f346d8523f8028df802f80204fc03693dbdf80228fc2037e657c3dc54f881cb1980d955e7ce72ffeb66309f3ee50af508be9074b3673df8028df802f80204fc03693db0f80228fc20075351c87fddbb0f3fde740a9819f30e19a44949e5d770684b05b65ab9297978f8028df802f80204fc03693da0f80228fc200771e3c0a1f4b324edd41e4db42cd952761e544c788bcd89c62f9ebbd7b3124ef8028df802f80204fc03693db3f80228fc204dfbc89c7f0e851a24b268fe880322bd9d69e1c46d8a62ae785cd0647c8d135df8028df802f80204fc03693dc6f80228fc2072e43d98f9fd4cb68497e912f19e5213b31a530830831463727dc6d574d2a941f8028df802f80204fc03693d9cf80228fc201ccc4abc4d8a086609a36676ac70b88239de67e55bdf4386b78df73d7a44b02df8028df802f80204fc03693dbcf80228fc206b173e023e336436dbcbea2d983faaf1b0a7697662844e048cb5c59afb6bb63af8028df802f80204fc03693deaf80228fc20743c6809ba72792a130f25ca5ea738c14def724665267c549722d81bc6627f7cf8028df802f80204fc03693dfaf80228fc209354715eaf6a49bd0b7a72ccd35777e5fcd275ae7134a6fde0a830b005329d27f8028df802f80204fc03693dd3f80228fc20f011afdd865bd58b764a780a8fb11e87125b92ae715c770d8c35d284aa6a012bf8028df802f80204fc03693dd1f80228fc20733e17dc333f3aed008c8a229bbdc8b3d412558be5efbeb7ceed0cf289ada250f8028df802f80204fc03693dd6f80228fc2031e85b8b699ed75b17ebf1983cc7d82410e5182b7fa3696e6ee4b4c4d6c40777f8028df802f80204fc03693df4f80228fc20a5c125fc8cabe9db5b8afdd545b829fba7ff52b3a84a9c07b9c1a5932eac7277f8028df802f80204fc03693dcaf80228fc20a89d8265477aa41ec4a8facbf0dd228dec766e785750e4f75f5f91c027b4935ef8028df802f80204fc03693deef80228fc20a4f525770ecca035e9d3f0c481562f886e1bb700810f49e170b5de3c69dcd260f8028df802f80204fc03693de0f80228fc20ef7eff1b535d4b02f475e67192e619a21634616d671247ecb39e3799319c3e47f8028df802f80204fc03693dd4f80228fc20c048f2f7c8460622233ac4ef35c11dd17d308e41122d9b1f93669e424eab2f1ef8028df802f80204fc03693df1f80228fc209b2230a28463070b314c34e81a99f437e9ed09edd6f6a7f04a8129fecd8a9a44f8028df802f80204fc03693df0f80228fc203f25f735e0a91caddcb85be6d072e9cbf5368ef9a9e504df902c33d577a5311ef8028df802f80204fc03693de8f80228fc20cf17be2af46e2f1dcc4fe47eb5a8fd0b391d9b6db23be5ebe48332a6ab3d330af8028df802f80204fc03693dccf80228fc20d99124815c23f9df43f0f5c8d9f7a2d60cf3f56052a103906bf1baea4e17c850f8028df802f80204fc03693de9f80228fc20bd1493b35b408aa843e3674f4accb1599f4021fc5eccac8faf3bea2c8bcf7254f8028df802f80204fc03693debf80228fc20f4b0e590d075e364b55e41217a335386008a0633d5d44681424e6faf7a513d0ff8028df802f80204fc03693decf80228fc2021a21dc53fcd54d4796412bf15cbafb0b583176b38dc42fe6a50702fee2ea265f8028df802f80204fc03693df6f80228fc2088cf91ad4a0529e18ce42222ed974fd6f237e0b585c65877bc1c9fc9b15a4752f8028df802f80204fc03693df7f80228fc20f541d4c0d3e88324db85c557a82052a9b21030a9cbb024c22826c7457d553158f8028df802f80204fc03693dddf80228fc20a0f9b6dd761e1733bceb74adc6b8c8c9bcc836158796926a52ad2186f0ced206f8028df802f80204fc03693dd2f80228fc206d5b17cfb3e7be102b18da5a46f324ba35d2acf51af7d229fbf0cb9c1b8b7b1df8028df802f80204fc03693de1f80228fc207b73ec80f0021bea0b63415f00b336045f7fae1b5feef94bd0eed3b5c26d462ff8028df802f80204fc03693df2f80228fc20723dba9e2c2ebfadc99e4e86d424b3ca342289277c05ae981bf49596c274053ef8028df802f80204fc03693dd7f80228fc20efd5d47afcd949bcc26bdc67a54028a4a8668cae1dbe4221ec0a2683e184e82bf8028df802f80204fc03693dedf80228fc2057c3bcf5b55cd4ad23e49221a4efaa8ee625141a872e24acc6604cb40c8bf64df8028df802f80204fc03693de6f80228fc20a2c3c03414cfc0ffd8acb5ee3229f5eb182ad42f3007f2de3599e21c2c4b1906f8028df802f80204fc03693df5f80228fc20908f75697cef30a87ef0ee50780bc74ee254eb04654447d0be3d9dbb05985a7bf8028df802f80204fc03693ddbf80228fc20b6c16e6f133ca37f5753053fb76ff87c8e8ea55ae139aa51edeec84c7f40bf1af8028df802f80204fc03693dcff80228fc20c96185df3bf10070148e84583de8cfc9a74f152c31c84a81e5395d5807007531f8028df802f80204fc03693deff80228fc2060b8a72e4cc1390d6359b2e9ed4d8953a5ab8721b9cd5b5f18cb31104b13f123f8028df802f80204fc03693de4f80228fc208e109205a7f8150f7386adcfe47284845e0129da5d0f0e50f873063329ec8e5df8028df802f80204fc03693ddcf80228fc20d31d6a9032560de076891ce4e1a97f2f324efd3b20c0a02e03142681da123b5bf8028df802f80204fc03693df8f80228fc20a1cd09acad13fd6b17957f4466ec03465768cf8534326124a39fc9292102c25df8028df802f80204fc03693dd5f80228fc20098e6acbee04194a5d76f62f4791bc621445334b813f64082310c9908e012778f8028df802f80204fc03693de5f80228fc20925667c3245b12f8c5f506e039b39b3afad6b20c19e9c23b4f5f163d2c0fa53ef8028df802f80204fc03693dd0f80228fc20f1658013e214a0414778a7d97ae9a8974c36e7e6006f2410da63ee471c167f57f8028df802f80204fc03693dfbf80228fc2004d0cbfdca51480d3bf5591e8ee447b1caa0542f08c5faabb3fbeff69907f211f8028df802f80204fc03693de7f80228fc205b2cc5dfb6030eddf45b29b53d1dd17628b027b32dfc79404b7ed5822cdbe67ff8028df802f80204fc03693dd9f80228fc20494a779cb7d735de75baae56bb5bb51000ada192787785c51e65c16816ce6a79f8028df802f80204fc03693df9f80228fc20e3336932c87b8dd7a53e61b9212e306cadff3a24c69d8aa7212d1fdfb1b02846f8028df802f80204fc03693df3f80228fc20d64b008286a0259a8667d8a50af8cc3629d54052460628ede456cbb96a45d205f8028df802f80204fc03693ddff80228fc20e8b15461a58e6f0fc7c920e5d02f400d6f42a62e316f4c359ef3ec1b08ccf632f8028df802f80204fc03693de3f80228fc202c9324c894771cb25c5af52edb4807ceb12bcd1001234036e069268437344320f8028df802f80204fc03693dcef80228fc20bacd87370fc257f018e9db3a27290c77789eab705c6eb2a81a7618d8fc8d111bf8028df802f80204fc03693ddaf80228fc2053ce84745a5cd2c249c6867290cd7df941f38b1978184e8062bad90bf488e370f8028df802f80204fc03693dcdf80228fc2062a77c84cc88e81bbd4460dc28a5cbff66ad84ad3e9d3c31f3f1154d39901b1af8028df802f80204fc03693dd8f80228fc20b2e847f90a0526f4c9e865c56ab22858ab72c44bbad7a85bc123fd6b1b85fe71f8028df802f80204fc03693de2f80228fc2053175c64e9ece023fb1c6fa888982de8d1c6f55ea6c6ffa76659b7eda7411e2ef8028df802f80204fc03693ddef80228fc20a5f5898123383ad3bc25f2e0b930754e3159f8a80e1d0822d7e32c21154bdb6ff8028df802f80204fc03693dcbf80228fc20ef8cc155387e69800935ef10bc0d2bdc7be0025460df7856df562329ccf31c7cf8028df802f80204fc03693e0cf80228fc20598c9922cff8c471a291fdeee1053d1fe8d42b698a8c82c75f17dc9ff6326e52f8028df802f80204fc03693e1bf80228fc20f71c44e753c0f09e0f893c08dcef5e8a399d6c697f7faf6d9872e1f4f498b77af8028df802f80204fc03693e11f80228fc20fb4e9acc638a7fbc3b9335f611bdebe5ddea4ae286bb4d9afba96e76f9910346f8028df802f80204fc03693e07f80228fc2050df1550f50c41629149e9e20102352ce4d9f40e7d50d0efaf48ee128e571b2ff8028df802f80204fc03693e13f80228fc204c85ddc7c366f1ce25bb42fb4341691ddedf2ceca3c7ff2b4e99b65a00aaf700f8028df802f80204fc03693e09f80228fc2018b98b9df7bf534c77aa72da0d27bc2405a5eaaf9bbc5fe20a39eb306365ea41f8028df802f80204fc03693e2ef80228fc20fcfd899ab3f6f146db0e3f177a49d06c804eedfbdf24e6bff1f82e45774d200df8028df802f80204fc03693e20f80228fc202cfd3b4d13c17e29987a3504566bc332e4814d24ee0eb0d8c8e867303d268835f8028df802f80204fc03693e1ff80228fc20592a759e3e7e7a5d7a6dbfc6e208d7bb100994196c3b616ecb4476aa08cf224df8028df802f80204fc03693e1af80228fc20c608333e1ee275ca0a22273074f6f3487318c864ab2dc1f5abd955b52831ca05f8028df802f80204fc03693e02f80228fc20c9b53f694c5933d020f97411bb98eb38f5671da9ad8f2eb2cf280d57ea633000f8028df802f80204fc03693e04f80228fc2037bc457279428b72efb17a91d9feb623a4d2f24a2487828bbfb2b63f74df3d0ff8028df802f80204fc03693e28f80228fc203c39b1b7d5b0f591e9a98c4cdfbcc0546d7a29093254a5ec84fc490905949773f8028df802f80204fc03693e2af80228fc20e2527542825db210226e5cba89f445757ff0ce9b3c5d881f946cdcac2d5ceb4df8028df802f80204fc03693dfff80228fc2081aca4261448371c1857c806902deb51515b20c28fd721570d90894723168851f8028df802f80204fc03693e12f80228fc20ae994d256daa803fa6eb3c9b308d7921e4e4cf44c1b4613c98f2de72e5df3708f8028df802f80204fc03693e29f80228fc20e0933afd74dcd624be0543d2ad3a25e7419055dc06832857e8049193360b0c60f8028df802f80204fc03693e17f80228fc205bac33447623942342049c2f1674cc9aed9d69cd4a7eed62dc53ac18658b5d78f8028df802f80204fc03693e03f80228fc20bc0dc9cd68afef5248351fca6e174a8c73d35eb5d97507e285284f87986ceb11f8028df802f80204fc03693e14f80228fc20ea68c569b168d044298837b75326040ff967650f0626895a1c79aed30576453af8028df802f80204fc03693e05f80228fc20956f9a21901e933b2b4e592b940652c6c0e3708513f0a6557a48895fdbec384cf8028df802f80204fc03693e00f80228fc20a374dcb6c39c230af5062cb276cd874445a780cb76e60e10f543bbffcd5e206df8028df802f80204fc03693e21f80228fc20408121bf2a31a97ba09eb95f1a81b66e353b1c17b8ce73b93f5594f32f83da2ef8028df802f80204fc03693e2df80228fc20af28d3f4b4fc25ad60d171f402a98a242aa71fcfeb83bc572c783b4df9fe2f31f8028df802f80204fc03693e2bf80228fc20c6e687fe547ad92410fe474af88f7f24fb6474fee4c911e0b6965d532743386cf8028df802f80204fc03693e06f80228fc20a63cb66a584e3a517beb289167f2d44ea92f78b544d4f50ca9687c280a9f8150f8028df802f80204fc03693e24f80228fc208777b958cdd2c1a3d16f50edaee3022f96e5eeccea02dc51a2dd59f7fb09200bf8028df802f80204fc03693e15f80228fc20bd05e8dc3b89aa439d45c2fc9d490a7271c8bfbe45a53b371da69a52b0279d00f8028df802f80204fc03693e08f80228fc2037a9a47bcdbaeda4ee0fa2fe8049303b74f1588d020e9d71a10e88bbf2b1a559f8028df802f80204fc03693e26f80228fc205853cccc4005cf7f0a51325bdad33d0ff44a3f5cdb07ffcad12188789aaa5e7ff8028df802f80204fc03693e0ff80228fc2096cceac7b135cff269eb911abef2b65ed38b028ae8e3727950b72eee6ab82220f8028df802f80204fc03693e0ef80228fc20612b884e94d3572a2cc268011cdac6f1e196ea2209ecde882bd15b8037a6a14ef8028df802f80204fc03693e1df80228fc2093b1736d04b6f261cff107036897f5294e6fd64007f79954fe4b0a4b1cf60c37f8028df802f80204fc03693e16f80228fc20cedd4adc3199ab37bb2056bdbe6003c9878ec51ecc202239ef4d99fcd67f8556f8028df802f80204fc03693e01f80228fc20d0c67a656b4e614865a2c82c65bca5e2b8ebbeb88cdcae89cf1bd9b2142cfb1ef8028df802f80204fc03693e0df80228fc20880328b93efce28b9106160cd69afe529cdb9b9b136458ce5bc317cd547d0a28f8028df802f80204fc03693dfef80228fc20be0c71af964fc04926a92df887d9a0a36526610f4409a3471bcdcb992e61651df8028df802f80204fc03693e1cf80228fc206ffa05c71145375d75c2551b18ca224e679dc8e5ff5726a4e003da60b022af59f8028df802f80204fc03693e1ef80228fc20beb2007ae1fa078ff89a8abcadb57ab382c6e1b835c530eff003d5a59ca66b08f8028df802f80204fc03693e2cf80228fc20134f08f3ee69835d1f3068419bc07ebeb076229911ddb93a65258129a7ddff26f8028df802f80204fc03693e19f80228fc20eae93154f4614d422443607ffaae93e2e1165e768a7fe0fe09618111886fe812f8028df802f80204fc03693e0bf80228fc209973c3f0551596c589ef59083945b6bda4111b5be1e8b6be00ccdf48aeec182af8028df802f80204fc03693e10f80228fc20b774477b6eac715c6a5cb4f17d47b20f22f8b8b84671cc32d3e93633ecda6819f8028df802f80204fc03693e23f80228fc20729c926860a71920cf7d99a80c85e896659729097ffb37a60c94a4c39e33980ef8028df802f80204fc03693e0af80228fc2066aa31b2a202f106d1183796b2f4d830c45382fa5acfae9891a1429432842f04f8028df802f80204fc03693e18f80228fc209d5cb75b499b4eceeb48e0428947a9827b7a794cdf1f6098343046d3336aa60ef8028df802f80204fc03693e22f80228fc2075ab4fad879b0674790613381475719a2afe570e2deac23d0b4117697a605213f8028df802f80204fc03693dfdf80228fc207230f71101a5409228988709a12d21aed0fd1ec8fe518f5605e9d0fea8f0cb3ef8028df802f80204fc03693e25f80228fc2070468f8882063d5ff6d1b7b797d2d4c0177a7619883b1950ba9b47eb6d3f5d3ef8028df802f80204fc03693e27f80228fc20e81ceeb49e23309eb491b3cdc08413064ff49141f08fa0815ac0bbd1fe6c2a5df8028df802f80204fc03693e43f80228fc2027fa07c7a8934eefc133f19187cca8fad4087a971a6c98e303c6c4144b8c9f01f8028df802f80204fc03693e59f80228fc201c5165b7497b6693aef775de32148682f48d70a67cd098cbe6b5d96d599e5c40f8028df802f80204fc03693e40f80228fc20f3e547130bf5b11ea9dfa1ba02beefe24db61b59c19f315767874344ea8cfc5ef8028df802f80204fc03693e4ef80228fc20cce1560e3bb25fd4a39ace4769e5b3285c279566ed157ed54a1a8e4ee21d2f31f8028df802f80204fc03693e3ff80228fc2028fa7300727ac076554e76446383d12011b75123448be7b91bd08e32d5a2104ef8028df802f80204fc03693e4bf80228fc20ccc49a2fda9d433767ca6f3766e3f15589158d31f72d7410d74825f9f20c792ef8028df802f80204fc03693e49f80228fc20790c0a7f8fdfe3c07b8885511daf6c2ffeeb78d0f2097251c4585bb2a663e921f8028df802f80204fc03693e4ff80228fc2047926b02d1db277134b3fe3ef44dc48d5e7148d7dd23218d985daa392f88a934f8028df802f80204fc03693e4df80228fc200234638df1d1a146e74310bb0477447526d3ac67ac1fd31e7da7b04a764b9432f8028df802f80204fc03693e3ef80228fc20a84072917bbbd08b57347e82d8576df5b84f3a9ca28a8deb4e58eec5de6f624af8028df802f80204fc03693e38f80228fc20c451c677f1d3469b368ef885880f32f2e9a146857601065cad43bf4f09ec3138f8028df802f80204fc03693e42f80228fc204b60d81de3ff4639fca0e9fd45f1b079a34198acf4a856a92f2b86299950b13ff8028df802f80204fc03693e50f80228fc20a94a9698ef23de24aaf2eddcdddd9f5e606bc5c707b2e780d757d5b6ae48f25af8028df802f80204fc03693e3bf80228fc20af681c9c14a7f9c2c83d6a04f0835672972ea9cc9893fd97fb6e0c5c4ce6b100f8028df802f80204fc03693e54f80228fc20710155806bacd0029e2e330736fb3765229d0e418ac45806c1d793bd710fb37bf8028df802f80204fc03693e58f80228fc20ef291036b536ad6dc632f5b331d517768c9069bb4607ce0e4a8149aec375aa30f8028df802f80204fc03693e5cf80228fc207c29d472119596227c8e7ef295d9a4a4e0b696a09007c5c2f12e0458db89fd50f8028df802f80204fc03693e30f80228fc20610c2e3c2ccb588ea3321f2d6b7162f0c5416ed94dd4a21f0cbb9e162bf26457f8028df802f80204fc03693e34f80228fc20d02cbd5f4b8f080d721fce1b1884ed34ff3216427f513d0483ef2eac2cfc6608f8028df802f80204fc03693e5bf80228fc20143ea76f6225fa0541305ee4078753aa8c0350f4ff569951806c796e91cc754ef8028df802f80204fc03693e32f80228fc2092f7d3da4f896b68bd8af706e3429311a3ed4932dba9e853cf50dd86a57b7b15f8028df802f80204fc03693e39f80228fc20818cb65a49b77183176a61dd88286d9f2c7efbcc250fcd34db08ecfdd508622ff8028df802f80204fc03693e53f80228fc2018e3c20f0ea230ade0965603ae4f7aa6d7db8db984440f85c57b4291a30a0f5df8028df802f80204fc03693e44f80228fc2018f1e5e622343a28cbd95b145236d302ca9617d99ff59d61e5dda49b4fb0cb7ef8028df802f80204fc03693e60f80228fc2025b1d2584fff177504e7d16683be6e565d9bbcae7983b138479ab8a864909d1df8028df802f80204fc03693e4af80228fc2087b79b8010d3d9ef7623ccfbffa5f1fa23a3cadb7af418b9415e5be657255b30f8028df802f80204fc03693e47f80228fc201367b452a1ba4275ae484a6d667c5f99834678201ebfc894076188e2774afa0cf8028df802f80204fc03693e31f80228fc209f6a9661257abd52282da0e7b2e3581468a02d6a439cf7e4ef7705d6d44f9c7bf8028df802f80204fc03693e48f80228fc20eca5b20862c868acdf9defc3b441a1d7d75ca8dffdce0b6965fd59c65fd14720f8028df802f80204fc03693e45f80228fc20ddaa5a79d32ba31a8d100cdc9e2e6492183582064cd7796993cd43e8dad9c737f8028df802f80204fc03693e33f80228fc2094928282c94675be3de6c821504af2abf0c7ec8716ee6579d9f9a06eace8bd35f8028df802f80204fc03693e35f80228fc20c2acc07367ef0cb4c075a9c2fa4d20cb80e98eb3ac06944263c4e7ef440a202bf8028df802f80204fc03693e55f80228fc206344953abe94cc7194a2ff1dab1df5b5130057197da127b6dacdee47a15d2446f8028df802f80204fc03693e36f80228fc20853a1680be4c715bb1b271f2b62bc3612ab9df31b4e7c5c59f1670afa7c65145f8028df802f80204fc03693e56f80228fc205f34177f89ef21f2a0a2bd83c1640436f2bef4d211944395d015b6cb36102e3ef8028df802f80204fc03693e57f80228fc206fe2ef95861b6effac3719485ba0b80fa41697fa9fd587831d4e4fe6d80a224df8028df802f80204fc03693e37f80228fc2087dde285e17499888aa82f295feeee9e3eed711abc2622f3a45a57b4970bb729f8028df802f80204fc03693e5ff80228fc20e87fe508e9b36575f732949e54dd382edc548981ac9e188e9ad2d0f885ba3100f8028df802f80204fc03693e52f80228fc20df5d5c07b14068fc51d96f832a9bb13dd8af9c085c8df230aa833294a9e79755f8028df802f80204fc03693e5ef80228fc20e3735ebcb36fda6521ebec321ee74495b57d186b4dc305e25b8d1920d9b19126f8028df802f80204fc03693e5df80228fc20020076be9bf6552c25f19c1d938235f16a2299f13c064ec56b229864355c6d66f8028df802f80204fc03693e3cf80228fc20ef055204fbd65a5ccd995f69cdbb3acfb57aa122edb6ca14bf91d7db19c22242f8028df802f80204fc03693e4cf80228fc205dd45eac99e5f0119027512ce20fbb12b70bd63412a33914e99ad3e42f89817ff8028df802f80204fc03693e3df80228fc20a84e3da76fbf9d6e49b3b0e0455f2ffe410e861224c3cbdf2daddd1de323fe51f8028df802f80204fc03693e41f80228fc200734860f2bfd07abe62dac82c9b37b15ff9726235033cb16a2362f1c822f3a34f8028df802f80204fc03693e3af80228fc203690cfb0996943a5f2539a1d366a6cb43f34d29ab44a9806efbcd3058a837918f8028df802f80204fc03693e5af80228fc205edf79bc37e95e7e7afced0f8ab9f3d4e781d96097448bc1d3a2f6fa4509c841f8028df802f80204fc03693e51f80228fc2029d997ae223afbd1b6df4f487f31acaa861d6a6ac6bac3b14b40f0877c4e9857f8028df802f80204fc03693e46f80228fc205eb60b197e8d59bd33536db9e44be22d5afe91c83d37be62186dc0ed5fecd71cf8028df802f80204fc03693e61f80228fc20e1753f3927d8225c225cdca59135c3ab6302c122aecf96548609b0b6e6ef2a11f8028df802f80204fc03693e69f80228fc2094d66caba6d63b9542c528c6e7bee81d62e41333a9ae5ced7ea053cec40f4033f8028df802f80204fc03693e67f80228fc20a6a0bb8bea367c7a2ba00937474e575ac7f7acdcdebe74ec283e48cf88251043f8028df802f80204fc03693e63f80228fc20bc4768c20960bd79511ad924439508a7647c03d34d29c2a5e0c75940d3bba355f8028df802f80204fc03693e66f80228fc20c4e833da6a7a40f106ecf5bd3e8efddf6b082a9c5d5807022d14603b8d77ab3af8028df802f80204fc03693e65f80228fc204ace6ca39a70887c2e11f08eec472a24c74403300e1e0580849769b982cb1925f8028df802f80204fc03693e6cf80228fc20676b59ef29e8b156c0ec6f13b1d5857feb09a9d1c28fc415e239c9c9c3465d46f8028df802f80204fc03693e6af80228fc206d5734cef1d4ea3eaff1a1404c891c8cdf9010165cde9e581b51f99aa47d762af8028df802f80204fc03693e6df80228fc2095c45fd1e46eca4c08773de951c1f604bb5f7cd42daf046bf0aabd96efbe1069f8028df802f80204fc03693e68f80228fc20f141f2a31d3e157f30cd45d3e33349aaa029c9ca9068e5aad194c10b697efb49f8028df802f80204fc03693e6ef80228fc209a55a774c4dedb0f27f96766f4b02acd9fbc2fe4b617531b52b3fbda90da7958f8028df802f80204fc03693e6bf80228fc204dffabd3bf1032fe698a839c6cc6a5f1fbe64a94a74d853caae090395d4ae932f8028df802f80204fc03693e64f80228fc208d872218c9109b5d23fca74e5a1d4326a43a5d1330a707ba50611ef6b6e98a29f802cff803f80204fc03000000f80228fc208a696212d51b1d4813188b940d1a4c8c8c1823966b3cb61adfafe636ba02fd0df802cefc4052de176f52323e92479b91e84ccf170670ad5f0206f5bfb0e112b6636250e7e868e01f11db996d648b7d8a50800725d3cfcebcf7974e4ebaa65e1331cd44710d")
	// save node to list
	i.SaveNode(iqId, keys)
	return keys
}

// BuildIqSetProfilePicture 设置头像
func (i *IqProcessor) BuildIqSetProfilePicture(pictureData []byte, to string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqSetProfilePicture(iqId, pictureData, to)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqSetProfilePicture time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqGetQr 获取二维码
func (i *IqProcessor) BuildIqGetQr() (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqGetQr(iqId)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqGetQr time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqNickName
func (i *IqProcessor) BuildIqNickName(name string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqNickName(iqId, name)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqNickName time out id:%d", iqId.Val()))
	})
	return build
}

// BuildSetQrRevoke 重置
func (i *IqProcessor) BuildSetQrRevoke() (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqRevokeQr(iqId)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildScanCode time out id:%d", iqId.Val()))
	})
	return build
}

// BuildScanCode 扫码二维码
func (i *IqProcessor) BuildScanCode(code string, opCode int32) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqScanCode(iqId, code, opCode)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildScanCode time out id:%d", iqId.Val()))
	})
	return build
}

// InviteCode 邀请code
func (i *IqProcessor) BuildInviteCode(code string, toWid string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqInvite(iqId, toWid, code)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildScanCode time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqMediaConIq 获取CDN
func (i *IqProcessor) BuildIqMediaConIq() (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqMediaCon(iqId)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqMediaConIq time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqGetProfilePicture 获取头像
func (i *IqProcessor) BuildIqGetProfilePicture(to string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqGetPicture(iqId, to)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqGetProfilePicture time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqGetProfilePicture 获取头像
func (i *IqProcessor) BuildIqGetProfilePreview(to string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqGetPreview(iqId, to)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqGetProfilePicture time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqCreateIqState 设置个性签名
func (i *IqProcessor) BuildIqCreateIqState(contact string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqState(iqId, contact)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateIqState time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIqCreateIqGetState 获取个性签名
func (i *IqProcessor) BuildIqCreateIqGetState(u string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIqGetState(iqId, u)
	/*build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	i.SaveNode(iqId, build)*/
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateIqGetState time out id:%d", iqId.Val()))
	})
	return build
}

// BuildIq2Fa 两步验证
func (i *IqProcessor) BuildIq2Fa(code string, email string) (build *IqNode) {
	iqId := i.iqId()
	//create
	build = createIq2Fa(iqId, code, email)
	//build.SetPromise(i.catchTimeOut(iqId, time.Second*30))
	// save node to list
	//i.SaveNode(iqId, build)
	i.SetNodeTimeOutRemove(iqId, build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildIqCreateIqGetState time out id:%d", iqId.Val()))
	})
	return build
}
