package app

// IWXConnectMgr 微信链接管理器
type IWXConnectMgr interface {
	Add(wxConnect IWXConnect)                                  // 添加链接
	GetWXConnectByUserInfoUUID(userInfoUUID string) IWXConnect // 根据accountId获取微信链接
	GetWXConnectByWXID(wxid string) IWXConnect                 // 根据WXID获取微信链接
	Remove(wxconn IWXConnect)                                  // 删除连接
	Len() int                                                  // 获取当前连接
	ClearWXConn()                                              // 删除并停止所有链接
	ShowConnectInfo()                                          // 打印链接数量
	GetConnectInfo() string
}
