package app

type WSRequest struct {
	Wapp *WaApp
}

func (iw *WSRequest) GetWapp() *WaApp {
	return iw.Wapp
}
