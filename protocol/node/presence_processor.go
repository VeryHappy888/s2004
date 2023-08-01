package node

import (
	"context"
	"errors"
	"github.com/gogf/gf/container/gmap"
	"github.com/gogf/gf/os/gtimer"
	"log"
	"strings"
	"time"
	_struct "ws-go/protocol/entity"
	"ws-go/protocol/newxxmp"
	"ws-go/protocol/utils/promise"
)

const NodePresence = "presence"

// PresenceNode
type PresenceNode struct {
	*BaseNode
	promise *promise.Promise
	id      string
}

// Process
func (p *PresenceNode) Process(node *newxxmp.Node) {
	if node == nil {
		return
	}
	// 通知
	if &p != nil {
		attributeType := node.GetAttribute("type")
		if attributeType == nil {
			attributeType = &newxxmp.Attribute{}
		}
		attributeLast := node.GetAttribute("last")
		if attributeLast == nil {
			attributeLast = &newxxmp.Attribute{}
		}
		online := ""
		if attributeType != nil {
			online = attributeType.Value()
		}
		last := ""
		if attributeLast != nil {
			last = attributeLast.Value()
		}
		p.SuccessNotice(_struct.PresenceResult{
			From:   node.GetAttribute("from").Value(),
			Online: online,
			Last:   last,
		})
	}
}

//createPresenceAvailable
func createPresenceAvailableNode(name string) *PresenceNode {
	//<presence type="available"/>
	p := &PresenceNode{id: "", BaseNode: NewBaseNode()}
	n := &newxxmp.Node{
		Tag: NodePresence,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("type", "available"),
		},
	}
	if name != "" {
		n.Attributes.AddAttr("name", name)
	}
	p.Node = n
	return p
}

// createPresencesSubscribeNode
func createPresencesSubscribeNode(u string) *PresenceNode {
	//<presence type="subscribe" to="xxxxx@s.whatsapp.net"/>
	// default promise 超时100秒

	if !strings.Contains(u, "@s.whatsapp.net") {
		u += "@s.whatsapp.net"
	}
	p := &PresenceNode{id: u, BaseNode: NewBaseNode(), promise: promise.New(nil)}
	// create node
	n := &newxxmp.Node{
		Tag: NodePresence,
		Attributes: []newxxmp.Attribute{
			newxxmp.NewAttribute("type", "subscribe"),
			newxxmp.NewAttribute("to", u),
		},
	}
	p.Node = n
	return p
}

func NewPresenceProcessor() *PresenceProcessor {
	return &PresenceProcessor{_nodeList: gmap.NewStrAnyMap(true)}
}

// PresenceProcessor
type PresenceProcessor struct {
	_nodeList *gmap.StrAnyMap
}

// SaveNode
func (p *PresenceProcessor) SaveNode(id string, i interface{}) {
	p._nodeList.Set(id, i)
}

// RemoveNode
func (p *PresenceProcessor) RemoveNode(id string) interface{} {
	return p._nodeList.Remove(id)
}

// catchTimeOut
//当超时,从列表移除节点 和上面 的catchRemoveNode 不同 catchTimeOut 创建了个新的 context timeout
// 可以重新定义超时时间
func (p *PresenceProcessor) catchTimeOut(id string, duration time.Duration) *promise.Promise {
	return promise.New(func(resolve func(promise.Any), reject func(error)) {
		timeout, _ := context.WithTimeout(context.Background(), duration)
		<-timeout.Done()
		if p.RemoveNode(id) == nil {
			return
		}
		log.Println("p1 catchTimeOut error", "remove node id->", id)
		reject(errors.New("run time our!" + " id -> " + id))
	}).Catch(func(err error) error {
		// time out remove node
		return err
	})
}
func (i *PresenceProcessor) SetNodeTimeOutRemove(
	id string, node interface{}, interval time.Duration, callBack func()) {
	if interval == 0 {
		// default 10 second
		interval = time.Second * 10
	}
	// add to list
	i._nodeList.Set(id, node)
	// add timer
	gtimer.AddOnce(interval, func() {
		if i.RemoveNode(id) != nil {
			// 移除成功回调
			callBack()
		}
	})

	/*id gtype.Int32, node interface{},interval time.Duration,callBack func()) {
	if interval == 0 {
		// default 10 second
		interval = time.Second * 10
	}
	// add to list
	i._nodeList.Set(int(id.Val()),node)
	// add timer
	gtimer.AddOnce(interval, func() {
		if i.RemoveNode(int(id.Val())) != nil {
			// 移除成功回调
			callBack()
		}
	})*/
}

// Handle 处理 Presence 相关 node
func (p *PresenceProcessor) Handle(node *newxxmp.Node) {
	if node == nil || node.GetTag() != NodePresence {
		return
	}

	fromAttribute := node.GetAttribute("from")
	if fromAttribute == nil {
		return
	}
	var fromId string
	// 尝试从列表获取
	if strings.Contains(fromAttribute.Value(), "@") {
		split := strings.Split(fromAttribute.Value(), "@")
		fromId = split[0]
	}
	removeNode := p.RemoveNode(fromId)
	if removeNode != nil {
		if preNode, ok := removeNode.(*PresenceNode); ok {
			if preNode != nil && node != nil {
				preNode.Process(node)
			}
		}
	}
}

// BuildPresencesSubscribe 发送订阅
func (p *PresenceProcessor) BuildPresencesSubscribe(u string) *PresenceNode {
	build := createPresencesSubscribeNode(u)
	// set promise
	build.SetPromise(p.catchTimeOut(u, time.Second*5))
	p.SaveNode((&JId{S: u}).RawId(), build)
	/*return build
	p.SetNodeTimeOutRemove((&JId{S: u}).RawId(), build, time.Second*10, func() {
		build.promise.Reject(fmt.Errorf("iq BuildPresencesSubscribe time out id:%s", (&JId{S: u}).RawId()))
	})*/
	return build
}

//BuildPresenceAvailable 登录成功时发送
func (p *PresenceProcessor) BuildPresenceAvailable(name string) *PresenceNode {
	subscribeNode := createPresenceAvailableNode(name)
	return subscribeNode
}

func (p *PresenceProcessor) Close() {
	p._nodeList.Clear()
}
