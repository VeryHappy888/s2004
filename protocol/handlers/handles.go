package handlers

import (
	"github.com/gogf/gf/container/gqueue"
	"sync"
	"ws-go/protocol/iface"
)

type Handlers struct {
	hs   map[string]iface.Handler
	lock sync.RWMutex
}

func NewHandles() *Handlers {
	return &Handlers{hs: map[string]iface.Handler{}, lock: sync.RWMutex{}}
}
func (h *Handlers) GetHandler(s string) (handler iface.Handler, fund bool) {
	h.lock.RLock()
	handler, fund = h.hs[s]
	defer h.lock.RUnlock()
	return
}
func (h *Handlers) AddHandler(handler iface.Handler) {
	h.lock.RLock()
	if h.hs == nil {
		h.hs = make(map[string]iface.Handler, 0)
	}
	// set map
	h.hs[handler.Tag()] = handler
	defer h.lock.RUnlock()
}

func (h *Handlers) Close() {
	h.lock.RLock()
	for _, handler := range h.hs {
		handler.Close()
	}
	defer h.lock.RUnlock()
}

func newBaseHandler(tag string, limit ...int) *baseHandler {
	return &baseHandler{
		tag:        tag,
		queueClose: false,
		queue:      gqueue.New(limit...),
		event:      nil,
	}
}

type baseHandler struct {
	tag string
	// 队列是否关闭
	queueClose bool

	queue *gqueue.Queue
	event iface.HandleCallBackEvent
}

func (b *baseHandler) Add(i interface{}) {
	if !b.queueClose {
		b.queue.Push(i)
	}
}
func (b *baseHandler) Close() {
	if !b.queueClose {
		b.queue.Close()
		b.queueClose = true
	}
}
func (b *baseHandler) SetNotifyEvent(event iface.HandleCallBackEvent) {
	b.event = event
}
func (b *baseHandler) notify(any interface{}) {
	if b.event == nil {
		return
	}
	// 使用协程进行通知不然。会被阻塞影响其他消息
	go b.event.NotifyHandleResult(&HandleResult{tag: b.tag, any: any})
}
func (b *baseHandler) Tag() string {
	return b.tag
}

// HandleResult 统一返回格式
type HandleResult struct {
	tag string
	any interface{}
}

// TargetTag return target handler tag
func (h *HandleResult) TargetTag() string {
	return h.tag
}

// GetResult return handler result
func (h *HandleResult) GetResult() interface{} {
	return h.any
}
