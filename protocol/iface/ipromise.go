package iface

import "ws-go/protocol/utils/promise"

type IPromise interface {
	SetPromise(promise *promise.Promise)
	GetPromise() *promise.Promise
	GetResult() (promise.Any, error)
	SetListenHandler(success func(any promise.Any), failure func(err error))
	SetNewListenHandler(handler *PromiseHandler)
}

type IPromiseHandler interface {
	SuccessFunc(any promise.Any)
	FailureFunc(err error)
}

type PromiseHandler struct {
	SuccessFunc func(any promise.Any)
	FailureFunc func(err error)
}
