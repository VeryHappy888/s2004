package app

//IWSRequest 检查是否有 未上传的identity key
type IWSRequest interface {
	GetWapp() *WaApp
}
