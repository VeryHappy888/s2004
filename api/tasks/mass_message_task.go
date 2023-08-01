package tasks

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/container/garray"
	"github.com/gogf/gf/os/gtimer"
	"github.com/gogf/gf/util/grand"
	"strings"
	"sync"
	"time"
	"ws-go/protocol/app"
	"ws-go/protocol/entity"
	"ws-go/protocol/msg"
	"ws-go/wslog"
)

const (
	TaskNameMassTask = "MassTask"
)

type MassTaskResult struct {
	Errors  map[string]error
	Success map[string]*msg.MySendMsg
}

func (m *MassTaskResult) AddSuccess(u string, sendMsg *msg.MySendMsg) {
	m.Success[u] = sendMsg
}
func (m *MassTaskResult) AddError(u string, err error) {
	m.Errors[u] = err
}

type MassMessageTask struct {
	TaskBase
	ws     *app.WaApp
	result *MassTaskResult
	// 随机等待时间后发送
	randomWait bool
	// 发送内容
	content string
	// 欲要发送的列表
	numbers *garray.StrArray
	// contacts 同步列表
	contacts *garray.StrArray
	// 已发送列表
	sentList *garray.StrArray
	// 发送条数
	count int32
}

// NewMassMessageTask 群发任务
func NewMassMessageTask(app *app.WaApp, numbers []string, content string, randWait bool) *MassMessageTask {
	m := &MassMessageTask{
		TaskBase: TaskBase{},
		result: &MassTaskResult{
			Errors:  make(map[string]error, 0),
			Success: make(map[string]*msg.MySendMsg, 0),
		},
		ws:         app,
		content:    content,
		randomWait: randWait,
		contacts:   garray.NewStrArray(true),
		numbers:    garray.NewStrArray(true),
		count:      0,
	}
	// set numbers
	m.numbers.SetArray(numbers)
	return m
}

// Stop
func (m *MassMessageTask) Stop() {
	m.status = TaskStop
}

// Worker
func (m *MassMessageTask) Worker() error {
	var resultErr error
	m.numbers.LockFunc(func(array []string) {
		// 停止运行
		if m.status == TaskStop {
			return
		}
		// 同步 WhatsApp 账号
		promise := m.ws.SendSyncContacts(array)
		result, err := promise.GetResult()
		if err != nil {
			resultErr = err
			return
		}
		// result
		iqResult := result.(entity.IqResult)
		for _, info := range iqResult.GetUSyncContacts() {
			if !strings.Contains(info.Status(), "fail") {
				m.contacts.Append(info.Jid())
			}
		}
	})
	//
	if resultErr == nil {
		m.timerSend()
	}
	return resultErr
}

// Result 返回执行结果
func (m *MassMessageTask) Result() (interface{}, error) {
	if m.status == TaskError {
		return nil, m.resultError
	}
	if m.status == TaskComplete {
		return m.result, nil
	}
	return nil, nil
}

// timerSend 定时发送
func (m *MassMessageTask) timerSend() {
	if m.contacts.Len() == 0 {
		m.status = TaskError
		m.resultError = errors.New("Failed to synchronize contacts")
		return
	}

	var (
		sentWait sync.WaitGroup
		waitTime time.Duration
	)
	// 开始设置定时任务
	m.contacts.LockFunc(func(array []string) {
		ctx := m.ws.Ctx()
		wslog.GetLogger().Ctx(ctx).Info("定时任务开始！")
		for _, s := range array {
			sentWait.Add(1)
			if m.randomWait {
				waitTime = time.Duration(time.Second * time.Duration(grand.N(5, 10))) //30  240
			}
			wslog.GetLogger().Ctx(ctx).Info("等待时间，", waitTime)
			// 定时任务
			gtimer.AddOnce(waitTime*time.Second, func() {
				defer sentWait.Done()
				// 发送订阅
				m.ws.SendPresencesSubscribe(s)
				//if err != nil {
				//	//m.result.AddError(s,err)
				//	return
				//}
				// 发送正在输入
				// 发送聊天状态 在输入文字时候发送
				m.ws.SendChatState(s, false, false)
				time.Sleep(time.Second)
				// 发送聊天状态 在输入文字时候发送
				m.ws.SendChatState(s, false, true)
				time.Sleep(time.Second)
				mymsg, err := m.ws.SendTextMessage(s, m.content, nil, "", "", "")
				if err != nil {
					m.result.AddError(s, err)
					return
				}
				m.result.AddSuccess(s, mymsg)
				m.count = m.count + 1
				fmt.Println("已成功发送数量为------------------------------------------------>", m.count)
			})
			sentWait.Wait()
		}
	})
	m.status = TaskComplete
}
