package app

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gogf/gf/util/gconv"

	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"ws-go/noise"
	"ws-go/protocol/axolotl"
	"ws-go/protocol/db"
	"ws-go/protocol/define"
	"ws-go/protocol/entity"
	"ws-go/protocol/handlers"
	_interface "ws-go/protocol/iface"
	"ws-go/protocol/impl"
	"ws-go/protocol/msg"
	"ws-go/protocol/network"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/node"
	"ws-go/protocol/utils/promise"
	"ws-go/waver"
	"ws-go/wslog"
)

// WSAppEvent
type WSAppEvent struct {
	NewChatMessageNotify func(message *entity.ChatMessage)
}

// SetNewMessageNotify 设置消息通知事件
func (w *WSAppEvent) SetNewMessageNotify(n func(message *entity.ChatMessage)) {
	if n != nil {
		w.NewChatMessageNotify = n
	}
}

type LoginStatus int32

func (l LoginStatus) String() string {
	switch l {
	case Online:
		return "Online"
	case Drops:
		return "Drops"
	case Disconnect:
		return "Disconnect"
	case Connect:
		return "Connect"
	case AuthFailed:
		return "AuthFailed"
	case HandshakeFailed:
		return "HandshakeFailed"
	case Banned:
		return "Banned"
	default:
		return ""
	}
}

const (
	_ LoginStatus = iota

	Online // 上线 认证成功后上线(连接成功的)  1
	Drops  // 掉线 认证成功后掉线 （连接断开）  2

	AuthFailed      // 认证失败 (连接成功后认证失败)  3
	Banned          // 被禁止使用  4
	HandshakeFailed // 握手失败   5
	Connect         // 连接成功   6
	Disconnect      // 断开连接   7
	NETNOT          // 网络异常   8
)

// NewWaAppCli
func NewWaAppCli(info *AccountInfo) *WaApp {
	// TODO 应该修改成针对不同的用户不同的版本
	platform := info.clientPayload.UserAgent.GetPlatform().String()
	//如果为安卓普通版
	if platform == "ANDROID" {
		newxxmp.SetWAXXMPVersion(waver.NewWA42())
	} else if platform == "PLATFORM_10" {
		//安卓企业版
		newxxmp.SetWAXXMPVersion(waver.NewBusinessWA42())
	} else {
		return nil
	}
	// set log context
	info.SetLogCtx(define.LOGKEYSUSERNAME, info.GetUserName())
	// create whatsapp client
	w := &WaApp{loginPromise: impl.NewResultPromise(), AccountInfo: info, WSAppEvent: &WSAppEvent{}}
	// msg manager
	msgManager := msg.NewManager()
	w.msgManager = msgManager
	// set axolotl manager
	axolotlManager, err := axolotl.NewAxolotlManager(info.GetUserName(), info.staticPubKey, info.staticPriKey)
	if err != nil {
		return nil
	}
	w.axolotlManager = axolotlManager
	// set network
	//payLoad, _ := proto.Marshal(info.clientPayload)
	w.netWork = network.NewNoiseClient(info.routingInfo, nil, noise.DHKey{}, w)
	segmentProcessor := w.netWork.GetSegment()
	// set node processor
	nodeProcessor := node.NewMainNodeProcessor()
	w.node = nodeProcessor
	w.node.SetAxolotlManager(w.axolotlManager)
	w.node.SetMsgManager(w.msgManager)
	w.node.SetSegmentOutputProcessor(segmentProcessor)
	// handles
	handles := handlers.NewHandles()
	// chat message handler
	chatMessageHandler := handlers.NewChatMessageHandler(axolotlManager, nodeProcessor)
	chatMessageHandler.SetNotifyEvent(w)
	handles.AddHandler(chatMessageHandler)
	// notification handler
	//handles.AddHandler(handlers.NewNotificationHandler())
	// set handlers
	w.node.SetHandles(handles)
	w.mutex = sync.Mutex{}
	w.autoLogin = false
	return w
}

// WaApp
type WaApp struct {
	*AccountInfo
	*WSAppEvent
	autoLogin   bool
	loginStatus LoginStatus
	// module
	loginPromise   *impl.ResultPromise
	netWork        *network.NoiseNetWork
	axolotlManager *axolotl.Manager
	msgManager     *msg.Manager
	node           *node.MainNodeProcessor
	// 重新登录等待 防止在没有登录完成时重复登录
	retryLoginWait sync.WaitGroup
	// Mutex protects against data race conditions.
	mutex sync.Mutex
}

func (w *WaApp) GetLoginStatus() LoginStatus {
	if w != nil {
		return w.loginStatus
	}
	return Disconnect
}

func (w *WaApp) SetLoginStatusOne(loginStatus LoginStatus) {
	w.loginStatus = loginStatus
}

// SetLoginStatus 设置登录状态
func (w *WaApp) SetLoginStatus(loginStatus LoginStatus) {
	w.loginStatus = loginStatus
	fmt.Println("账号[", w.GetUserName(), "]推送->loginStatus -> ", w.loginStatus)
	db.PushQueue(
		db.PushMsg{
			UserName: w.clientPayload.GetUsername(),
			Time:     time.Now().Unix(),
			Type:     db.Status.Number(),
			Data:     w.loginStatus,
		},
	)
}

func (w *WaApp) NewtWorkClose() {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("打印错误堆栈信息-NewtWorkClose panic: %v\n", r)
		}
	}()
	w.netWork.Close()
	w.node.Close()
}

func (w *WaApp) SetLoginStatusText(loginStatus LoginStatus, text string) {
	w.loginStatus = loginStatus
	log.Println("loginStatus -> ", w.loginStatus)
	db.PushQueue(
		db.PushMsg{
			UserName: w.clientPayload.GetUsername(),
			Time:     time.Now().Unix(),
			Type:     db.Status.Number(),
			Data:     w.loginStatus,
			Text:     text,
		},
	)
}

// WALogin
func (w *WaApp) WALogin() _interface.IPromise {
	w.mutex.Lock()
	// 当socket 没有连接或登录状态不为在线时可执行登录
	if !w.netWork.Connected() || w.loginStatus != Online {
		// 使用异步登录
		executor := func(resolve func(promise.Any), reject func(error)) {
			// time out
			/*go func() {
				<-time.After(time.Second * 30)
				reject(errors.New("login time out"))
			}()*/
			// update handshake settings
			settings := w.GetLoginSettings()
			// start
			err := w.netWork.Connect(settings)
			if err != nil {
				w.loginStatus = Disconnect
				w.NewtWorkClose()
				reject(err)
				return
			}
		}
		//初始化登录承诺超时30秒
		w.loginPromise.SetPromise(promise.New(executor))
	} else if w.netWork.Connected() && w.loginStatus == Online {
		// TODO 已经在线状态
		if w.loginPromise == nil {
			w.loginPromise.SuccessResolve("success")
		}
	}
	w.mutex.Unlock()
	return w.loginPromise
}

func (w *WaApp) ResetNetWork() _interface.IPromise {
	// 不是掉线的不继续重新登录
	if w.loginStatus == Drops {
		// 重置网络
		w.netWork.Reset()
		// 重置 processor
		w.node.Reset()
		return w.WALogin()
	}
	return nil
}

func (w *WaApp) SetNetWorkProxyLogin() {
	// 防止重复重新登录
	w.retryLoginWait.Add(1)
	defer w.retryLoginWait.Done()
	w.RetryLogin()
}

func (w *WaApp) HasUnsentPreKeys() string {
	if w.axolotlManager.HasUnsentPreKeys() {
		w.node.SendActive()
		// upload pre keys
		err := w.node.SendSetEncryptKeys()
		if err != nil {
			log.Println("update pre keys err", err)
			return "update pre keys err"
		}
	} else {
		return "已上传key"
	}
	return "未知错误"
}

// RetryLogin 重新登录
func (w *WaApp) RetryLogin() _interface.IPromise {
	// 不是掉线的不继续重新登录
	if w.loginStatus == Drops {
		w.loginPromise.Reject(errors.New("RetryLogin"))
		// 重置网络
		w.netWork.Reset()
		// 重置 processor
		w.node.Reset()
		// 等待十秒
		time.Sleep(time.Second * 10)
		return w.WALogin()
	}
	return nil
}

/*// timingSendIqPing 定时发送 iq ping
func (w *WaApp) timingSendIqPing() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("run web error:timingSendIqPing")
			}
		}()

		if w == nil && w.netWork == nil {
			return
		}
		if w.loginStatus == Disconnect {
			fmt.Println("賬號退出---")
			return
		}
		// 检查网络连接情况
		if !w.netWork.Connected() {
			return
		}
		// 四分钟发送一次
		<-time.After(time.Minute * 4)
		// success
		successFunc := func(any promise.Any) {
			//log.Println("send iq ping resp")
			wslog.GetLogger().Ctx(w.ctx).Info("send iq ping resp ok")
			db.PushQueue(
				db.PushMsg{
					Time:     time.Now().Unix(),
					UserName: w.clientPayload.GetUsername(),
					Type:     db.System.Number(),
					Data:     "send iq ping resp succeed",
				},
			)
			w.timingSendIqPing()

			wslog.GetLogger().Ctx(w.ctx).Info("继续发送 ping")

		}
		// failure
		failureFunc := func(err error) {
			wslog.GetLogger().Ctx(w.ctx).Error("send ping failure ")
			db.PushQueue(
				db.PushMsg{
					Time:     time.Now().Unix(),
					UserName: w.clientPayload.GetUsername(),
					Type:     db.System.Number(),
					Data:     "send iq ping failure",
				},
			)
			//return
		}
		w.node.SendIqPing().
			SetListenHandler(successFunc, failureFunc)
	}()
}*/

// ===================== API =================================
// AddGroupMember 添加群成员
func (w *WaApp) AddGroupMember(groupId string, members ...string) _interface.IPromise {
	return w.node.SendAddGroup(groupId, members...)
}

// CreateGroup 创建群聊
func (w *WaApp) CreateGroup(subject string, participants []string) _interface.IPromise {
	return w.node.SendCreateGroup(w.GetUserName(), subject, participants)
}

// GetGroupMember 获取群成员
func (w *WaApp) GetGroupMember(groupId string) _interface.IPromise {
	grid := node.NewJid(groupId)
	return w.node.SendGetGroupMember(grid.GroupId())
}

// GetGroupCode 获取群二维码
func (w *WaApp) GetGroupCode(groupId string) _interface.IPromise {
	return w.node.SendGetGroupCode(node.NewJid(w.GetUserName()), node.NewJid(groupId))
}

// CreateGroupAdmin  设置群管理&取消群管理
func (w *WaApp) CreateGroupAdmin(groupId string, opcode int32, toWid string) _interface.IPromise {
	if opcode == 0 {
		return w.node.CreateDemoteGroupAdmin(node.NewJid(w.GetUserName()), node.NewJid(groupId), node.NewJid(toWid))
	}
	return w.node.CreateGroupAdmin(node.NewJid(w.GetUserName()), node.NewJid(groupId), node.NewJid(toWid))
}

// SetGroupDesc  设置群描述
func (w *WaApp) SetGroupDesc(groupId string, desc string) _interface.IPromise {
	return w.node.SetGroupDesc(node.NewJid(w.GetUserName()), node.NewJid(groupId), desc)
}

// SendLogOutGroup 退出群组
func (w *WaApp) SendLogOutGroup(groupId string) _interface.IPromise {
	return w.node.CreateLogOutGroup(node.NewJid(w.GetUserName()), node.NewJid(groupId))
}

func (w *WaApp) SendPresencesSubscribeNew(u string) _interface.IPromise {
	// 发送订阅
	return w.node.SendPresencesSubscribeNew(u)
}

// SendPresencesSubscribe 发送订阅（发送消息前需要提前订阅指定用户）
func (w *WaApp) SendPresencesSubscribe(u string) _interface.IPromise {
	// 发送订阅
	/*presencesSubscribe := w.node.SendPresencesSubscribe(u)
	result, err := presencesSubscribe.GetResult()
	log.Println("presencesSubscribe ", result, err)*/
	return w.node.SendPresencesSubscribe(u)
}

// SendEncrypt 发送消息前会用到
func (w *WaApp) SendEncrypt(u string) _interface.IPromise {
	// 发送订阅
	node := w.node.SendEncrypt(node.NewJid(u).Jid())
	return node
}

// SendGroupTextMessage 发送群聊消息
func (w *WaApp) SendGroupTextMessage(g string, msg string, at []string, stanzaId string, participant string, conversation string) (*msg.MySendMsg, error) {
	return w.node.SendTextGroupMessage(w.GetVeriFiledName(), node.NewJid(w.GetUserName()), node.NewJid(g), msg, at, stanzaId, participant, conversation)
}

// SendTextMessage 发送文本消息
func (w *WaApp) SendTextMessage(u, msg string, at []string, stanzaId string, participant string, conversation string) (*msg.MySendMsg, error) {
	// send text
	return w.node.SendTextMessage(u, msg, w.GetVeriFiledName(), at, stanzaId, participant, conversation)
}

// SendNumberExistence 查询用户是否存在
func (w *WaApp) SendNumberExistence(number []string) (*msg.MySendMsg, error) {
	return w.node.SendNumberExistence(number)
}

// 发送图片消息
func (w *WaApp) SendImageMessage(u, base64, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64) (*msg.MySendMsg, error) {
	return w.node.SendImageMessage(u, base64, url, directPath, mediaKey, fileEncSha256, FileSha256, FileLength, w.GetVeriFiledName())
}

// 发送语音消息
func (w *WaApp) SendAudioMessage(u, base64, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64) (*msg.MySendMsg, error) {
	return w.node.SendAudioMessage(u, base64, url, directPath, mediaKey, fileEncSha256, FileSha256, FileLength, w.GetVeriFiledName())
}

// 发送视频消息
func (w *WaApp) SendVideoMessage(u, base64, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64) (*msg.MySendMsg, error) {
	return w.node.SendVideoMessage(u, base64, url, directPath, mediaKey, fileEncSha256, FileSha256, FileLength, w.GetVeriFiledName())
}

// 发送名片消息
func (w *WaApp) SendVcardMessage(u string, tel, vcardName string) (*msg.MySendMsg, error) {
	return w.node.SendVcardMessage(u, tel, vcardName, w.GetVeriFiledName())
}

// SendGroupImageMessage 发送群聊图片消息
func (w *WaApp) SendGroupImageMessage(g string, base64, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64) (*msg.MySendMsg, error) {
	return w.node.SendImageGroupMessage(node.NewJid(w.GetUserName()), node.NewJid(g), base64, url, directPath, mediaKey, fileEncSha256, FileSha256, FileLength, w.GetVeriFiledName())
}
func (w *WaApp) SendGroupAudioMessage(g string, base64, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64) (*msg.MySendMsg, error) {
	return w.node.SendAudioGroupMessage(node.NewJid(w.GetUserName()), node.NewJid(g), base64, url, directPath, mediaKey, fileEncSha256, FileSha256, FileLength, w.GetVeriFiledName())
}

// SendGroupVideoMessage 发送群聊视频消息
func (w *WaApp) SendGroupVideoMessage(g string, base64, url, directPath string, mediaKey, fileEncSha256, FileSha256 []byte, FileLength uint64) (*msg.MySendMsg, error) {
	return w.node.SendVideoGroupMessage(node.NewJid(w.GetUserName()), node.NewJid(g), base64, url, directPath, mediaKey, fileEncSha256, FileSha256, FileLength, w.GetVeriFiledName())
}

// SendSyncContacts 同步联系人
func (w *WaApp) SendSyncContacts(u []string) _interface.IPromise {
	return w.node.SendSyncContacts(u)
}

// SendSyncContactsAdd 同步联系人->扫对方二维码后发送 1
func (w *WaApp) SendSyncContactsAdd(u []string) _interface.IPromise {
	return w.node.SendSyncContactsAdd(u)
}

// SendSyncContactsInteractive 同步联系人-> 扫对方二维码后发送 2
func (w *WaApp) SendSyncContactsInteractive(u []string) _interface.IPromise {
	return w.node.SendSyncContactsInteractive(u)
}
func (w *WaApp) SyncAddOneContacts(u []string) _interface.IPromise {
	return w.node.SyncAddOneContacts(u)
}

// SyncAddScanContacts 扫号用
func (w *WaApp) SyncAddScanContacts(u []string) _interface.IPromise {
	return w.node.SyncAddScanContacts(u)
}

// SendChatState 发送聊天状态 paused -> Composing
func (w *WaApp) SendChatState(u string, isGroup, paused bool) {
	w.node.SendChatState(u, isGroup, paused)
}

// SendProfilePicture 设置头像
func (w *WaApp) SendProfilePicture(picture []byte) _interface.IPromise {
	return w.node.SendProfilePicture(w.GetUserName(), picture)
}

// SendGetQ 获取二维码
func (w *WaApp) SendGetQr() _interface.IPromise {
	return w.node.SendGetQr(w.GetUserName())
}

// SendNickName 设置名称
func (w *WaApp) SendNickName(name string) _interface.IPromise {
	return w.node.SendNickName(name)
}

// SetQrRevoke 重置二维码
func (w *WaApp) SetQrRevoke() _interface.IPromise {
	return w.node.SetQrRevoke(w.GetUserName())
}

// ScanCodeService 扫描二维码
func (w *WaApp) ScanCode(code string, opCode int32) _interface.IPromise {
	return w.node.ScanCode(code, opCode)
}

// InviteCode 邀请code
func (w *WaApp) InviteCode(code, toWid string) _interface.IPromise {
	return w.node.InviteCode(code, toWid)
}

// SendMediaConIq 获取cdn
func (w *WaApp) SendMediaConIq() _interface.IPromise {
	return w.node.SendMediaConIq()
}

// GetProfilePicture 获取用户头像
func (w *WaApp) GetProfilePicture(u string) _interface.IPromise {
	return w.node.SendGetProfilePicture(u)
}

// GetPreviewPicture 获取头像
func (w *WaApp) GetPreviewPicture(u string) _interface.IPromise {
	return w.node.SendGetProfilePreview(u)
}

// GetProfilePicture 设置个性签名
func (w *WaApp) SendSetState(content string) _interface.IPromise {
	return w.node.SendSetState(content)
}

// SendGetState 获取状态
func (w *WaApp) SendGetState(toWid string) _interface.IPromise {
	return w.node.SendGetState(w.GetUserName(), toWid)
}

// SendSnsText 发送文字动态
func (w *WaApp) SendSnsText(text string, participants []string) (_interface.IPromise, error) {
	return w.node.SendSnsText(w.GetVeriFiledName(), w.GetUserName(), text, participants)
}

// TwoVerify 二步安全验证
func (w *WaApp) TwoVerify(code, email string) _interface.IPromise {
	return w.node.Send2Fa(code, email)
}

// SendCategories 获取商业版类型列表
func (w *WaApp) SendCategories() _interface.IPromise {
	return w.node.SendCategories()
}

// SendBusinessProfile 设置商业版类型
func (w *WaApp) SendBusinessProfile(categoryId string) _interface.IPromise {
	return w.node.SendBusinessProfile(categoryId)
}

// SendBusinessPresenceAvailable 设置商业名字
func (w *WaApp) SendBusinessPresenceAvailable(name string) _interface.IPromise {
	return w.node.SendBusinessPresenceAvailable(name)
}

// GetQueryMyMsg
func (w *WaApp) GetQueryMyMsg() ([]msg.MySendMsg, error) {
	return nil, nil
}

// ===================== System settings =======================
// SetNetWorkProxy 设置代理
func (w *WaApp) SetNetWorkProxy(p string) {
	w.netWork.SetNetWorkProxy(p)
}
func (w *WaApp) GetNetWorkProxy() string {
	return w.netWork.GetNetWorkProxyStr()
}

// loginResultNotify 登录结果通知
func (w *WaApp) loginResultNotify(loginSuccess bool) {
	// TODO 通知登录结果
	if loginSuccess {
		// 服务器返回 success 表示登录成功
		//log.Println("上线成功")
		wslog.GetLogger().Ctx(w.ctx).Info("上线成功")
		w.SetLoginStatus(Online)

	} else {
		// 服务器返回 failure 表示认证失败
		//w.SetLoginStatus(AuthFailed)
		//log.Println("上线失败")
		wslog.GetLogger().Ctx(w.ctx).Info("上线失败")
		// 认证失败 不是返回的错误。握手失败或者socket断开才返回error
		w.loginPromise.SuccessResolve(w.loginStatus)
	}
}

// handlerNodeTree
func (w *WaApp) handleNodeTree(node *newxxmp.Node) {
	if node == nil {
		return
	}
	//fmt.Println("消息状态：--------------------：" + node.GetString())
	switch node.GetTag() {
	case "success":
		w.handleTagSuccess(node)
		break
	case "failure":
		w.handleTagFailure(*node)
		break
	case "stream:error":
		w.handleStreamError(*node)
		break
	default:
		//判断是否带有biz_status 商业版本初始化数据
		//bizNode:=node.GetChildrenByTag("biz_status")
		w.node.ProcessNode(node)
		/*if bizNode!=nil{
			fmt.Println("biz_status")
			w.node.SendCategories()
			w.node.SendPresenceAvailable("汽修学校教育001")
			w.node.SendBusinessProfile("1223524174334504")
			w.node.SendBusinessProfileTow(w.GetUserName())
		}*/
		if node.GetTag() == "receipt" {
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
			//fmt.Println("======================================:"+receiptType);
			if receiptType == "read" {
				pushData := entity.RespMessage{
					Participant: participantAttr,
					From:        from,
					ContextType: receiptType,
					Id:          id,
				}
				db.PushQueue(
					db.PushMsg{
						Time:     time.Now().Unix(),
						UserName: w.clientPayload.GetUsername(),
						Type:     db.Reads.Number(),
						Data:     pushData,
					},
				)
			}
		}
	}
}

func (w *WaApp) handleStreamError(node newxxmp.Node) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:handleStreamError", err)
		}
	}()
	log.Println("handleStreamError builder node ", node.GetString())
	//两个地方抢线
	wslog.GetLogger().Ctx(w.ctx).Info("handleStreamError", w.clientPayload.GetUsername())
	if node.GetChildrenByTag("conflict") != nil && node.GetChildrenByTag("conflict").GetAttributeByValue("type") != "" {
		value := node.GetChildrenByTag("conflict").GetAttributeByValue("type")
		if value == "replaced" {
			wslog.GetLogger().Ctx(w.ctx).Info("账号被抢登录", w.clientPayload.GetUsername())
			w.loginStatus = Drops
			db.PushQueue(
				db.PushMsg{
					UserName: w.clientPayload.GetUsername(),
					Time:     time.Now().Unix(),
					Type:     db.System.Number(),
					Data:     -1,
					Text:     "handleStreamError stream:error:账号抢线,已自动下线",
				},
			)
		}
	}
}

// handleTagFailure
func (w *WaApp) handleTagFailure(node newxxmp.Node) {
	//log.Println("WhatsApp login:", node.GetTag())
	wslog.GetLogger().Ctx(w.ctx).Info("WhatsApp login:", node.GetTag())
	// 通知认证失败
	if w.loginPromise != nil {
		reason := node.GetAttributeByValue("reason")
		switch reason {
		case "503": //TODO 登录频繁了？
		case "401":
			w.SetLoginStatus(AuthFailed)
		case "403":
			// 账号被封
			w.SetLoginStatus(Banned)
		default:
			w.loginPromise.SuccessResolve(AuthFailed)
		}

	}

	// notify login failure
	w.loginResultNotify(false)
}

func (w *WaApp) HandleCall(n *newxxmp.Node) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:handleTagSuccess", err)
		}
	}()
}

// handleTagSuccess
func (w *WaApp) handleTagSuccess(n *newxxmp.Node) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:handleTagSuccess", err)
		}
	}()
	//wslog.GetLogger().Ctx(w.ctx).Info("go WhatsApp login:", n.Tag, "create:", n.GetAttribute("creation").Value())
	// notify login success
	w.loginResultNotify(true)
	// 通知登录成功
	if w.loginPromise != nil {
		w.loginPromise.SuccessResolve(Online)
	}
	// send urn:xmpp:whatsapp:push
	w.node.SendIqConfig()
	w.node.SendIqConfigOne()
	w.node.SendIqConfigTwo()
	//<iq to='s.whatsapp.net' xmlns='urn:xmpp:whatsapp:account' type='get' id='4'><crypto action='create'><google>aISZpqfM2+JRNfND2XzsiNNZnUDewdG4fuIC9gM1jD4=</google></crypto></iq>
	//
	// send ping
	w.node.SendIqPing()
	//登录成功开启
	connectMgr := WXServer.GetWXConnectMgr()
	//查询该链接是否存在
	iwxConnect := connectMgr.GetWXConnectByWXID(w.GetUserName())
	wslog.GetLogger().Ctx(w.ctx).Info("账号[", w.GetUserName(), "]查询链接信息开始")
	if iwxConnect == nil {
		//创建一个新的心跳管理器
		iwxConnect = NewWXConnect(w, connectMgr)
		err := iwxConnect.Start()
		if err != nil {
			fmt.Println("开启心跳服务失败", err.Error())
		}
	} else {
		wslog.GetLogger().Ctx(w.ctx).Info("账号[", w.GetUserName(), "]链接存在=", iwxConnect.GetWxConnID())
	}
	//end
	///w.timingSendIqPing()
	// send available
	w.node.SendPresenceAvailable()

	//检查key
	/*key := fmt.Sprintf("sentPreKeys:%s", w.GetUserName())
	//檢查key
	v, _ := db.Exists(key)
	if v {
		fmt.Println(w.GetUserName() + "----檢查Key走緩存")
		return
	}
	preKeys := w.axolotlManager.HasUnsentPreKeys()
	if preKeys {
		fmt.Println(w.GetUserName() + "----檢查Key發送")
		_ = db.SETExpirationObj(key, preKeys, 3600)
		w.node.SendActive()
		// upload pre keys
		err := w.node.SendSetEncryptKeys()
		if err != nil {
			log.Println("update pre keys err", err)
			return
		}
	}*/
}

// 消息回調通知redis
func (w *WaApp) NotifyHandleResult(any ...interface{}) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:NotifyHandleResult", err)
		}
	}()
	if len(any) <= 0 || any[0] == nil {
		return
	}
	// handler result
	if handle, ok := any[0].(*handlers.HandleResult); ok {
		result := handle.GetResult()
		switch result.(type) {
		case error:
		case *entity.ChatMessage:
			if w.NewChatMessageNotify != nil {
				message := result.(*entity.ChatMessage)
				//如果是消息媒体类型
				pushData := entity.RespMessage{
					T:           message.T(),
					Participant: message.Participant(),
					From:        message.From(),
					ContextType: message.ContextType(),
					Id:          message.Id(),
				}

				if message.ContextType() == "media" {
					if message.GetMessage() != nil {
						pushData.Message = entity.ParseProtoMessage(message.GetMessage())
					}
				} else {
					pushData.Message = entity.ParseProtoMessage(message.GetMessage())
				}
				MsgPost(gconv.String(db.PushMsg{Time: time.Now().Unix(), UserName: w.clientPayload.GetUsername(), Type: db.Msg.Number(), Data: pushData}))
				db.PushQueue(
					db.PushMsg{
						Time:     time.Now().Unix(),
						UserName: w.clientPayload.GetUsername(),
						Type:     db.Msg.Number(),
						Data:     pushData,
					},
				)
				w.NewChatMessageNotify(result.(*entity.ChatMessage))
			}
		case *entity.Receipt:
			fmt.Println("------------------------Receipt11369855", result.(*entity.Receipt))
		case *entity.Notification:
			fmt.Println("-------------notification", result.(*entity.Notification))
		}
	}
}
func MsgPost(data string) {
	payload := strings.NewReader(data)
	client := &http.Client{}
	req, err := http.NewRequest("POST", "URL", payload)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	return
}

// ======================= network ============================
func (w *WaApp) OnConnect() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:OnConnect", err)
		}
	}()
	// 设置登录状态为连接
	w.SetLoginStatus(Connect)
	//log.Println("连接成功")
	wslog.GetLogger().Ctx(w.ctx).Println("OnConnect success!")

	// 重置 segment processor
	segmentProcessor := w.netWork.GetSegment()
	w.node.SetSegmentOutputProcessor(segmentProcessor)
}
func (w *WaApp) OnRecvData(d []byte) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:OnRecvData", err)
		}
	}()
	decodeNode, err := newxxmp.XMMPDecodeNode(d)
	if err != nil {
		//log.Println("WaApp OnRecvData DecodeNode error:", err)
		wslog.GetLogger().Ctx(w.ctx).Error("WaApp OnRecvData DecodeNode error:", err)
		return
	}
	if decodeNode == nil {
		wslog.GetLogger().Ctx(w.ctx).Error("WaApp decodeNode decodeNode error:", decodeNode, hex.EncodeToString(d))
		return
	}
	//log.Println("DecodeNode:", decodeNode.GetString())
	//wslog.GetLogger().Ctx(w.ctx).Info("onRecvData", "decode node", decodeNode.GetString())
	go w.handleNodeTree(decodeNode)
}
func (w *WaApp) OnHandShakeFailed(err error) {
	// 出现握手失败的表示认证失败
	//log.Println("onHandShakeFailed ", "出现握手失败 -> ", err)
	wslog.GetLogger().Ctx(w.ctx).Error("onHandShakeFailed ", "出现握手失败 -> ", err)
	w.SetLoginStatus(HandshakeFailed)
	w.loginPromise.Reject(err)
}
func (w *WaApp) OnError(err error) {
	log.Println("onError ", "发送错误", err.Error()) //
	if strings.Index(err.Error(), "use of closed network connection") == -1 {
		w.loginStatus = NETNOT
		//登录成功开启
		connectMgr := WXServer.GetWXConnectMgr()
		//查询该链接是否存在
		iwxConnect := connectMgr.GetWXConnectByWXID(w.GetUserName())
		if iwxConnect != nil {
			iwxConnect.Stop()
		}
	} else {
		w.SetLoginStatus(Disconnect)
	}
	w.loginPromise.Reject(err)
}
func (w *WaApp) OnDisconnect() {
	//log.Println("断开连接", "当前用户状态:", w.loginStatus)
	wslog.GetLogger().Ctx(w.ctx).Info("断开连接", "当前用户状态:", w.loginStatus)
	w.NewtWorkClose()
	// 被禁止
	if w.loginStatus == Banned {
		//推送过去上线失败，断开连接
		w.SetLoginStatusText(Banned, "Banned账号被封!")
		return
	}
	if w.loginStatus == NETNOT {
		w.SetLoginStatusText(NETNOT, "网络链接异常!")
		return
	}

	//如果是切代理不需要走下面
	if w.autoLogin {
		return
	}
	/** 因客户要求把掉线自动重连去掉**/
	if w.loginStatus == Drops {
		w.retryLoginWait.Wait()
		return
	}
	// 如果上次登录状态是在线的，断开连接后将状态重置为掉线
	if w.loginStatus == Online {
		//推送过去上线失败，断开连接
		w.SetLoginStatus(Drops)
		//登录成功开启
		connectMgr := WXServer.GetWXConnectMgr()
		//查询该链接是否存在
		iwxConnect := connectMgr.GetWXConnectByWXID(w.GetUserName())
		if iwxConnect != nil {
			iwxConnect.Stop()
		}
		log.Println("retryLoginResult is nil！")
		/*w.SetLoginStatus(Drops)
		log.Println("OnDisconnect", "开始重新尝试上线")
		// 防止重复重新登录
		w.retryLoginWait.Add(1)
		defer w.retryLoginWait.Done()
		// TODO 可进行重连
		retryLoginResult := w.RetryLogin()
		if retryLoginResult != nil {
			_, err := retryLoginResult.GetResult()
			if err != nil {
				//重新上线失败，推送给客户端
				wslog.GetLogger().Ctx(w.ctx).Error("retryLoginResult:", err)
				//推送过去上线失败，断开连接
				w.SetLoginStatusText(Disconnect, err.Error())
				return
			}
			log.Println("重新上线成功！")
		} else {
			//推送过去上线失败，断开连接
			w.SetLoginStatusText(Disconnect, "retryLoginResult is nil！")
			log.Println("retryLoginResult is nil！")
		}*/
	} else {
		// 如果没有登录成功置登录为断开socket 连接
		w.SetLoginStatusText(Disconnect, "handshake fail")
		w.loginPromise.Reject(errors.New("handshake fail"))

	}

}
