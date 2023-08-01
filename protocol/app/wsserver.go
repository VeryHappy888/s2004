package app

type WSServer struct {
	wxConnectMgr IWXConnectMgr
}

// NewWXServer 新建微信服务对象
func NewWXServer() IWXServer {
	return &WSServer{
		wxConnectMgr: NewWXConnManager(),
	}
}

// Start 开启微信服务
func (wxs *WSServer) Start() {
	// 开启微信消息线程池
	//wxs.wxFileMgr.Start()
}

// GetWXConnectMgr 获取微信链接管理器
func (wxs *WSServer) GetWXConnectMgr() IWXConnectMgr {
	return wxs.wxConnectMgr
}
