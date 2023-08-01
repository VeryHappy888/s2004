package node

import (
	"strconv"
	"testing"
	"ws-go/protocol/newxxmp"
	"ws-go/waver"
)

func TestMainNodeProcessor_SendGetIqUserKeys(t *testing.T) {
	newxxmp.SetWAXXMPVersion(waver.NewWA41())

	t.Log(strconv.FormatInt(int64(2), 16))

	//nodeProcessor := NewMainNodeProcessor()
	//nodeProcessor.SendGetIqUserKeys([]string{"111111111111", "222222222"})

	//t.Log(nodeProcessor.iq.GetResult(int(iqUserKeys.GetIqId())))
	//time.Sleep(time.Second * 10000)
}
