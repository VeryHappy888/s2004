package app

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gogf/gf/util/gconv"
	"github.com/golang/protobuf/proto"
	"log"
	"time"
	"ws-go/noise"
	"ws-go/protocol/db"
	"ws-go/protocol/handshake"
	"ws-go/protocol/waproto"
)

// loginInfo
type loginInfo struct {
	ctx context.Context
	// login info
	clientPayload  *waproto.ClientPayload
	priKey, pubKey []byte
	routingInfo    []byte
	staticPubKey   string
	staticPriKey   string
}

func (l *loginInfo) SetStaticPubKey(d string) {
	l.staticPubKey = d
}
func (l *loginInfo) SetStaticPriKey(d string) {
	l.staticPriKey = d
}
func (l *loginInfo) Ctx() context.Context {
	return l.ctx
}

func (l *loginInfo) SetRoutingInfo(d []byte) {
	l.routingInfo = d
}

func (l *loginInfo) SetStaticHdBase64Keys(pri, pub string) error {
	priData, err := base64.StdEncoding.DecodeString(pri)
	if err != nil {
		return err
	}
	pubData, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		return err
	}

	if len(priData) != 32 || len(pubData) < 32 {
		return errors.New("keys length less than 32 bit")
	}

	if len(pubData) == 33 {
		pubData = pubData[1:]
	}
	if len(priData) == 33 {
		priData = priData[1:]
	}
	// set keys
	l.priKey, l.pubKey = priData, pubData
	return nil
}

func (l *loginInfo) SetStaticHdKeys(pri, pub []byte) error {
	if len(pri) != 32 || len(pub) != 32 {
		return errors.New("keys length less than 32 bit")
	}
	// set keys
	l.priKey, l.pubKey = pri, pub
	return nil
}

// SetStaticKeys
func (l *loginInfo) SetStaticKeys(base64Data string) error {
	skData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return err
	}
	// if sk data length not 64
	if len(skData) < 64 {
		return errors.New("static keys length less than 64 bit")
	}
	// set static keys
	l.priKey, l.pubKey = skData[:32], skData[32:]
	return nil
}

func (l *loginInfo) SetCliPayload(payload *waproto.ClientPayload) {
	if payload != nil {
		l.clientPayload = payload
	}
}

// SetCliPayloadData
func (l *loginInfo) SetCliPayloadData(data []byte) error {
	return proto.Unmarshal(data, l.clientPayload)
}

// GetLoginSettings
func (l *loginInfo) GetLoginSettings() *handshake.WAHandshakeSettings {
	return &handshake.WAHandshakeSettings{
		RoutingInfo: l.routingInfo,
		Payload:     l.buildClientPayload(),
		StaticKey: noise.DHKey{
			Private: l.priKey,
			Public:  l.pubKey,
		},
		PeerStatic: []byte{},
	}
}

func (l *loginInfo) buildClientPayload() []byte {
	if l.clientPayload == nil {
		return nil
	}
	d, _ := json.MarshalIndent(&l.clientPayload, " ", "  ")
	log.Println("AUTHDATA:", string(d))
	db.PushQueue(
		db.PushMsg{
			Time:     time.Now().Unix(),
			UserName: l.clientPayload.GetUsername(),
			Type:     db.System.Number(),
			Data:     l.clientPayload,
		},
	)
	// marshal
	payloadData, err := proto.Marshal(l.clientPayload)
	if err != nil {
		return nil
	}
	log.Println("payloadData", hex.EncodeToString(payloadData))
	return payloadData
}

// AccountInfo
type AccountInfo struct {
	*loginInfo
	verifiedName uint64
}

func EmptyAccountInfo() *AccountInfo {
	return &AccountInfo{
		loginInfo: &loginInfo{
			ctx:           context.Background(),
			clientPayload: &waproto.ClientPayload{},
			priKey:        []byte{},
			pubKey:        []byte{},
			routingInfo:   []byte{},
		}}
}

// GetUserName
func (a *AccountInfo) GetUserName() string {
	if a.clientPayload != nil {
		return gconv.String(a.clientPayload.GetUsername())
	}
	return ""
}
func (a *AccountInfo) GetVeriFiledName() uint64 {
	if a.verifiedName != 0 {
		return a.verifiedName
	}
	return 0
}
func (a *AccountInfo) SetVeriFiledName(VeriFiledName uint64) {
	a.verifiedName = VeriFiledName
}

// 获取平台
func (a *AccountInfo) GetPlatform() string {
	if a.clientPayload != nil {
		return a.clientPayload.GetUserAgent().GetPlatform().String()
	}
	return "no"
}

// SetLogCtx
func (a *AccountInfo) SetLogCtx(k, v string) {
	a.ctx = context.WithValue(a.ctx, k, v)
}

func (a *AccountInfo) SetUserName(u uint64) {
	if a.clientPayload != nil {
		a.clientPayload.Username = proto.Uint64(u)
		//a.clientPayload.SessionId = proto.Int32(0x0e844d0f)

		//a.clientPayload.UserAgent.PhoneId = proto.String("90196710-7a70-45cf-8d8e-364c40d79296")
	}
}
