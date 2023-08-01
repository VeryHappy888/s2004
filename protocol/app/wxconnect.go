package app

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	"ws-go/protocol/db"
	"ws-go/protocol/entity"
	"ws-go/protocol/utils/promise"
	"ws-go/wslog"
)

// WXConnect 链接
type WXConnect struct {
	//微信链接管理器
	wxConnectMgr IWXConnectMgr
	// 发送请求缓存队列
	longReqQueue chan IWSRequest
	// 微信链接ID
	wxConnID uint32
	// 微信账号信息
	wxAccount *WaApp
	// 心跳定时器
	heartBeatTimer *time.Timer
	// 断开链接
	ExitFlagChan chan bool
	// 首次登录初始化只执行一次
	onceInit sync.Once
	// 失败次数
	fullSum    int32
	wxConnLock sync.RWMutex //读写连接的读写锁
}

func (wxconn *WXConnect) Stop() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:Stop", err)
		}
	}()
	wxconn.ExitFlagChan <- true
	// 立即过期
	wxconn.heartBeatTimer.Reset(0)
	//移除管理器中的鏈接
	wxconn.wxConnectMgr.Remove(wxconn)
	if wxconn.GetWXAccount() != nil {
		wxconn.GetWXAccount().NewtWorkClose()
		wxconn.GetWXAccount().SetLoginStatus(Disconnect)
	}
}

// NewWXConnect 新的微信连接
func NewWXConnect(wxAccount *WaApp, wxConnectMgr IWXConnectMgr) IWXConnect {
	wxconn := &WXConnect{
		wxConnectMgr: wxConnectMgr,
		longReqQueue: make(chan IWSRequest, 1),
		wxAccount:    wxAccount,
		ExitFlagChan: make(chan bool, 1),
		fullSum:      0,
		wxConnLock:   sync.RWMutex{},
	}
	return wxconn
}

// Start 开启链接任务
func (wxconn *WXConnect) Start() error {
	userInfo := wxconn.wxAccount
	// 判断微信信息是否为空
	if userInfo == nil {
		wxconn.Stop()
		return errors.New("wxconn.Start() err: userInfo == nil")
	}
	// 开启任务管理器
	fmt.Println("[" + userInfo.GetUserName() + "]开始任务状态！")
	// 启动心跳
	wxconn.heartBeatTimer = time.NewTimer(time.Minute * 1)
	go wxconn.startInit()
	go wxconn.startLongWriter()
	// 添加鏈接至鏈接管理器
	wxconn.wxConnectMgr.Add(wxconn)
	return nil
}

// 一些初始化工作
func (wxconn *WXConnect) startInit() {
	w := wxconn.GetWXAccount()
	//如果是商业版
	if w.GetPlatform() == "PLATFORM_10" {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println("run web error:SendGetVerifiedName", err)
				}
			}()
			key := fmt.Sprintf("whatsapp:VerifiedName:%v", w.GetUserName())
			val, err := db.Exists(key)
			if err != nil {
				log.Println("Exists DB失败", err.Error())
			}
			if val {
				resultDB := entity.IqResult{}
				_ = db.GETObj(key, &resultDB)
				number := resultDB.VerifiedName.GetVerifiedOne().GetVerifiedOne1()
				w.SetVeriFiledName(number)
			}
			promise := w.node.SendGetVerifiedName(w.GetUserName())
			result, err := promise.GetResult()
			if err != nil {
				log.Println("获取VerifiedName失败", err.Error())
			} else {
				iqResult := result.(entity.IqResult)
				if iqResult.GetErrorEntityResult() != nil {
					log.Println("获取VerifiedName失败iq-Result", err.Error())
				} else {
					if iqResult.VerifiedName != nil {
						number := iqResult.VerifiedName.GetVerifiedOne().GetVerifiedOne1()
						w.SetVeriFiledName(number)
						_ = db.SETObj(key, iqResult)
					}
				}
			}
		}()
	}
	//添加到隊列
	wxconn.SendToWXLongReqQueue(&WSRequest{
		Wapp: w,
	})
}

// startLongWriter 开启发送数据
func (wxconn *WXConnect) startLongWriter() {
	for {
		select {
		case <-wxconn.longReqQueue:
			SetReqQueueList(wxconn)
			continue
		case <-wxconn.heartBeatTimer.C:
			// 发送心跳包
			SetHeartBeatRequest(wxconn)
			continue
		case <-wxconn.ExitFlagChan:
			return
		}
	}
}

func SetReqQueueList(wxConn *WXConnect) {
	w := wxConn.GetWXAccount()
	fmt.Println(w.GetUserName(), "--->>请求上传key>>>>>>>")
	wxConn.wxConnLock.RLock()
	preKeys := w.axolotlManager.HasUnsentPreKeys()
	wxConn.wxConnLock.RUnlock()
	if preKeys {
		w.node.SendActive()
		// upload pre keys
		err := w.node.SendSetEncryptKeys()
		if err != nil {
			log.Println("update pre keys err", err)
			return
		}
	}
}

func SetHeartBeatRequest(wxConn *WXConnect) {
	w := wxConn.GetWXAccount()
	if w == nil || w.netWork == nil {
		wxConn.Stop()
		return
	}
	if w.loginStatus == Disconnect {
		//fmt.Println("账号退出---")
		wxConn.Stop()
		return
	}
	// 检查网络连接情况
	if !w.netWork.Connected() {
		wxConn.Stop()
		return
	}
	fmt.Println(w.GetUserName() + "--->开始发送心跳包")
	wxConn.wxConnectMgr.ShowConnectInfo()
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
		wslog.GetLogger().Ctx(w.ctx).Info("继续发送 ping")
		wxConn.fullSum = 0 //成功初始化为0
	}
	// failure
	failureFunc := func(err error) {
		db.PushQueue(
			db.PushMsg{
				Time:     time.Now().Unix(),
				UserName: w.clientPayload.GetUsername(),
				Type:     db.System.Number(),
				Data:     "send iq ping failure",
			},
		)
		wxConn.fullSum++
		fmt.Println(w.GetUserName(), "心跳失败", wxConn.fullSum, "次")
		if wxConn.fullSum <= 2 {
			SetHeartBeatRequest(wxConn)
			return
		}
		/*go wxConn.Stop()*/
		wslog.GetLogger().Ctx(w.ctx).Error("send ping failure ")
	}
	w.node.SendIqPing().SetListenHandler(successFunc, failureFunc)
	//fmt.Println("wxConn.fullSum")
	if wxConn.fullSum >= 2 {
		//关闭
		wxConn.Stop()
		return
	}
	//4分钟后在次发送
	wxConn.SendHeartBeatWaitingSeconds(1)
}

// GetWXAccount 获取微信帐号信息
func (wxconn *WXConnect) GetWXAccount() *WaApp {
	return wxconn.wxAccount
}

// SetWXConnID 设置微信链接ID
func (wxconn *WXConnect) SetWXConnID(wxConnID uint32) {
	wxconn.wxConnID = wxConnID
}
func (wxconn *WXConnect) GetWxConnID() uint32 {
	return wxconn.wxConnID
}

// SendHeartBeatWaitingSeconds 添加到微信心跳包队列
func (wxconn *WXConnect) SendHeartBeatWaitingSeconds(seconds uint32) {
	wxconn.heartBeatTimer.Reset(time.Minute * time.Duration(seconds))
}
func (wxconn *WXConnect) SendHeartWaitingSeconds(seconds uint32) {
	wxconn.heartBeatTimer.Reset(time.Duration(seconds))
}

// SendToWXLongReqQueue 添加到请求队列
func (wxconn *WXConnect) SendToWXLongReqQueue(wxLongReq IWSRequest) {
	wxconn.longReqQueue <- wxLongReq
}
