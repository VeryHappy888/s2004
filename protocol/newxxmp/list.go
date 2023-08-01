package newxxmp

import (
	"bytes"
	"ws-go/protocol/iface/ixxmp"
)

type abstractTokenList struct {
	ixxmp.IToken
	ixxmp.ITokenList
	items []ixxmp.IToken
}

// ShortArray
func ShortArray() *shortArray {
	return &shortArray{
		abstractTokenList: &abstractTokenList{
			IToken: NewToken(0xF8),
		},
		length: 0,
	}
}

// NewShortArray
func NewShortArray(b byte) *shortArray {
	return &shortArray{
		abstractTokenList: &abstractTokenList{
			IToken: NewToken(b),
		},
		length: 0,
	}
}

// ShortArray 248
type shortArray struct {
	*abstractTokenList
	length int
}

func (s *shortArray) GetBytes(headBit ...bool) []byte {
	buffer := bytes.Buffer{}
	if len(headBit) > 0 && headBit[0] {
		if len(headBit) > 1 && headBit[1] {
			buffer.WriteByte(0x02)
		} else {
			buffer.WriteByte(0x00)
		}
	}

	buffer.WriteByte(s.GetTokenByte())
	buffer.WriteByte(byte(len(s.items)))
	for _, item := range s.items {
		if tokenList, ok := item.(ixxmp.ITokenList); ok {
			//log.Println(hex.EncodeToString(tokenList.GetBytes()))
			buffer.Write(tokenList.GetBytes())
		} else {
			//log.Println(hex.EncodeToString(item.GetTokenBytes()))
			buffer.Write(item.GetTokenBytes())
		}
	}
	return buffer.Bytes()
}

func (s *shortArray) AddItem(t ixxmp.IToken) {
	s.items = append(s.items, t)
}
func (s *shortArray) GetItems() []ixxmp.IToken {
	return s.items
}

// NewLongArray
func NewLongArray(b byte) *longArray {
	return &longArray{
		abstractTokenList: &abstractTokenList{
			IToken: NewToken(b),
		},
		length: 0,
	}
}

// LongArray
func LongArray() ixxmp.ITokenList {
	return &longArray{
		abstractTokenList: &abstractTokenList{
			IToken: NewToken(0xF9),
		},
		length: 0,
	}
}

func (s *longArray) GetBytes(headBit ...bool) []byte {
	buffer := bytes.Buffer{}
	if len(headBit) > 0 && headBit[0] {
		if len(headBit) > 1 && headBit[1] {
			buffer.WriteByte(0x02)
		} else {
			buffer.WriteByte(0x00)
		}
	}

	buffer.WriteByte(s.GetTokenByte())
	buffer.WriteByte(byte(len(s.items) >> 8 & 0xFF))
	buffer.WriteByte(byte(len(s.items) >> 0 & 0xFF))
	for _, item := range s.items {
		if tokenList, ok := item.(ixxmp.ITokenList); ok {
			//log.Println(hex.EncodeToString(tokenList.GetBytes()))
			buffer.Write(tokenList.GetBytes())
		} else {
			//log.Println(hex.EncodeToString(item.GetTokenBytes()))
			buffer.Write(item.GetTokenBytes())
		}
	}
	return buffer.Bytes()
}

func (s *longArray) AddItem(t ixxmp.IToken) {
	s.items = append(s.items, t)
}
func (s *longArray) GetItems() []ixxmp.IToken {
	return s.items
}

type longArray struct {
	*abstractTokenList
	length int
}
