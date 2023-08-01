package test

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"ws-go/noise"
	"ws-go/protocol/network"
	"ws-go/protocol/waproto"
)

func TestData(t *testing.T) {
	piarKey, _ := base64.StdEncoding.DecodeString("MHWEKRG2N67e5bWAZWOShNLIH5NK9fa5nQZ4IdUWxHLhkZTK7X6gBT0+zEm1IB12IFK9fNXKz50fI5Sw03eASQ==")
	t.Log(base64.StdEncoding.EncodeToString(piarKey[:32]), base64.StdEncoding.EncodeToString(piarKey[32:]))
}

func TestNetWork(t *testing.T) {
	piarKey, _ := base64.StdEncoding.DecodeString("+LWEKbfhtyCfaAwuNWIIKwVIuFOqAWohboQsPHvw6nKWlvIMmNDFf\\/tBqKyFSkmdJxGzVcMhHgdPo+B0hv8PPQ==")
	t.Log(base64.StdEncoding.EncodeToString(piarKey[:32]), base64.StdEncoding.EncodeToString(piarKey[32:]))
	//piarKey, _ := base64.StdEncoding.DecodeString("cCUUuTsgaQL3smMWPdoRYbFT095bBBv0uY+5AOo2v0Oc4nPlWa6FXbpyjI8V7XodyKgoVFnpLG3Sez2XVYypNQ==")
	UAHex := "08e8c982cf840d18012a73080012090802101418cd0120101a0330303022033030302a03362e30320556464f4e453a044d6f6f6e42224a313036475f56666f6e655f4d6f6f6e5f4231355f563030315f32303139313031384a2434376265346566382d623066622d346438342d396463642d6164393739613265613731663a0547657474794d21f4c91c60006801"
	UAData, err := hex.DecodeString(UAHex)

	if err != nil {
		t.Fatal(err)
	}
	uaProto := &waproto.ClientPayload{}
	err = proto.Unmarshal(UAData, uaProto)
	if err != nil {
		t.Fail()
	}

	/*uaProto.Username = proto.Uint64(447417595455)
	uaProto.UserAgent.PhoneId = proto.String("")
	uaProto.UserAgent.AppVersion.Secondary = proto.Uint32(21)
	uaProto.UserAgent.AppVersion.Tertiary = proto.Uint32(5)
	uaProto.UserAgent.AppVersion.Quaternary = proto.Uint32(13)*/
	uajson, _ := json.MarshalIndent(uaProto, " ", " ")
	t.Log(string(uajson))

	UAData, err = proto.Marshal(uaProto)
	if err != nil {
		return
	}

	routingInfo := []byte{0x08, 0x08, 0x08, 0x02}
	noiseClient := network.NewNoiseClient(routingInfo, UAData, noise.DHKey{
		Private: piarKey[:32],
		Public:  piarKey[32:],
	}, nil)
	noiseClient.SetNetWorkProxy("socks5://127.0.0.1:51837")
	t.Log(noiseClient.Connect())

	sigs := make(chan os.Signal, 1)
	//signal.Notify 注册这个给定的通道用于接收特定信号。
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
