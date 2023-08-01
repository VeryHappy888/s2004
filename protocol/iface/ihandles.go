package iface

type IBaseHandler interface {
	Tag() string
	SetNotifyEvent(event HandleCallBackEvent)
}

type IHandlers interface {
	GetHandler(s string) (handler Handler, fund bool)
	AddHandler(handler Handler)
	Close()
}

type Handler interface {
	IBaseHandler
	AddHandleTask(i interface{}) error
	Close()
}

type HandleCallBackEvent interface {
	NotifyHandleResult(any ...interface{})
}
