package waproto

import (
	"github.com/golang/protobuf/proto"
)

type WAMessageType int

const (
	_ WAMessageType = iota

	WAMessageTypeText  // text message
	WAMessageTypeSkMsg // SKMSG message
)

// CreateWAMessageText
func CreatePBWAMessageText(context string) ([]byte, error) {
	var w WAMessage
	w.CONVERSATION = proto.String(context)
	return proto.Marshal(&w)
}

func CreatePBWAMessageNewText(context string) ([]byte, error) {
	var w Message
	w.Conversation = proto.String(context)
	w.ExtendedTextMessage = &ExtendedTextMessage{
		ContextInfo: &ContextInfo{
			MentionedJid: []string{"6283826990500@s.whatsapp.net"},
		},
		Text: proto.String("@6283826990500"),
	}
	return proto.Marshal(&w)
}

// CreatePBWAMessageSkMsg
func CreatePBWAMessageSkMsg(id string, skMsg []byte) ([]byte, error) {
	var w WAMessage
	w.SKMSG = &SenderKeyGroupMessage{
		GROUP_ID:   proto.String(id),
		SENDER_KEY: skMsg,
	}
	return proto.Marshal(&w)
}
