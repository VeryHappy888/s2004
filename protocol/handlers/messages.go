package handlers

import (
	"errors"
	"github.com/gogf/gf/container/gmap"
	"github.com/golang/protobuf/proto"
	"log"
	"ws-go/libsignal/exception"
	"ws-go/protocol/axolotl"
	"ws-go/protocol/define"
	"ws-go/protocol/entity"
	"ws-go/protocol/iface"
	"ws-go/protocol/node"
	"ws-go/protocol/waproto"
)

// NewChatMessageHandler 创建 chat message handler
func NewChatMessageHandler(axolotl *axolotl.Manager, api iface.INodeApi) iface.Handler {
	handler := &ChatMessageHandler{
		baseHandler: newBaseHandler(define.HandlerChatMessage, 10),
		NodeApi:     api,
		Axolotl:     axolotl,
		retryList:   gmap.NewStrAnyMap(true),
	}
	// run loop queue
	go handler.LoopQueue()
	return handler
}

// ChatMessageHandler 对聊天消息进行处理
type ChatMessageHandler struct {
	*baseHandler
	NodeApi iface.INodeApi
	Axolotl *axolotl.Manager
	// 重试列表
	retryList *gmap.StrAnyMap
}

// AddHandleTask add chat message decrypt task
func (c *ChatMessageHandler) AddHandleTask(i interface{}) error {
	// nil ptr
	if i == nil {
		return errors.New("Tasks cannot be empty")
	}
	// Is not the handler's processing object
	if _, ok := i.(*entity.ChatMessage); !ok {
		return errors.New("Is not the handler's processing object")
	}
	// add queue
	c.Add(i)
	return nil
}

// Close
func (c *ChatMessageHandler) Close() {
	c.baseHandler.Close()
}

// LoopQueue
func (c *ChatMessageHandler) LoopQueue() {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("LoopQueue panic: %v\n", r)
		}
	}()
	for {
		v := c.queue.Pop()
		if v == nil {
			if !c.queueClose {
				c.queue.Close()
			}
			break
		} else {
			// chat message
			if message, ok := v.(*entity.ChatMessage); ok {
				c.handleChatMessage(message)
			}
			// wa message
			if wamessage, ok := v.(*proto.Message); ok {
				_ = wamessage
			}

		}
	}
}

// sendRetry
func (c *ChatMessageHandler) sendRetry(message *entity.ChatMessage) {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("sendRetry panic: %v\n", r)
		}
	}()
	var retryInfo *entity.RetryInfo
	//moreNode := &newxxmp.Nodes{}
	// message content
	id := message.Id()
	to := message.From()
	participant := message.Participant()
	t := message.T()
	// get retry info
	if !c.retryList.Contains(id) {
		// save retry info
		c.retryList.Set(id, entity.NewRetryInfo(nil, message))
	}
	retryInfo = c.retryList.Get(id).(*entity.RetryInfo)
	// 超过两次解密失败
	if retryInfo.Count().Val() >= 2 {
		// remove retry
		c.retryList.Remove(message.Id())
		c.NodeApi.SendNormalAndRead(message.Id(), message.From(), message.Participant())
		return
	}
	// send retry
	retryInfo.Count().Add(1)
	// SendReceiptRetry
	c.NodeApi.SendReceiptRetry(to, id, participant, t, *retryInfo.Count())
}

//消息回调
func (c *ChatMessageHandler) handleWaMessage(decryptedData []byte, msgInfo *entity.ChatMessage) {
	waMessage := &waproto.WAMessage{}
	//新结构
	message := &waproto.Message{}
	// remove retry
	c.retryList.Remove(msgInfo.Id())
	// unmarshal
	err := proto.Unmarshal(decryptedData, waMessage)
	err = proto.Unmarshal(decryptedData, message)
	if err != nil {
		log.Println("ReceiveHandleMessage unmarshal error", err)
		return
	}
	// 群聊
	if waMessage.GetSKMSG() != nil {
		var groupId, participant node.JId
		skmsg := waMessage.GetSKMSG()
		groupId = node.NewJid(skmsg.GetGROUP_ID())
		participant = node.NewJid(msgInfo.Participant())
		c.Axolotl.ProcessGroupSession(
			groupId.RawId(), participant.RawId(), skmsg.GetSENDER_KEY())
		//return
	}
	if message.GetExtendedTextMessage() != nil && message.GetExtendedTextMessage().GetText() != "" {
		v := message.ExtendedTextMessage.GetText()
		waMessage.CONVERSATION = &v
	}

	// 文本消息
	if waMessage.GetCONVERSATION() != "" {
		log.Println("收到消息 Msg:", waMessage.GetCONVERSATION())
	}
	// set content
	msgInfo.SetContent(waMessage)
	//set message
	msgInfo.SetMessage(message)
	// send receipt
	c.NodeApi.SendNormalAndRead(msgInfo.Id(), msgInfo.From(), msgInfo.Participant())
	// notify
	c.notify(msgInfo)
}

// chatMessageDecryptFailure 解密失败
func (c *ChatMessageHandler) chatMessageDecryptFailure(message *entity.ChatMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("chatMessageDecryptFailure panic: %v\n", r)
		}
	}()
	//log.Println("handleDecryptFailure", err)
	// 统一处理错误
	switch err.(type) {
	case *exception.NoSessionException:
		add := (err.(*exception.NoSessionException)).Addr()
		err := c.NodeApi.GetPreKeys(false, add)
		if err != nil {
			return
		}
		c.handleChatMessage(message)
		break
	case *exception.NoValidSessions:
		// 发送重试
		c.sendRetry(message)
		break
	default:
		// 发送重试
		c.sendRetry(message)
	}
}

// handleChatMessage
func (c *ChatMessageHandler) handleChatMessage(message *entity.ChatMessage) {
	if message == nil || len(message.EncList()) <= 0 {
		return
	}

	// decrypt enc array
	for _, enc := range message.EncList() {
		// decrypt
		//wslog.GetLogger().Debug("ReceiveHandleMessage:data", hex.EncodeToString(enc.Data()))
		jid := node.NewJid(message.From())
		participant := node.NewJid(message.Participant())
		decryptedData, err := c.Axolotl.Decrypt(jid.RawId(), participant.RawId(), enc.Data(), enc.EncType())
		if err != nil {
			c.chatMessageDecryptFailure(message, err)
			break
		}
		// 解密成功发送
		//log.Println("ReceiveHandleMessage decrypt:", hex.EncodeToString(decryptedData))
		// handle
		c.handleWaMessage(decryptedData, message)
	}
}
