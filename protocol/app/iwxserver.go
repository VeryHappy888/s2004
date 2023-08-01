package app

// IWXServer 微信服务
type IWXServer interface {
	Start()
	GetWXConnectMgr() IWXConnectMgr
}
