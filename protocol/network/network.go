package network

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"ws-go/noise"
	"ws-go/protocol/handshake"
	"ws-go/protocol/iface"
	"ws-go/wslog"
)

type CallbackEvent interface {
	OnConnectEvent()
}

type NetWork struct {
	net.Conn
}

func (n *NetWork) GetConn() net.Conn {
	return n.Conn
}
func (n *NetWork) SetConn(c net.Conn) {
	n.Conn = c
}

// WASegment whatsapp
type WASegment struct {
	iface.INetWork
	csIn, csOut *noise.CipherState
}

// WriteSegmentOutputData
func (w *WASegment) WriteSegmentOutputData(d []byte) error {
	if w.csOut != nil {
		d = w.csOut.Encrypt([]byte{}, []byte{}, d)
	}

	dataLength := len(d)
	if dataLength > 16777216 {
		panic(errors.New("data too large to write; length > 16777216"))
	}

	buffer := bytes.Buffer{}
	headerLen := w.writHeaderLen(dataLength)
	buffer.Write(headerLen)
	buffer.Write(d)
	//处理下
	if w.GetConn() != nil && buffer.Bytes() != nil {
		_, err := w.GetConn().Write(buffer.Bytes())
		if err != nil {
			return err
		}
	}
	//wslog.GetLogger().Debug("WriteSegmentOutputData => ", len(d), hex.EncodeToString(buffer.Bytes()))
	return nil
}

// ReadInputSegmentData
func (w *WASegment) ReadInputSegmentData() ([]byte, error) {
	DataLen, err := w.readHeaderLen()
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, DataLen)
	// read A complete package
	if _, err := io.ReadFull(w.GetConn(), buffer); err != nil {
		return nil, err
	}
	//var nn int
	//for {
	//	n, err = w.GetConn().Read(buffer[nn:])
	//	if err != nil {
	//		return nil, err
	//	}
	//	// continue read
	//	log.Printf("read n %d,%d,%d\n", n, nn, DataLen)
	//	if uint32(n) > DataLen {
	//		break
	//	}
	//	DataLen = DataLen - uint32(n)
	//	nn = nn + n
	//	if DataLen <= 0 {
	//		break
	//	}
	//}

	// decrypt
	if w.csIn != nil {
		buffer, err = w.csIn.Decrypt([]byte{}, nil, buffer)
	}

	//wslog.GetLogger().Debug("ReadInputSegmentData => ", len(buffer), hex.EncodeToString(buffer))
	return buffer, err
}

// SetCiphersStateGroup
func (w *WASegment) SetCiphersStateGroup(csIn, csOut *noise.CipherState) {
	if csIn != nil && csOut != nil {
		w.csIn = csIn
		w.csOut = csOut
	}
}

// readHeaderLen 取包的长度
func (w *WASegment) readHeaderLen() (uint32, error) {
	wslog.GetLogger().Debug("start read header len.")
	DataLenBytes := make([]byte, 3)
	n, err := w.GetConn().Read(DataLenBytes)
	if err != nil {
		return 0, err
	}

	if n != 3 {
		return 0, errors.New("header len != 3")
	}

	b := uint32(DataLenBytes[2]) | uint32(DataLenBytes[1])<<8 | uint32(DataLenBytes[0])<<16
	wslog.GetLogger().Debug("Parse the length of the packet:", b)
	return b, nil
}

// writeHeaderLen
func (w *WASegment) writHeaderLen(i int) []byte {
	wslog.GetLogger().Debug("Length of write data packet body ", i)
	bArr := make([]byte, 3)
	bArr[2] = byte(i)
	bArr[1] = (byte)(i >> 8)
	bArr[0] = (byte)(i >> 16)
	return bArr

}

type noiseState struct {
	handshake bool
	connected bool
}

func (n *noiseState) Connected() bool {
	return n.connected
}
func (n *noiseState) SetConnected(connected bool) {
	n.connected = connected
}

// NewNoiseClient
// @p:认证参数
// @sk:注册时创建密钥对
func NewNoiseClient(routingInfo, p []byte, sk noise.DHKey, events iface.NetWorkHandler) *NoiseNetWork {
	noiseClient := &NoiseNetWork{
		events:           events,
		segmentProcessor: &WASegment{INetWork: &NetWork{}},
		handshake: &handshake.WAHandshake{
			WaProtocolVersion: handshake.WAProtocolVersion{
				VersionMajor: 5,
				VersionMinor: 2,
			},
			WAHandshakeSettings: &handshake.WAHandshakeSettings{
				RoutingInfo: routingInfo,
				Payload:     p,
				PeerStatic:  []byte{},
				StaticKey:   sk,
			},
		},
	}
	return noiseClient
}

// NoiseNetWork
type NoiseNetWork struct {
	noiseState
	NoiseClientConfig

	c      net.Conn
	events iface.NetWorkHandler
	// segmentProcessor handler recv and write data
	segmentProcessor iface.SegmentProcessor
	handshake        iface.IHandshake
	// waitGroup
	wg sync.WaitGroup
}

func (n *NoiseNetWork) GetSegment() iface.SegmentProcessor {
	return n.segmentProcessor
}

// Reset 重置
func (n *NoiseNetWork) Reset() {
	// 先关闭连接
	n.Close()
	n.segmentProcessor = &WASegment{INetWork: &NetWork{}}
}

// Connect 建立连接
func (n *NoiseNetWork) Connect(handshakeSettings ...interface{}) error {
	// update handshake settings
	if len(handshakeSettings) > 0 && handshakeSettings[0] != nil {
		n.handshake.UpdateHandshakeSettings(handshakeSettings[0])
	}
	// proxy
	proxyDialer, err := n.GetNetWorkProxy()
	if err != nil && err != NetProxyEmptyError {
		return err
	}
	// whatsapp server addr
	serverAddr := "g.whatsapp.net:443"
	// proxyDialer not nil use network Proxy
	if proxyDialer != nil {
		n.c, err = proxyDialer.Dial("tcp6", serverAddr)
		if err != nil {
			fmt.Println("--->", err.Error())
			return err
		}
	} else {
		n.c, err = net.Dial("tcp4", serverAddr)
		if err != nil {
			return err
		}
	}
	//set segmentProcessor Conn
	if netWork, ok := n.segmentProcessor.(iface.INetWork); ok {
		netWork.SetConn(n.c)
	}
	// set connect successful
	n.noiseState.SetConnected(true)
	// call handler
	n.handleConnectEvent()
	return nil
}

// Close
func (n *NoiseNetWork) Close() {
	if !n.connected {
		return
	}
	if n == nil || n.c == nil {
		log.Println("NoiseNetWork 关闭失败!")
		return
	}
	_ = n.c.Close()
	// 等待其他线程的关闭
	n.wg.Wait()
	// 关闭
	n.SetConnected(false)
	//TODO 回调关闭事件
	n.closeNotify()

}

// recvThread 启动一条线程接收数据
func (n *NoiseNetWork) recvThread() {
	n.wg.Add(1)
	for {
		if n.segmentProcessor == nil {
			//n.segmentProcessor
			//TODO 建议 使用默认 segment
			break
		}
		data, err := n.segmentProcessor.ReadInputSegmentData()
		if err != nil {
			if err == io.EOF {
				// TODO 直接跳出循环 call Close 关闭后会通知 连接关闭
				break
			}
			//TODO 其他 error
			n.errorNotify(err)
			break
		}
		n.handleRecvDataEvent(data)
	}

	n.wg.Done()
	// 退出for 循环意味着连接关闭了 或者发生了错误
	// 关闭掉链接
	if n.connected {
		n.Close()
	}
}

// handleConnectEvent 处理链接成功
func (n *NoiseNetWork) handleConnectEvent() {
	// notify connect success
	n.connectNotify()
	wslog.GetLogger().Info("noise connect successful")
	// enter next handshake
	if n.handshake != nil && n.connected {
		wslog.GetLogger().Info("start enter noise handshake....")
		err := n.handshake.RunHandshake(n.segmentProcessor)
		if err != nil {
			// TODO 先通知握手失败
			wslog.GetLogger().Error("handshake err ", err)
			if err == io.EOF {
				n.Close()
			} else {
				n.Close()
				// TODO call handshakeNotify
				n.handshakeNotify(err)
			}
			return
		}

		wslog.GetLogger().Info("noise handshake successful !")
		csIn, csOut := n.handshake.GetCipherStateGroup()
		_, _ = csIn, csOut
		if csOut != nil && csIn != nil {
			wslog.GetLogger().Debug("set segmentProcessor Ciphers")
			if iCiphersStateGroup, ok := n.segmentProcessor.(iface.ICiphersStateGroup); ok {
				iCiphersStateGroup.SetCiphersStateGroup(csIn, csOut)
			}
		}
	}
	// 握手失败连接会关闭
	if n.connected {
		go n.recvThread()
	}

}

// handleRecvDataEvent 处理接收数据
func (n *NoiseNetWork) handleRecvDataEvent(d []byte) {
	// call
	if n.events != nil {
		go n.events.OnRecvData(d)
	}
}

// closeNotify 通知连接关闭
func (n *NoiseNetWork) closeNotify() {
	if n.events != nil {
		// 使用协成
		go n.events.OnDisconnect()
	}
}

// connectNotify 连接成功通知
func (n *NoiseNetWork) connectNotify() {
	if n.events != nil {
		n.events.OnConnect()
	}
}

// handshakeNotify 通知握手失败
func (n *NoiseNetWork) handshakeNotify(err error) {
	// 有错误信息进行通知
	if n.events != nil && err != nil {
		n.events.OnHandShakeFailed(err)
	}
}

// errorNotify 通知 连接发生错误
func (n *NoiseNetWork) errorNotify(err error) {
	if n.events != nil {
		n.events.OnError(err)
	}
}
