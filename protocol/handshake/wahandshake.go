package handshake

import (
	"crypto/rand"
	"errors"
	"github.com/golang/protobuf/proto"
	"log"
	"ws-go/noise"
	_interface "ws-go/protocol/iface"
	"ws-go/protocol/waproto"
)

var DHString = []byte{69, 68, 0, 1}
var InitString = []byte("WA")

type VerifyCallbackFunc func(publicKey []byte, data []byte) error

// WAProtocolVersion noise 握手版本
type WAProtocolVersion struct {
	// 主要版本
	VersionMajor byte
	// 次要版本
	VersionMinor byte
}

type WAHandshakeSettings struct {
	RoutingInfo []byte
	//认证数据
	Payload []byte //certificates, signs etc
	// 用户注册时生成的公私秘钥
	StaticKey noise.DHKey
	// 服务器公钥
	PeerStatic []byte
}

type WAHandshake struct {
	// WhatsApp 协议版本
	WaProtocolVersion WAProtocolVersion

	*WAHandshakeSettings
	VerifyCallback VerifyCallbackFunc
	netWorkSegment _interface.SegmentProcessor
	csIn, csOut    *noise.CipherState
}

// RunHandshake 开始进行握手
func (w *WAHandshake) RunHandshake(segment _interface.SegmentProcessor) error {
	if segment != nil {
		w.netWorkSegment = segment
	}

	// send routing info data
	if len(w.RoutingInfo) > 0 {
		routingInfoLen := len(w.RoutingInfo)
		lenData := make([]byte, 3)
		lenData[2] = byte(routingInfoLen)
		lenData[1] = (byte)(routingInfoLen >> 8)
		lenData[0] = (byte)(routingInfoLen >> 16)
		newRouting := append(DHString, lenData...)
		_, _ = w.netWorkSegment.(_interface.INetWork).GetConn().Write(append(newRouting, w.RoutingInfo...))
	}

	// send handshake header data
	HEADER := append(InitString, w.WaProtocolVersion.VersionMajor, w.WaProtocolVersion.VersionMinor)
	_, _ = w.netWorkSegment.(_interface.INetWork).GetConn().Write(HEADER)
	// 开始进行握手
	if len(w.PeerStatic) == 0 {
		return w.startHandshakeXX()
	} else {
		//TODO 当有服务器秘钥使用 IX进行握手
	}
	return nil
}

// GetCipherStateGroup 握手成功后的密钥对
func (w *WAHandshake) GetCipherStateGroup() (csIn *noise.CipherState, csOut *noise.CipherState) {
	return w.csIn, w.csOut
}

// UpdateHandshakeSettings 更新握手配置
func (w *WAHandshake) UpdateHandshakeSettings(i interface{}) {
	if settings, ok := i.(*WAHandshakeSettings); ok {
		w.WAHandshakeSettings = settings
	}
}

// start_handshake_xx
func (w *WAHandshake) startHandshakeXX() error {

	var (
		state   *noise.HandshakeState
		outData []byte
		err     error
	)
	// {87, 65, 4, 1};
	prologue := append(InitString, []byte{w.WaProtocolVersion.VersionMajor, w.WaProtocolVersion.VersionMinor}...)
	// 初始化noise协议
	state, err = noise.NewHandshakeState(noise.Config{
		StaticKeypair: w.StaticKey,
		Initiator:     true,
		Pattern:       noise.HandshakeXX,
		CipherSuite:   noise.NewCipherSuite(noise.DH25519, noise.CipherAESGCM, noise.HashSHA256),
		PeerStatic:    w.PeerStatic,
		Prologue:      prologue,
		Random:        rand.Reader,
	})
	if err != nil {
		return err
	}

	outData, _, _, err = state.WriteMessage(outData, []byte{})
	if err != nil {
		return err
	}

	// 序列化客户端握手数据
	clientHelloProtoData, err := serializationClientHello(outData, nil, nil)
	if err != nil {
		return err
	}
	/*fmt.Println("clientHelloData - ")
	fmt.Println(hex.Dump(clientHelloProtoData))*/
	//发送公钥
	//clientHelloProtoData,_= hex.DecodeString("12220a205fad5480850e05ddf2f49e8edcfd87715a5a279a14836d5187f8b406dd28c844")
	err = w.netWorkSegment.WriteSegmentOutputData(clientHelloProtoData)
	if err != nil {
		return err
	}

	serverHelloData, err := w.netWorkSegment.ReadInputSegmentData()
	if err != nil {
		return err
	}
	serverHello, err := unmarshalServerHello(serverHelloData)
	if err != nil {
		return err
	}
	//, _ := hex.DecodeString("08b69eb59a830d18012a5f080012070802101418c4011a03343630220230312a05382e302e30320773616d73756e673a04616161614205382e302e304a2461383963316639392d326337362d343236662d623431312d3165376235663264316639665a02656e620255533a06796f777375704d981fc15350016001")

	log.Println("Ephemeral")
	//fmt.Println(hex.Dump(serverHello.Ephemeral))
	log.Println("Static")
	//fmt.Println(hex.Dump(serverHello.Static))
	//fmt.Println("Ephemeral",hex.EncodeToString(serverHello.Ephemeral))

	// 进行握手解密拿到公钥和和校验证书
	outData = make([]byte, 0)
	message := append(serverHello.Ephemeral, serverHello.Static...)
	message = append(message, serverHello.Payload...)
	outData, _, _, err = state.ReadMessage(outData, message)
	if err != nil {
		return err
	}
	// 是否有证书
	if len(state.PeerStatic()) > 0 {
		_ = w.processCallback(state.PeerStatic(), outData)
	}

	//最后发送握手完成包
	if len(w.Payload) == 0 {
		return errors.New("No information to Payload")
	}

	payload, csIn, csOut, err := state.WriteMessage([]byte{}, w.Payload)
	if err != nil {
		return err
	}
	//fmt.Println(csIn,csOut)

	// 创建最终握手数据
	clientFinishData, err := createClientFinishData(payload[:48], payload[48:])
	if err != nil {
		return err
	}

	if err := w.netWorkSegment.WriteSegmentOutputData(clientFinishData); err != nil {
		return err
	}

	if csOut != nil && csIn != nil {
		w.csIn = csOut
		w.csOut = csIn
	}

	//d, err := w.netWorkSegment.ReadInputSegmentData()
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println("d------",hex.EncodeToString(d))

	/*if err := c.readPacket();err != nil {
		return err
	}*/
	return nil
}
func (w *WAHandshake) processCallback(publicKey []byte, payload []byte) error {
	if w.VerifyCallback == nil {
		return nil
	}

	err := w.VerifyCallback(publicKey, payload)
	return err
}
func createClientFinishData(static, payload []byte) ([]byte, error) {

	message := waproto.HandshakeMessage{}
	message.ClientFinish = &waproto.HandshakeMessage_ClientFinish{
		Static:  static,
		Payload: payload,
	}
	return proto.Marshal(&message)
}
func unmarshalServerHello(data []byte) (*waproto.HandshakeMessage_ServerHello, error) {
	message := &waproto.HandshakeMessage{}
	err := proto.Unmarshal(data, message)
	if err != nil {
		return nil, err
	}
	return message.ServerHello, nil
}
func serializationClientHello(ephemeral, static, payload []byte) ([]byte, error) {
	message := &waproto.HandshakeMessage{}
	message.ClientHello = &waproto.HandshakeMessage_ClientHello{
		Ephemeral: ephemeral,
		Static:    nil,
		Payload:   nil,
	}
	return proto.Marshal(message)
}
