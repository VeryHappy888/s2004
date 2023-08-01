package newxxmp

import (
	"errors"
	"fmt"
	"ws-go/waver"
)

var (
	dictionary          []string
	secondaryDictionary []string
	//errors
	EncoderErrNilPointer = errors.New("EncoderFail Node in null pointer exception")
	DecoderErrException  = errors.New("Decoder Unknown error occurred")
)

// SetWAXXMPVersion 设置 WA XXMP 版本
func SetWAXXMPVersion(wav waver.WAVInterface) {
	if wav == nil {
		return
	}
	dictionary = wav.GetWADictionary()
	secondaryDictionary = wav.GetSecondaryDictionary()
}

//XMMPDecodeNode
func XMMPDecodeNode(d []byte) (*Node, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:iToken 1", err)
		}
	}()
	if len(d) <= 0 {
		return nil, errors.New("data len <= 0")
	}
	token := NewToken(0)
	iToken := token.NewFrom(d)
	if iToken != nil {
		node := &Node{}
		return node.From(iToken), nil
	}
	return nil, errors.New("iToken is nil")
}
