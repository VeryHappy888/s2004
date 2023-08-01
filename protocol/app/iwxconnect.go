package app

// IWXConnect 链接接口
type IWXConnect interface {
	// 开启
	Start() error
	// 关闭
	Stop()
	// 设置链接ID
	SetWXConnID(wxConnID uint32)
	//获取账号信息
	GetWXAccount() *WaApp
	// 获取设置长链接ID
	GetWxConnID() uint32
	// 等待 waitTimes后发送心跳包
	SendHeartBeatWaitingSeconds(seconds uint32)
	// 等待 waitTimes后发送心跳包
	SendHeartWaitingSeconds(seconds uint32)
	// 添加到长链接请求队列
	SendToWXLongReqQueue(wxLongReq IWSRequest)
}
