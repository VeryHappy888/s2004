package app

// 長連接服务功能
var WXServer IWXServer

func ServerStart() {
	// 开启服务器管理
	wxServer := NewWXServer()
	WXServer = wxServer
}
