package node

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"testing"
	"ws-go/protocol/newxxmp"
	"ws-go/waver"
)

func TestIqNode_Process(t *testing.T) {
	ddd, _ := hex.DecodeString("7ce92301")
	t.Log(binary.BigEndian.Uint32(ddd))

	newxxmp.SetWAXXMPVersion(waver.NewWA41())
	token := newxxmp.NewToken(0)
	d, err := hex.DecodeString("f8081106fa0003051c04fc023063f801f8025cf801f806170cfaff068869269324910309fc0a31363135353331343333f805f8029afc047ed158fef80205fc0105f8029bfc20e5a3b222f86ee11608b7533980e082105142c54ba0a7c18c22fd8e36102f194bf802cff803f80204fc0300000cf80228fc2041fe9862fc29bc89476b63f85e5d57f62c3967e859dfbd096d88232017d76426f802cefc40f6c2f08f280d14d85e5a5796b9e7a3b7ef983704877fc403bfc9de41f0a81e0333e44bfc5e027a396ca8a1d763e80def76c9455fa80e158c14f5cc752e937704f8028df802f80204fc03086a96f80228fc201dc299cea0dd671b153eb8a031e9feec70c4edd0b9f517cf86df6525a468f95c")
	if err != nil {
		t.Fatal(err)
	}
	buffer := bytes.NewBuffer(d)
	iToken := token.From(buffer)
	t.Log(iToken)

	node := &newxxmp.Node{}
	fromNode := node.From(iToken)
	log.Println("\n" + fromNode.GetString())

	iqNode := &IqNode{}
	iqNode.Process(fromNode)
}
