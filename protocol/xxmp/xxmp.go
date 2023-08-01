package xxmp

import (
	"bytes"
	"errors"
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

// Encoder
func Encoder(node *Node) ([]byte, error) {
	if node == nil {
		return []byte{}, EncoderErrNilPointer
	}
	return node.GetToken().GetBytes(), nil
}

// Decoder
func Decoder(d []byte) (*Node, error) {
	t, err := new(XMMPToken).From(bytes.NewBuffer(d))
	if err != nil {
		return nil, err
	}
	if listInterface, ok := t.(AbstractListInterface); ok {
		n := &Node{}
		node := n.From(listInterface)
		return node, nil
	}
	return nil, DecoderErrException
}
