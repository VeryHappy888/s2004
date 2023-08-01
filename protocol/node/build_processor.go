package node

import (
	"github.com/gogf/gf/container/gqueue"
	"io"
	"log"
	_interface "ws-go/protocol/iface"
)

type processor struct {
	close         bool
	segmentOutput _interface.SegmentOutputProcessor
	sendQueue     *gqueue.Queue
}

func Processor(s _interface.SegmentOutputProcessor) *processor {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("Processor panic: %v\n", r)
		}
	}()
	p := &processor{
		segmentOutput: s,
		sendQueue:     gqueue.New(10),
	}

	// run
	go func() {
		p.runSendQueue()
	}()
	return p
}

// Close
func (p *processor) Close() bool {
	if p.close {
		p.sendQueue.Close()
		p.close = false
	}
	return true
}

// SendBuilder
func (p *processor) SendBuilder(b _interface.NodeBuilder) {
	// Confirm whether the queue is closed and build
	if !p.close && b == nil {
		return
	}
	p.sendQueue.Push(b)
}

// SendData 这个应该放到 NetWork里
func (p *processor) SendData(d []byte) error {
	return p.segmentOutput.WriteSegmentOutputData(d)
}

// runSendChan
func (p *processor) runSendQueue() {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("runSendQueue panic: %v\n", r)
		}
	}()
	p.close = true

	for {
		v := p.sendQueue.Pop()
		if v == nil {
			if p.close {
				p.Close()
			}
			// 退出队列
			log.Println("Exit run send queue")
			break
		} else {
			if builder, ok := v.(_interface.NodeBuilder); ok {
				nodeData, err := builder.Builder()
				//wslog.GetLogger().Debug("send builder", hex.EncodeToString(nodeData), err)
				err = p.SendData(nodeData)
				if err != nil {
					if err == io.EOF {
						//TODO 连接断开了
						break
					}
					// TODO 其他错误
					if ipromise, ok := v.(_interface.IPromise); ok {
						if promise := ipromise.GetPromise(); promise != nil {
							promise.Reject(err)
						}
					}
				}
			} else {
				log.Println("I got the wrong object", v)
			}
		}
	}
	// 关闭队列
	if p.close {
		p.sendQueue.Close()
	}
}
