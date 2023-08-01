package app

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WXConnectMgr 微信链接管理器
type WXConnectMgr struct {
	canUseConnIDList []uint32 // 删掉/回收后的connID
	currentWxConnID  uint32
	wxConnectMap     map[string]IWXConnect //管理的连接信息
	wxConnLock       sync.RWMutex          //读写连接的读写锁
}

// NewWXConnManager 创建一个WX链接管理
func NewWXConnManager() IWXConnectMgr {
	return &WXConnectMgr{
		canUseConnIDList: make([]uint32, 0),
		currentWxConnID:  0,
		wxConnectMap:     make(map[string]IWXConnect),
		wxConnLock:       sync.RWMutex{},
	}
}

// Add 添加链接
func (wm *WXConnectMgr) Add(wxConnect IWXConnect) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:Add", err)
		}
	}()
	wm.wxConnLock.Lock() //方法将rw锁定为读取状态，禁止其他线程写入，但不禁止读取。
	newConnID := wm.currentWxConnID
	wm.currentWxConnID++
	wxConnect.SetWXConnID(newConnID)
	wm.wxConnectMap[wxConnect.GetWXAccount().GetUserName()] = wxConnect
	defer wm.wxConnLock.Unlock()
	// 打印链接数量
	wm.ShowConnectInfo()
}

// GetWXConnectByUserInfoUUID 根据UserInfoUUID获取微信链接
func (wm *WXConnectMgr) GetWXConnectByUserInfoUUID(userInfoUUID string) IWXConnect {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetWXConnectByUserInfoUUID", err)
		}
	}()
	wm.wxConnLock.Lock()
	defer wm.wxConnLock.Unlock()
	wxConn, ok := wm.wxConnectMap[userInfoUUID]
	if ok {
		fmt.Println(fmt.Sprintf("GET Connection locfree success by %s", userInfoUUID))
		return wxConn
	}
	fmt.Println(fmt.Sprintf("GET Connection locfree Failed by %s  abandon the conntection get  !", userInfoUUID))
	return nil
}

// GetWXConnectByWXID 根据WXID获取微信链接
func (wm *WXConnectMgr) GetWXConnectByWXID(userName string) IWXConnect {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetWXConnectByWXID", err)
		}
	}()
	//保护共享资源Map 加读锁
	wm.wxConnLock.Lock()
	defer wm.wxConnLock.Unlock()
	//根据WXID获取微信链接
	for _, wxConn := range wm.wxConnectMap {
		tmpUserInfo := wxConn.GetWXAccount()
		if tmpUserInfo == nil || strings.Compare(tmpUserInfo.GetUserName(), userName) != 0 {
			continue
		}
		return wxConn
	}
	return nil
}

// Remove 删除连接
func (wm *WXConnectMgr) Remove(wxconn IWXConnect) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:Remove", err)
		}
	}()
	wm.wxConnLock.Lock()
	//删除
	currentUserInfo := wxconn.GetWXAccount()
	delete(wm.wxConnectMap, currentUserInfo.GetUserName())
	//wm.canUseConnIDList = append(wm.canUseConnIDList, wxconn.GetWXConnID())
	currentUserInfo = nil
	// 打印链接数量
	defer wm.wxConnLock.Unlock()
	wm.ShowConnectInfo()
}

// Len 获取当前连接
func (wm *WXConnectMgr) Len() int {
	return len(wm.wxConnectMap)
}

// ClearWXConn 删除并停止所有链接
func (wm *WXConnectMgr) ClearWXConn() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:ClearWXConn", err)
		}
	}()
	//保护共享资源Map 加写锁
	wm.wxConnLock.Lock()
	defer wm.wxConnLock.Unlock()
	//停止并删除全部的连接信息
	for uuid, wxConn := range wm.wxConnectMap {
		//停止
		wxConn.Stop()
		//删除
		delete(wm.wxConnectMap, uuid)
	}
	// 打印链接数量
	wm.ShowConnectInfo()
}

// ShowConnectInfo 打印链接情况
func (wm *WXConnectMgr) ShowConnectInfo() {
	totalNum := wm.Len()
	showText := time.Now().Format("2006-01-02 15:04:05")
	showText = showText + " 总服务数量: " + strconv.Itoa(totalNum)
	fmt.Println(showText)
}

func (wm *WXConnectMgr) GetConnectInfo() string {
	totalNum := wm.Len()
	showText := time.Now().Format("2006-01-02 15:04:05")
	showText = showText + " 总服务数量: " + strconv.Itoa(totalNum)
	fmt.Println(showText)
	return showText
}
