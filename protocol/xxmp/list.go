package xxmp

import (
	"bytes"
)

type AbstractListInterface interface {
	GetBytes() []byte
	AddItems(t AbstractListInterface)
	AddItem(token tokenInterface)
	GetItems() []tokenInterface
}

type BaseList struct {
	*XMMPToken
	Items []tokenInterface
}

// ShortList 默认不带参数
func ShortList() *shortList {
	return &shortList{
		BaseList: &BaseList{XMMPToken: NewXMMPToken(0xf8)},
	}
}

// NewShortList 带参
func NewShortList(t byte) *shortList {
	return &shortList{
		BaseList: &BaseList{
			XMMPToken: NewXMMPToken(t),
		},
	}
}

// LongList
func LongList() *longList {
	return &longList{BaseList: &BaseList{
		XMMPToken: NewXMMPToken(0xF9),
	}}
}

// NewLongList
func NewLongList(t byte) *longList {
	return &longList{BaseList: &BaseList{
		XMMPToken: NewXMMPToken(t),
	}}
}

// shorList 248
type shortList struct {
	*BaseList
	length byte
}

func (s *shortList) GetItems() []tokenInterface {
	return s.Items
}

func (s *shortList) AddItems(t AbstractListInterface) {
	s.Items = append(s.Items, t.GetItems()...)
}

func (s *shortList) AddItem(token tokenInterface) {
	s.Items = append(s.Items, token)
}

// getBytes TokenList to bytes
func (s *shortList) GetBytes() []byte {
	newBuffer := bytes.NewBuffer([]byte{})

	newBuffer.Write(s.XMMPToken.getBytes())
	newBuffer.WriteByte(byte(len(s.Items)))

	for _, item := range s.Items {
		if listInterface, ok := item.(AbstractListInterface); ok {
			//log.Println(hex.EncodeToString(listInterface.GetBytes()))
			newBuffer.Write(listInterface.GetBytes())
		} else {
			newBuffer.Write(item.getBytes())
		}

	}

	return newBuffer.Bytes()
}

// longList 256
type longList struct {
	*BaseList
	length byte
}

func (l *longList) GetItems() []tokenInterface {
	return l.Items
}

func (l *longList) AddItems(t AbstractListInterface) {
	l.Items = append(l.Items, t.GetItems()...)
}

func (l *longList) AddItem(token tokenInterface) {
	l.Items = append(l.Items, token)
}

func (l *longList) GetBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.Write(l.BaseList.getBytes())
	buffer.WriteByte(byte((len(l.Items) >> 8) & 0xFF))
	buffer.WriteByte(byte((len(l.Items) >> 0) & 0xFF))
	for _, item := range l.Items {
		buffer.Write(item.getBytes())
	}
	return buffer.Bytes()
}
