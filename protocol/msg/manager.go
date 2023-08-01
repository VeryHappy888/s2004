package msg

import (
	"errors"
	"github.com/gogf/gcache-adapter/adapter"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/gcache"
	"ws-go/protocol/define"
)

// errors
var IdEmptyErr = errors.New("The message ID is empty")

type MySendMsg struct {
	Id      string
	To      string
	Content string
	MsgType string
	Status  define.MsgStatus
}

// CreateNewMsg
func CreateMySendMsg(to, content, msgType string) *MySendMsg {
	return &MySendMsg{
		To:      to,
		Content: content,
		MsgType: msgType,
	}
}
func (m *MySendMsg) ChangeStatus(new define.MsgStatus) {
	m.Status = new
}

type Manager struct {
	msgCache *gcache.Cache
	//MsgList *gmap.StrAnyMap
}

// NewManager
func NewManager() *Manager {
	m := &Manager{msgCache: gcache.New()}
	// change cache adapter
	m.msgCache.SetAdapter(adapter.NewRedis(g.Redis()))
	return m
}

// AddNewMsg add new send message
func (m *Manager) AddMySendMsg(id string, newMsg *MySendMsg) error {
	if id == "" && newMsg == nil {
		return errors.New("")
	}
	// save
	newMsg.Id = id
	return nil //m.msgCache.Set(id, newMsg, 0)
}

// UpdateMsgStatus 更改消息状态
func (m *Manager) UpdateMsgStatus(id string, status define.MsgStatus) error {
	if id == "" {
		return IdEmptyErr
	}
	// 从缓存中取出我发送的消息
	v, err := m.msgCache.Get(id)
	if err != nil {
		return err
	}
	// 断言
	if s, ok := v.(string); ok {
		msg := &MySendMsg{}
		err := gjson.DecodeTo(s, msg)
		if err != nil {
			return err
		}
		// 更改状态
		msg.ChangeStatus(status)
		// 更新缓存
		_, _, err = m.msgCache.Update(id, msg)
		if err != nil {
			return err
		}
	}
	return nil
}
