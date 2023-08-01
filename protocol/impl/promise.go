package impl

import (
	"errors"
	_interface "ws-go/protocol/iface"
	"ws-go/protocol/utils/promise"
)

type ResultPromise struct {
	*promise.Promise
}

func NewResultPromise() *ResultPromise {
	return &ResultPromise{}
}

func (r *ResultPromise) SetPromise(promise *promise.Promise) {
	r.Promise = promise
}

func (r *ResultPromise) GetPromise() *promise.Promise {
	return r.Promise
}

// GetResult 等待返回数据阻塞
func (r *ResultPromise) GetResult() (promise.Any, error) {
	if r.Promise == nil {
		return nil, errors.New("promise not set")
	}
	return r.Promise.Await()
}

// SetNewListenHandler 用于回调 无阻塞
func (r *ResultPromise) SetNewListenHandler(handler *_interface.PromiseHandler) {
	if r.Promise == nil && handler == nil {
		return
	}
	// 回调成功
	if handler.SuccessFunc != nil {
		r.Promise.Then(func(data promise.Any) promise.Any {
			handler.SuccessFunc(data)
			return nil
		})
	}
	// 回调报错
	if handler.FailureFunc != nil {
		r.Promise.Catch(func(err error) error {
			handler.FailureFunc(err)
			return err
		})
	}
}

// SetCallBack 用于回调 无阻塞
func (r *ResultPromise) SetListenHandler(success func(any promise.Any), failure func(err error)) {
	if r.Promise == nil {
		return
	}
	// 回调成功
	if success != nil {
		r.Promise.Then(func(data promise.Any) promise.Any {
			success(data)
			return nil
		})
	}
	// 回调报错
	if failure != nil {
		r.Promise.Catch(func(err error) error {
			failure(err)
			return err
		})
	}
}
