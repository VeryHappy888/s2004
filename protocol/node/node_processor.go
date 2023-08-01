package node

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/gogf/gf/container/gtype"
	"github.com/golang/protobuf/proto"
	"log"
	"strings"
	"time"
	"ws-go/libsignal/protocol"
	"ws-go/protocol/axolotl"
	"ws-go/protocol/define"
	entity "ws-go/protocol/entity"
	iface "ws-go/protocol/iface"
	"ws-go/protocol/msg"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/utils"
	"ws-go/protocol/waproto"
)

// MainNodeProcessor 主节点处理器
type MainNodeProcessor struct {
	*processor
	iq           *IqProcessor
	presence     *PresenceProcessor
	message      *MessageProcessor
	notification *NotificationProcessor
	call         *CallProcessor

	axolotlManager *axolotl.Manager
	msgManager     *msg.Manager
	handlers       iface.IHandlers
}

func NewMainNodeProcessor() *MainNodeProcessor {
	p := Processor(nil)
	m := &MainNodeProcessor{
		processor: p,
		iq:        NewIqProcessor(),
		presence:  NewPresenceProcessor(),
		message:   NewMessageProcessor(),
	}
	m.call = NewCallProcessor(m)
	m.notification = NewNotificationProcessor(m)
	return m
}

func (m *MainNodeProcessor) ProcessNode(node *newxxmp.Node) {
	//wslog.GetLogger().Debug("MainNodeProcessor ,process", node.GetTag())
	switch node.GetTag() {
	case NodeIq:
		_ = m.iq.Handle(node)
	case NodePresence:
		m.presence.Handle(node)
	case NodeReceipt, NodeMessage:
		m.handles(m.message.Handle(node))
	case NodeNotification:
		m.notification.handle(node)
	case NodeCall:
		m.call.Handle(node)
	}

}

// Close 关闭
func (m *MainNodeProcessor) Close() bool {
	m.processor.Close()
	m.presence.Close()
	m.handlers.Close()
	m.Reset()
	fmt.Println("---", m.sendQueue.Size())
	return true
}

// handles 处理通知 和 消息的解密
func (m *MainNodeProcessor) handles(is ...interface{}) {
	if is == nil && m.handlers != nil {
		return
	}
	// 对 processor 返回多个结果进行处理
	for _, i := range is {
		switch i.(type) {
		case iface.NodeBuilder:
			m.SendBuilder(i.(iface.NodeBuilder))
		case *entity.ChatMessage:
			handler, fund := m.handlers.GetHandler(define.HandlerChatMessage)
			if fund {
				_ = handler.AddHandleTask(i)
			}
		case *entity.Notification:
			// TODO 通知
		case *entity.Receipt:
			// change my msg status
			e := i.(*entity.Receipt)
			/**	if strings.Contains(e.ReceiptType, "read") {
				_ = m.msgManager.UpdateMsgStatus(e.MsgId, define.Read)
			}**/
			// 发送确认
			m.SendAck(e.MsgId, e.RecipientId, e.ReceiptType, ClassReceipt, e.Participant)
		}
	}
}

// SendAck 发送确认
func (m *MainNodeProcessor) SendAck(id, to, xtype, class, participant string) {
	m.SendBuilder(createAck(id, to, xtype, class, participant))
}

// SendReceiptRetry
func (m *MainNodeProcessor) SendReceiptRetry(to, id, participant, t string, count gtype.Int32) {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("SendReceiptRetry panic: %v\n", r)
		}
	}()
	more := make([]*newxxmp.Node, 0)
	if count.Val() == 1 {
		// registration node
		registrationNode := newxxmp.EmptyNode("registration")
		// uint32 to bytes
		regIdData := make([]byte, 4)
		binary.BigEndian.PutUint32(regIdData, m.axolotlManager.IdentityStore.GetLocalRegistrationId())
		registrationNode.SetData(regIdData)
		more = append(more, registrationNode)
	}

	if count.Val() == 2 {
		keys := newxxmp.EmptyNode("keys")
		// registration node
		registrationNode := newxxmp.EmptyNode("registration")
		// uint32 to bytes
		regIdData := make([]byte, 4)
		binary.BigEndian.PutUint32(regIdData, m.axolotlManager.IdentityStore.GetLocalRegistrationId())
		registrationNode.SetData(regIdData)
		more = append(more, registrationNode)
		// identity node
		identityNode := newxxmp.EmptyNode("identity")
		identityNode.SetData(m.axolotlManager.IdentityStore.GetIdentityKeyPair().PublicKey().Serialize()[1:])
		// set identity node to iq node
		keys.Children.AddNode(identityNode)
		// type
		typeNode := newxxmp.EmptyNode("type", []byte{0x05})
		keys.Children.AddNode(typeNode)
		// key
		preKeys, err := m.axolotlManager.GetPreKeys()
		if err != nil {
			return
		}
		if preKeys == nil || len(preKeys) <= 0 {
			return
		}
		keys.Children.AddNode(createPreKeys(preKeys[2]))
		// skey
		keys.Children.AddNode(createSignedPreKeys(m.axolotlManager.SignedPreKeyStore.LoadSignedPreKey(0)))
		more = append(more, keys)
	}

	m.SendBuilder(createReceiptRetry(to, id, participant, t, count, more...))
}

// sendNormalAndRead 发送确认和已读
func (m *MainNodeProcessor) SendNormalAndRead(id, to, participant string) {
	//m.SendBuilder(m.message.BuildNormalReceipt(id, to, participant, false))
	// 读消息
	m.SendBuilder(m.message.BuildNormalReceipt(id, to, participant, true))
}

func (m *MainNodeProcessor) GetPreKeysNumber(reason bool, us ...string) error {
	var needGetUsers []string
	// get pre keys
	// 过滤已经获取的
	for _, u := range us {
		if strings.Contains(u, "@") {
			u = strings.Split(u, "@")[0]
		}
		if !m.axolotlManager.ContainsSession(u) {
			needGetUsers = append(needGetUsers, u)
		}
	}

	if len(needGetUsers) == 0 {
		return nil
	}

	getIqUserKeys := m.SendGetIqUserKeys(needGetUsers, reason)
	_, err := getIqUserKeys.GetResult()
	if err != nil {
		return err
	}
	//fmt.Println(result)
	// set session
	/*if iqResult, ok := result.(entity.IqResult); ok {
		for _, keys := range iqResult.GetPreKeys() {
			// TODO 应该处理下这个错误
			err := m.axolotlManager.CreateSession(
				keys.GetJid(),
				keys.CreatePreKeyBundle())
			if err != nil {
				return err
			}
		}
	}*/
	return nil
}

// GetPreKeys
func (m *MainNodeProcessor) GetPreKeys(reason bool, us ...string) error {
	var needGetUsers []string
	// get pre keys
	// 过滤已经获取的
	for _, u := range us {
		if strings.Contains(u, "@") {
			u = strings.Split(u, "@")[0]
		}

		if !m.axolotlManager.ContainsSession(u) {
			needGetUsers = append(needGetUsers, u)
		}
	}

	if len(needGetUsers) == 0 {
		return nil
	}

	getIqUserKeys := m.SendGetIqUserKeys(needGetUsers, reason)
	result, err := getIqUserKeys.GetResult()
	if err != nil {
		return err
	}
	// set session
	if iqResult, ok := result.(entity.IqResult); ok {
		for _, keys := range iqResult.GetPreKeys() {
			// TODO 应该处理下这个错误
			err := m.axolotlManager.CreateSession(
				keys.GetJid(),
				keys.CreatePreKeyBundle())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Reset 重置所有 processor
func (m *MainNodeProcessor) Reset() bool {
	m.iq.reset()
	return true
}

// =============  settings ==================
// SetAxolotlManager
func (m *MainNodeProcessor) SetAxolotlManager(axolotlManager *axolotl.Manager) {
	m.axolotlManager = axolotlManager
}
func (m *MainNodeProcessor) SetMsgManager(manager *msg.Manager) {
	m.msgManager = manager
}

// SetSegmentOutputProcessor
func (m *MainNodeProcessor) SetSegmentOutputProcessor(outputProcessor iface.SegmentOutputProcessor) {
	if m.processor == nil {
		m.processor = Processor(outputProcessor)
	}
	m.processor.segmentOutput = outputProcessor
}

// SetHandles
func (m *MainNodeProcessor) SetHandles(hs iface.IHandlers) {
	m.handlers = hs
}

// =============== node api ==================
// SendGetIqUserKeys 获取指定用户的keys
func (m *MainNodeProcessor) SendGetIqUserKeys(users []string, reason bool) iface.NodeBuilder {
	// create builder
	iqUserKeys := m.iq.BuilderIqUserKeys(users, reason)
	m.SendBuilder(iqUserKeys)

	return iqUserKeys
}

// SendIqConfig 登录成功后发送 1
func (m *MainNodeProcessor) SendIqConfig() iface.NodeBuilder {
	builderIqConfig := m.iq.BuilderIqConfig()
	m.SendBuilder(builderIqConfig)
	return builderIqConfig
}

// SendIqConfigOne 登录成功后发送 1
func (m *MainNodeProcessor) SendIqConfigOne() iface.NodeBuilder {
	builderIqConfig := m.iq.BuilderIqConfigOne()
	m.SendBuilder(builderIqConfig)
	return builderIqConfig
}

// SendIqConfigTwo 登录成功后发送 2
func (m *MainNodeProcessor) SendIqConfigTwo() iface.NodeBuilder {
	builderIqConfig := m.iq.BuilderIqConfigTwo()
	m.SendBuilder(builderIqConfig)
	return builderIqConfig
}

// SendIqPing 间隔发送
func (m *MainNodeProcessor) SendIqPing() iface.NodeBuilder {
	builder := m.iq.BuilderIqPing()
	m.SendBuilder(builder)
	return builder
}

// SendGetVerifiedName
func (m *MainNodeProcessor) SendGetVerifiedName(jid string) iface.NodeBuilder {
	builder := m.iq.BuilderGetVerifiedName(NewJid(jid).Jid())
	m.SendBuilder(builder)
	return builder
}

// SendCategories  商业版获取类型列表
func (m *MainNodeProcessor) SendCategories() iface.NodeBuilder {
	builder := m.iq.BuilderSendCategories()
	m.SendBuilder(builder)
	return builder
}

// SendBusinessProfile  商业版本设置信息
func (m *MainNodeProcessor) SendBusinessProfile(categoryId string) iface.NodeBuilder {
	builder := m.iq.BuilderBusinessProfile(categoryId)
	m.SendBuilder(builder)
	return builder
}

// SendBusinessProfileTow 商业版本设置信息2
func (m *MainNodeProcessor) SendBusinessProfileTow(u string) iface.NodeBuilder {
	builder := m.iq.BuilderBusinessProfileTow(u)
	m.SendBuilder(builder)
	return builder
}

// SendPresencesSubscribe 发送消息时候发送
func (m *MainNodeProcessor) SendPresencesSubscribe(u string) iface.NodeBuilder {
	build := m.presence.BuildPresencesSubscribe(u)
	m.SendBuilder(build)
	return build
}

// SendActive 上传密钥时候发送
func (m *MainNodeProcessor) SendActive() iface.NodeBuilder {
	build := m.iq.BuildIqActive()
	m.SendBuilder(build)
	return build
}

// SendEncrypt 发消息用到
func (m *MainNodeProcessor) SendEncrypt(u string) iface.NodeBuilder {
	build := m.iq.BuildEncrypt(u)
	m.SendBuilder(build)
	return build
}

// SendPresenceAvailable 设备激活 登录成功发送
func (m *MainNodeProcessor) SendPresenceAvailable() {
	buildPresenceAvailable := m.presence.BuildPresenceAvailable("")
	m.SendBuilder(buildPresenceAvailable)
}

func (m *MainNodeProcessor) SendBusinessPresenceAvailable(name string) iface.NodeBuilder {
	buildPresenceAvailable := m.presence.BuildPresenceAvailable(name)
	m.SendBuilder(buildPresenceAvailable)
	return buildPresenceAvailable
}

// SendCreateGroup 创建群聊
func (m *MainNodeProcessor) SendCreateGroup(u, subject string, participants []string) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqCreateGroup(u, subject, participants)
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// SendGetGroupMember 获取所有群成员
func (m *MainNodeProcessor) SendGetGroupMember(u string) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqCreateGroupMember(u)
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// SendGetGroupCode 获取二维码 code
func (m *MainNodeProcessor) SendGetGroupCode(u, groupId JId) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqGetGroupCode(u.Jid(), groupId.GroupId())
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// CreateGroupAdmin 设置群管理
func (m *MainNodeProcessor) CreateGroupAdmin(u, groupId, toWid JId) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqSetGroupAdmin(u.Jid(), groupId.GroupId(), toWid.Jid())
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// CreateDemoteGroupAdmin 取消息群管理
func (m *MainNodeProcessor) CreateDemoteGroupAdmin(u, groupId, toWid JId) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqSetDemoteGroupAdmin(u.Jid(), groupId.GroupId(), toWid.Jid())
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// CreateLogOutGroup 退出群组
func (m *MainNodeProcessor) CreateLogOutGroup(u, groupId JId) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqLogOutGroup(u.Jid(), groupId.GroupId())
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// SendPresencesSubscribeNew 发送订阅
func (m *MainNodeProcessor) SendPresencesSubscribeNew(u string) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildPresencesSubscribeNew(u)
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// SetGroupDesc  设置群描述
func (m *MainNodeProcessor) SetGroupDesc(u, groupId JId, desc string) iface.NodeBuilder {
	buildIqCreateGroup := m.iq.BuildIqGroupDesc(u.Jid(), groupId.GroupId(), desc)
	m.SendBuilder(buildIqCreateGroup)
	return buildIqCreateGroup
}

// encryptSenderKeyDistributionsBroadCast --发动态
func (m *MainNodeProcessor) encryptSenderKeyDistributionsBroadCast(u, groupId JId, participants []string) (map[string]protocol.CiphertextMessage, error) {
	cs := make(map[string]protocol.CiphertextMessage, 0)
	// 获取没有保存在数据库中的 keys
	if err := m.GetPreKeys(false, participants...); err != nil {
		return nil, err
	}
	// create group sender keys session
	senderKeyDistribution, err := m.axolotlManager.CreateGroupSession("status@broadcast", u.RawId())
	if err != nil {
		return nil, err
	}
	// encrypts
	senderDistributionSerializeData := senderKeyDistribution.Serialize()
	log.Println("encryptSenderKeyDistributions senderDistributionSerializeData:", hex.EncodeToString(senderDistributionSerializeData))
	// wa message pb
	pbData, err := waproto.CreatePBWAMessageSkMsg("status@broadcast", senderDistributionSerializeData)
	log.Println("encryptSenderKeyDistributions pbData:", hex.EncodeToString(pbData))
	if err != nil {
		return nil, err
	}
	for _, participant := range participants {
		jid := NewJid(participant)
		cMessage, err := m.axolotlManager.Encrypt(jid.RawId(), pbData, false)
		if err != nil {
			return nil, err
		}
		cs[participant] = cMessage
	}
	return cs, nil
}

// encryptSenderKeyDistributions
func (m *MainNodeProcessor) encryptSenderKeyDistributions(u, groupId JId, participants []string) (map[string]protocol.CiphertextMessage, error) {
	cs := make(map[string]protocol.CiphertextMessage, 0)
	// 获取没有保存在数据库中的 keys
	if err := m.GetPreKeys(false, participants...); err != nil {
		return nil, err
	}
	// create group sender keys session
	senderKeyDistribution, err := m.axolotlManager.CreateGroupSession(groupId.RawId(), u.RawId())
	if err != nil {
		return nil, err
	}
	// encrypts
	senderDistributionSerializeData := senderKeyDistribution.Serialize()
	log.Println("encryptSenderKeyDistributions senderDistributionSerializeData:", hex.EncodeToString(senderDistributionSerializeData))
	// wa message pb
	pbData, err := waproto.CreatePBWAMessageSkMsg(groupId.GroupId(), senderDistributionSerializeData)
	log.Println("encryptSenderKeyDistributions pbData:", hex.EncodeToString(pbData))
	if err != nil {
		return nil, err
	}

	for _, participant := range participants {
		jid := NewJid(participant)
		cMessage, err := m.axolotlManager.Encrypt(jid.RawId(), pbData, false)
		if err != nil {
			fmt.Println(jid, err.Error())
			continue
			//return nil, err
		}
		cs[participant] = cMessage
	}
	return cs, nil
}

// SendAddGroup 邀请成员
func (m *MainNodeProcessor) SendAddGroup(groupId string, participants ...string) iface.NodeBuilder {
	buildIqAddGroup := m.iq.BuildIqAddGroup(groupId, participants...)
	m.SendBuilder(buildIqAddGroup)
	return buildIqAddGroup
}

// SendTextGroupMessage 发送群消息
func (m *MainNodeProcessor) SendTextGroupMessage(veriFiledName uint64, u, groupId JId, content string, at []string, stanzaId string, participant string, conversation string) (*msg.MySendMsg, error) {
	var participants []string
	// get group numbers
	iqWg2Query := m.iq.BuildIqWg2Query(groupId.GroupId())
	m.SendBuilder(iqWg2Query)
	// wait result
	result, err := iqWg2Query.GetResult()
	if err != nil {
		return nil, err
	}
	// iqResult
	if iqResult, ok := result.(entity.IqResult); ok {
		groupInfo := iqResult.GetGroupInfo()
		for jid, participant := range groupInfo.Participants() {
			// if is supper admin
			if xtype, ok := participant["type"]; ok ||
				strings.Contains(xtype.String(), "superadmin") {
				if u.Jid() == jid {
					continue
				}
				//u = NewJid(jid)
			}
			participants = append(participants, jid)
		}
	}
	// encryptSenderKeyDistributions
	cs, err := m.encryptSenderKeyDistributions(u, groupId, participants)
	if err != nil {
		return nil, err
	}
	var preview = waproto.ExtendedTextMessage_NONE
	w3 := &waproto.Message{
		ExtendedTextMessage: &waproto.ExtendedTextMessage{
			Text:        proto.String(content),
			PreviewType: &preview,
			ContextInfo: &waproto.ContextInfo{
				MentionedJid: at,
				Participant:  &participant,
				StanzaId:     &stanzaId,
				QuotedMessage: &waproto.Message{
					Conversation: &conversation,
				},
			},
		},
	}
	//marshal
	pbData, err := proto.Marshal(w3)
	if err != nil {
		fmt.Println("发送群@", err.Error())
		return nil, err
	}
	// encrypt pb data
	c, err := m.axolotlManager.Encrypt(groupId.RawId(), pbData, true, u.RawId())
	if err != nil {
		return nil, err
	}
	builder := m.message.BuildMessage(groupId.GroupId(), veriFiledName, "text", c, cs, utils.CalcPHash(participants))
	m.SendBuilder(builder)
	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(groupId.GroupId(), content, "text")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendSnsText 发动态文本
func (m *MainNodeProcessor) SendSnsText(veriFiledName uint64, u, content string, participants []string) (iface.NodeBuilder, error) {
	jid := NewJid(u)
	//GetPreKeys
	if err := m.GetPreKeys(false, jid.Jid()); err != nil {
		fmt.Println("发动态文本GetPreKeys", err.Error())
		return nil, err
	}

	cs, err := m.encryptSenderKeyDistributionsBroadCast(jid, NewJid("status@broadcast"), participants)
	if err != nil {
		fmt.Println("发动态文本encryptSenderKeyDistributionsBroadCast", err.Error())
		return nil, err
	}
	var font = waproto.ExtendedTextMessage_SANS_SERIF
	var preview = waproto.ExtendedTextMessage_NONE
	w3 := &waproto.Message{
		ExtendedTextMessage: &waproto.ExtendedTextMessage{
			Text:           proto.String(content),
			TextArgb:       proto.Uint32(0xFFFFFFFF),
			BackgroundArgb: proto.Uint32(0xFFC1A03F),
			Font:           &font,
			PreviewType:    &preview,
		},
	}
	//marshal
	pbData, err := proto.Marshal(w3)
	if err != nil {
		fmt.Println("发动态文本marshal", err.Error())
		return nil, err
	}
	// encrypt
	ciphertextMessage, err := m.axolotlManager.Encrypt("status@broadcast", pbData, true, jid.RawId())
	if err != nil {
		fmt.Println("发动态文本encrypt", err.Error())
		return nil, err
	}
	snsIqNodeBuilder := m.iq.BuildIqCreateIqSnsText(veriFiledName, jid.Jid(), ciphertextMessage, cs, utils.CalcPHash(participants))
	m.SendBuilder(snsIqNodeBuilder)
	return snsIqNodeBuilder, nil
}

// SendNumberExistence 检测号码是否存在
func (m *MainNodeProcessor) SendNumberExistence(number []string) (*msg.MySendMsg, error) {
	err := m.GetPreKeysNumber(false, number...)
	if err != nil {
		fmt.Println("号码不存在!", err.Error())
		return nil, err
	}
	return nil, nil
}

// SendTextMessage 发送文本消息
func (m *MainNodeProcessor) SendTextMessage(u, content string, veriFiledName uint64, at []string, stanzaId string, participant string, conversation string) (*msg.MySendMsg, error) {
	// is JId
	jid := NewJid(u)
	//GetPreKeys
	if err := m.GetPreKeys(false, jid.Jid()); err != nil {
		return nil, err
	}
	//set message
	w2 := &waproto.WAMessage{
		CONVERSATION: proto.String(content),
	}
	/*var preview = waproto.ExtendedTextMessage_NONE
	w2 := &waproto.Message{
		ExtendedTextMessage: &waproto.ExtendedTextMessage{
			Text:        proto.String(content),
			PreviewType: &preview,
			ContextInfo: &waproto.ContextInfo{
				Participant: &participant,
				StanzaId:    &stanzaId,
				QuotedMessage: &waproto.Message{
					Conversation: &conversation,
				},
			},
		},
	}*/
	//marshal
	d, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt
	ciphertextMessage, err := m.axolotlManager.Encrypt(jid.Jid(), d, false)
	if err != nil {
		return nil, err
	}
	// sendTextMessage
	builder := m.message.BuildMessage(jid.Jid(), veriFiledName, "text", ciphertextMessage, nil)
	m.SendBuilder(builder)

	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(jid.Jid(), content, "text")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendImageMessage 发送图片消息
func (m *MainNodeProcessor) SendImageMessage(u, base64Data, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64, veriFiledName uint64) (*msg.MySendMsg, error) {
	// is JId
	jid := NewJid(u)
	//GetPreKeys
	if err := m.GetPreKeys(false, jid.Jid()); err != nil {
		return nil, err
	}
	fileByte, errs := base64.StdEncoding.DecodeString(base64Data)
	if errs != nil {
		return nil, errs
	}
	w2 := &waproto.Message{
		ImageMessage: &waproto.ImageMessage{
			Url:               &url,
			JpegThumbnail:     fileByte,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			MediaKey:          mediaKey,
			FileEncSha256:     fileEncSha256,
			FileSha256:        FileSha256,
			FileLength:        &FileLength,
			DirectPath:        &directPath,
			Mimetype:          proto.String("image/png"),
		},
	}
	//marshal
	d, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt
	ciphertextMessage, err := m.axolotlManager.Encrypt(jid.Jid(), d, false)
	if err != nil {
		return nil, err
	}
	// sendImageMessage
	builder := m.message.BuildImageMessage(veriFiledName, jid.Jid(), "media", ciphertextMessage, nil)
	m.SendBuilder(builder)

	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(jid.Jid(), "", "image")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendAudioMessage 发送语音
func (m *MainNodeProcessor) SendAudioMessage(u, base64Data, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64, veriFiledName uint64) (*msg.MySendMsg, error) {
	// is JId
	jid := NewJid(u)
	//GetPreKeys
	if err := m.GetPreKeys(false, jid.Jid()); err != nil {
		return nil, err
	}
	_, errs := base64.StdEncoding.DecodeString(base64Data)
	if errs != nil {
		return nil, errs
	}
	w2 := &waproto.Message{
		AudioMessage: &waproto.AudioMessage{
			Url:      &url,
			MediaKey: mediaKey,
			//Seconds: proto.Uint32(2),
			FileEncSha256:     fileEncSha256,
			FileSha256:        FileSha256,
			FileLength:        &FileLength,
			Ptt:               proto.Bool(true),
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			DirectPath:        &directPath,
			Mimetype:          proto.String("audio/ogg; codecs=opus"),
			ContextInfo:       &waproto.ContextInfo{},
		},
	}
	//marshal
	d, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt
	ciphertextMessage, err := m.axolotlManager.Encrypt(jid.Jid(), d, false)
	if err != nil {
		return nil, err
	}
	// sendImageMessage
	builder := m.message.BuildVideoMessage(veriFiledName, jid.Jid(), "media", ciphertextMessage, nil)
	m.SendBuilder(builder)

	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(jid.Jid(), "", "audio")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// 发送视频 SendVideoMessage
func (m *MainNodeProcessor) SendVideoMessage(u, base64Data, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64, veriFiledName uint64) (*msg.MySendMsg, error) {
	// is JId
	jid := NewJid(u)
	//GetPreKeys
	if err := m.GetPreKeys(false, jid.Jid()); err != nil {
		return nil, err
	}
	fileByte, errs := base64.StdEncoding.DecodeString(base64Data)
	if errs != nil {
		return nil, errs
	}
	w2 := &waproto.Message{
		VideoMessage: &waproto.VideoMessage{
			//Caption:       &msg.Caption,
			JpegThumbnail: fileByte,
			Url:           proto.String(url),
			//GifPlayback:   &msg.GifPlayback,
			MediaKey: mediaKey,
			//Seconds:       &msg.Length,
			FileEncSha256:     fileEncSha256,
			FileSha256:        FileSha256,
			FileLength:        proto.Uint64(FileLength),
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			DirectPath:        &directPath,
			Mimetype:          proto.String("video/mp4"),
			ContextInfo:       &waproto.ContextInfo{},
		},
	}
	//marshal
	d, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt
	ciphertextMessage, err := m.axolotlManager.Encrypt(jid.Jid(), d, false)
	if err != nil {
		return nil, err
	}
	// sendImageMessage
	builder := m.message.BuildAudioMessage(veriFiledName, jid.Jid(), "media", ciphertextMessage, nil)
	m.SendBuilder(builder)

	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(jid.Jid(), "", "audio")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendVcardMessage 发送名片消息
func (m *MainNodeProcessor) SendVcardMessage(u string, tel, vcardName string, veriFiledName uint64) (*msg.MySendMsg, error) {
	// is JId
	jid := NewJid(u)
	//GetPreKeys
	if err := m.GetPreKeys(false, jid.Jid()); err != nil {
		return nil, err
	}
	vcard := "BEGIN:VCARD\nVERSION:3.0\nN:;" + vcardName + ";;;\nFN:" + vcardName + "\nitem1.TEL:" + tel + "\nitem1.X-ABLabel:Mobile\nEND:VCARD"
	w2 := &waproto.Message{
		ContactMessage: &waproto.ContactMessage{
			DisplayName: proto.String(tel), //"+1 631-480-9861"
			Vcard:       proto.String(vcard),
		},
	}
	//marshal
	d, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt
	ciphertextMessage, err := m.axolotlManager.Encrypt(jid.Jid(), d, false)
	if err != nil {
		return nil, err
	}
	// sendImageMessage
	builder := m.message.BuildVcardMessage(veriFiledName, jid.Jid(), "media", ciphertextMessage, nil)
	m.SendBuilder(builder)

	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(jid.Jid(), "", "vcard")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendImageGroupMessage 发送群图片消息
func (m *MainNodeProcessor) SendImageGroupMessage(u, groupId JId, base64Data, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64, veriFiledName uint64) (*msg.MySendMsg, error) {
	var participants []string
	// get group numbers
	iqWg2Query := m.iq.BuildIqWg2Query(groupId.GroupId())
	m.SendBuilder(iqWg2Query)
	// wait result
	result, err := iqWg2Query.GetResult()
	if err != nil {
		return nil, err
	}
	// iqResult
	if iqResult, ok := result.(entity.IqResult); ok {
		groupInfo := iqResult.GetGroupInfo()
		for jid, participant := range groupInfo.Participants() {
			// if is supper admin
			if xtype, ok := participant["type"]; ok ||
				strings.Contains(xtype.String(), "superadmin") {
				if u.Jid() == jid {
					continue
				}
				//u = NewJid(jid)
			}
			participants = append(participants, jid)
		}
	}
	// encryptSenderKeyDistributions
	cs, err := m.encryptSenderKeyDistributions(u, groupId, participants)
	fileByte, errs := base64.StdEncoding.DecodeString(base64Data)
	if errs != nil {
		return nil, errs
	}
	w2 := &waproto.Message{
		ImageMessage: &waproto.ImageMessage{
			Url:               &url,
			JpegThumbnail:     fileByte,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			MediaKey:          mediaKey,
			FileEncSha256:     fileEncSha256,
			FileSha256:        FileSha256,
			FileLength:        &FileLength,
			DirectPath:        &directPath,
			Mimetype:          proto.String("image/png"),
		},
	}
	//marshal
	pbData, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt pb data
	c, err := m.axolotlManager.Encrypt(groupId.RawId(), pbData, true, u.RawId())
	if err != nil {
		return nil, err
	}
	builder := m.message.BuildImageMessage(veriFiledName, groupId.GroupId(), "media", c, cs, utils.CalcPHash(participants))
	m.SendBuilder(builder)
	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(groupId.GroupId(), "", "image")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendAudioGroupMessage
func (m *MainNodeProcessor) SendAudioGroupMessage(u, groupId JId, base64Data, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64, veriFiledName uint64) (*msg.MySendMsg, error) {
	var participants []string
	// get group numbers
	iqWg2Query := m.iq.BuildIqWg2Query(groupId.GroupId())
	m.SendBuilder(iqWg2Query)
	// wait result
	result, err := iqWg2Query.GetResult()
	if err != nil {
		return nil, err
	}
	// iqResult
	if iqResult, ok := result.(entity.IqResult); ok {
		groupInfo := iqResult.GetGroupInfo()
		for jid, participant := range groupInfo.Participants() {
			// if is supper admin
			if xtype, ok := participant["type"]; ok ||
				strings.Contains(xtype.String(), "superadmin") {
				if u.Jid() == jid {
					continue
				}
				//u = NewJid(jid)
			}
			participants = append(participants, jid)
		}
	}
	// encryptSenderKeyDistributions
	cs, err := m.encryptSenderKeyDistributions(u, groupId, participants)
	_, errs := base64.StdEncoding.DecodeString(base64Data)
	if errs != nil {
		return nil, errs
	}
	w2 := &waproto.Message{
		AudioMessage: &waproto.AudioMessage{
			Url:               &url,
			MediaKey:          mediaKey,
			FileEncSha256:     fileEncSha256,
			FileSha256:        FileSha256,
			FileLength:        &FileLength,
			Ptt:               proto.Bool(false),
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			DirectPath:        &directPath,
			Mimetype:          proto.String("audio/ogg; codecs=opus"),
		},
	}
	//marshal
	pbData, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt pb data
	c, err := m.axolotlManager.Encrypt(groupId.RawId(), pbData, true, u.RawId())
	if err != nil {
		return nil, err
	}
	builder := m.message.BuildAudioMessage(veriFiledName, groupId.GroupId(), "media", c, cs, utils.CalcPHash(participants))
	m.SendBuilder(builder)
	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(groupId.GroupId(), "", "audio")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendVideoGroupMessage 发送群视频消息
func (m *MainNodeProcessor) SendVideoGroupMessage(u, groupId JId, base64Data, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64, veriFiledName uint64) (*msg.MySendMsg, error) {
	var participants []string
	// get group numbers
	iqWg2Query := m.iq.BuildIqWg2Query(groupId.GroupId())
	m.SendBuilder(iqWg2Query)
	// wait result
	result, err := iqWg2Query.GetResult()
	if err != nil {
		return nil, err
	}
	// iqResult
	if iqResult, ok := result.(entity.IqResult); ok {
		groupInfo := iqResult.GetGroupInfo()
		for jid, participant := range groupInfo.Participants() {
			// if is supper admin
			if xtype, ok := participant["type"]; ok ||
				strings.Contains(xtype.String(), "superadmin") {
				if u.Jid() == jid {
					continue
				}
				//u = NewJid(jid)
			}
			participants = append(participants, jid)
		}
	}
	// encryptSenderKeyDistributions
	cs, err := m.encryptSenderKeyDistributions(u, groupId, participants)
	fileByte, errs := base64.StdEncoding.DecodeString(base64Data)
	if errs != nil {
		return nil, errs
	}
	w2 := &waproto.Message{
		VideoMessage: &waproto.VideoMessage{
			//Caption:       &msg.Caption,
			JpegThumbnail: fileByte,
			Url:           proto.String(url),
			//GifPlayback:   &msg.GifPlayback,
			MediaKey: mediaKey,
			//Seconds:       &msg.Length,
			FileEncSha256:     fileEncSha256,
			FileSha256:        FileSha256,
			FileLength:        proto.Uint64(FileLength),
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
			DirectPath:        &directPath,
			Mimetype:          proto.String("video/mp4"),
			ContextInfo:       &waproto.ContextInfo{},
		},
	}
	//marshal
	pbData, err := proto.Marshal(w2)
	if err != nil {
		return nil, err
	}
	// encrypt pb data
	c, err := m.axolotlManager.Encrypt(groupId.RawId(), pbData, true, u.RawId())
	if err != nil {
		return nil, err
	}
	builder := m.message.BuildVideoMessage(veriFiledName, groupId.GroupId(), "media", c, cs, utils.CalcPHash(participants))
	m.SendBuilder(builder)
	// save content id
	id := builder.GetMsgId()
	mySendMsg := msg.CreateMySendMsg(groupId.GroupId(), "", "video")
	err = m.msgManager.AddMySendMsg(id, mySendMsg)
	return mySendMsg, err
}

// SendSyncContacts 同步联系人
func (m *MainNodeProcessor) SendSyncContacts(contacts []string) iface.NodeBuilder {
	buildIqUSyncContact := m.iq.BuildIqUSyncContact(contacts)
	m.SendBuilder(buildIqUSyncContact)
	return buildIqUSyncContact
}

// SendSyncContactsAdd 扫码后-会调用 1
func (m *MainNodeProcessor) SendSyncContactsAdd(contacts []string) iface.NodeBuilder {
	buildIqUSyncContact := m.iq.BuildIqUSyncContactAdd(contacts)
	m.SendBuilder(buildIqUSyncContact)
	return buildIqUSyncContact
}

// SendSyncContactsInteractive 扫码后-会调用 2
func (m *MainNodeProcessor) SendSyncContactsInteractive(contacts []string) iface.NodeBuilder {
	buildIqUSyncContact := m.iq.BuildIqUSyncContactInteractive(contacts)
	m.SendBuilder(buildIqUSyncContact)
	return buildIqUSyncContact
}

// SyncAddOneContacts
func (m *MainNodeProcessor) SyncAddOneContacts(contacts []string) iface.NodeBuilder {
	buildIqUSyncContact := m.iq.BuildIqUSyncSyncAddOneContacts(contacts)
	m.SendBuilder(buildIqUSyncContact)
	return buildIqUSyncContact
}

// SyncAddScanContacts --扫号用
func (m *MainNodeProcessor) SyncAddScanContacts(contacts []string) iface.NodeBuilder {
	buildIqUSyncContact := m.iq.BuildIqUSyncSyncAddScanContacts(contacts)
	m.SendBuilder(buildIqUSyncContact)
	return buildIqUSyncContact
}

// SendSetEncryptKeys
func (m *MainNodeProcessor) SendSetEncryptKeys() error {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("SendSetEncryptKeys err: %v\n", r)
		}
	}()
	preKeysCount := m.axolotlManager.GetUnSentPreKeysCount(0)
	if preKeysCount <= 0 {
		m.axolotlManager.GeneratingPreKeys()
	}
	// get pre keys
	preKeys, err := m.axolotlManager.LoadUnSendPreKey()
	if err != nil {
		return err
	}
	// get skey
	signedPreKey := m.axolotlManager.SignedPreKeyStore.LoadSignedPreKey(0)
	// get  Identity Key
	identityKeyPair := m.axolotlManager.IdentityStore.GetIdentityKeyPair()
	// get reg id
	registrationId := m.axolotlManager.IdentityStore.GetLocalRegistrationId()
	encryptKeys := m.iq.BuildIqIqSetEncryptKeys(preKeys, signedPreKey, *identityKeyPair.PublicKey(), registrationId)
	// send
	m.SendBuilder(encryptKeys)
	// wait resp
	result, err := encryptKeys.GetResult()
	if err != nil {
		return err
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		fmt.Println("成功----->", result)
		//m.SendSetEncryptKeys()
		return nil
	}
	ids := make([]int, 0)
	for _, key := range preKeys {
		ids = append(ids, int(key.ID().Value))
	}
	return m.axolotlManager.UpdatePreKeysSent(ids)
}

// SendChatState 发送聊天状态
func (m *MainNodeProcessor) SendChatState(u string, isGroup, paused bool) {
	var to string
	// create jid
	jid := NewJid(u)
	// if is group chat
	if isGroup {
		to = jid.GroupId()
	} else {
		to = jid.Jid()
	}
	// create chat state
	var state *ChatStateNode
	// paused
	if paused {
		state = createChatStatePaused(to)
	} else {
		state = createChatStateComposing(to)
	}
	// send
	m.SendBuilder(state)
}

// SendProfilePicture 设置头像
func (m *MainNodeProcessor) SendProfilePicture(u string, picture []byte) iface.NodeBuilder {
	jid := NewJid(u)
	profilePicture := m.iq.BuildIqSetProfilePicture(picture, jid.Jid())
	m.SendBuilder(profilePicture)
	return profilePicture
}

// SendGetQr 获取个人二维码
func (m *MainNodeProcessor) SendGetQr(u string) iface.NodeBuilder {
	qr := m.iq.BuildIqGetQr()
	m.SendBuilder(qr)
	return qr
}

// SendNickName 设置名称
func (m *MainNodeProcessor) SendNickName(name string) iface.NodeBuilder {
	qr := m.iq.BuildIqNickName(name)
	m.SendBuilder(qr)
	return qr
}

// SetQrRevoke
func (m *MainNodeProcessor) SetQrRevoke(u string) iface.NodeBuilder {
	qr := m.iq.BuildSetQrRevoke()
	m.SendBuilder(qr)
	return qr
}

// ScanCode 扫描二维码
func (m *MainNodeProcessor) ScanCode(code string, opCode int32) iface.NodeBuilder {
	qr := m.iq.BuildScanCode(code, opCode)
	m.SendBuilder(qr)
	return qr
}

// InviteCode 邀请code
func (m *MainNodeProcessor) InviteCode(code, toWid string) iface.NodeBuilder {
	qr := m.iq.BuildInviteCode(code, toWid)
	m.SendBuilder(qr)
	return qr
}

// SendMediaConIq 获取CDN
func (m *MainNodeProcessor) SendMediaConIq() iface.NodeBuilder {
	mediaRsp := m.iq.BuildIqMediaConIq()
	m.SendBuilder(mediaRsp)
	return mediaRsp
}

// SendGetProfilePicture 获取头像
func (m *MainNodeProcessor) SendGetProfilePicture(u string) iface.NodeBuilder {
	jid := NewJid(u)
	profilePicture := m.iq.BuildIqGetProfilePicture(jid.Jid())
	m.SendBuilder(profilePicture)
	return profilePicture
}

// SendGetProfilePreview 获取头像
func (m *MainNodeProcessor) SendGetProfilePreview(u string) iface.NodeBuilder {
	jid := NewJid(u)
	profilePicture := m.iq.BuildIqGetProfilePreview(jid.Jid())
	m.SendBuilder(profilePicture)
	return profilePicture
}

// createIqState 上传个人签名
func (m *MainNodeProcessor) SendSetState(content string) iface.NodeBuilder {
	profilePicture := m.iq.BuildIqCreateIqState(content)
	m.SendBuilder(profilePicture)
	return profilePicture
}

// 获取状态
func (m *MainNodeProcessor) SendGetState(wid, toWid string) iface.NodeBuilder {
	jid := NewJid(wid)
	if toWid != "" {
		jid = NewJid(toWid)
	}
	profilePicture := m.iq.BuildIqCreateIqGetState(jid.Jid())
	m.SendBuilder(profilePicture)
	return profilePicture
}

// Send2Fa 两步验证
func (m *MainNodeProcessor) Send2Fa(code, email string) iface.NodeBuilder {
	fa2 := m.iq.BuildIq2Fa(code, email)
	m.SendBuilder(fa2)
	return fa2
}
