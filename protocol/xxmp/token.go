package xxmp

import (
	"bytes"
	"strconv"
	"strings"
)

type tokenInterface interface {
	getToken() byte
	getBytes() []byte
	getString() string
}

func NewXMMPToken(i byte) *XMMPToken {
	return &XMMPToken{token: i}
}

type XMMPToken struct {
	token byte
}

func (x *XMMPToken) From(buffer *bytes.Buffer) (tokenInterface, error) {
	b, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}

	if b == 0xEC || b == 0xED || b == 0xEE || b == 0xEF {
		_SecondaryToken, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		s := &SecondaryToken{token: b, secondaryToken: _SecondaryToken}
		//fmt.Println(b, _SecondaryToken, s.getString())
		return s, nil
	} else if b == 0xF8 {
		t := NewShortList(b)
		bb, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		t.length = bb
		for i := 0; i < int(t.length); i++ {
			token, err := x.From(buffer)
			if err != nil {
				return nil, err
			}
			t.AddItem(token)
		}
		return t, nil
	} else if b == 0xF9 {
		t := NewLongList(b)
		bb, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		t.length = bb
		for i := 0; i < int(t.length); i++ {
			token, err := x.From(buffer)
			if err != nil {
				return nil, err
			}
			t.AddItem(token)
		}
		return t, nil
	} else if b == 0xFA {
		u, err := x.From(buffer)
		if err != nil {
			return nil, err
		}
		s, err := x.From(buffer)
		if err != nil {
			return nil, err
		}
		return &JabberId{
			XMMPToken: NewXMMPToken(b), user: u, server: s,
		}, nil
	} else if b == 0xFB {
		packedHex := NewPackedHex(b)
		startByte, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		packedHex.length = int(startByte & 0x7F)
		buf := make([]byte, packedHex.length)
		_, err = buffer.Read(buf)
		if err != nil {
			return nil, err
		}
		packedHex.data = string(packedHex.unpack2(startByte, buf))
		return packedHex, nil
	} else if b == 0xFC {
		t := Int8LengthArrayString2(b)
		bb, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		t.length = int(bb)
		buf := make([]byte, t.length)
		_, err = buffer.Read(buf)
		if err != nil {
			return nil, err
		}
		t.data = string(buf)
		return t, nil
	} else if b == 0xFD {
		t := NewInt20LengthArrayString2(b)
		b1, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		b2, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		b3, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}

		t.length = ((int(b1) & 0xF) << 16) + (int(b2) << 8) + int(b3)
		buf := make([]byte, t.length)
		_, err = buffer.Read(buf)
		if err != nil {
			return nil, err
		}
		t.data = string(buf)
		return t, nil
	} else if b == 0xFE {
		t := NewInt31LengthArrayString2(b)
		b1, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		b2, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		b3, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		t.length = (int(b1) << 24) | (int(b1) << 16) | int(b2)<<8 | int(b3)
		buf := make([]byte, t.length)
		_, err = buffer.Read(buf)
		if err != nil {
			return nil, err
		}
		t.data = string(buf)
		return t, nil
	} else if b == 0xFF {
		t := NewPackedNibble(b)
		startByte, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		t.length = startByte & 0x7F
		buf := make([]byte, t.length)
		_, err = buffer.Read(buf)
		if err != nil {
			return nil, err
		}
		t.data = t.unpack(startByte, buf)
		return t, nil
	} else {
		return NewXMMPToken(b), nil
	}
	return nil, nil
}

func (x *XMMPToken) getBytes() []byte {
	return []byte{x.token}
}

func (x *XMMPToken) getString() string {
	return dictionary[x.token]
}

func (x *XMMPToken) getToken() byte {
	return x.token
}

func NewSecondaryToken(i int) *SecondaryToken {
	return &SecondaryToken{
		i:              i,
		token:          byte(236 + i/256),
		secondaryToken: byte(i % 256),
	}
}

type SecondaryToken struct {
	i              int // index
	token          byte
	secondaryToken byte
}

func (x *SecondaryToken) getBytes() []byte {
	return []byte{x.token, x.secondaryToken}
}

func (x *SecondaryToken) getString() string {
	return ""
}

func (x *SecondaryToken) getToken() byte {
	return x.token
}

func NewJabberId(user, server tokenInterface) *JabberId {
	return &JabberId{
		XMMPToken: NewXMMPToken(0xFA),
		user:      user,
		server:    server,
	}
}

type JabberId struct {
	*XMMPToken
	user, server tokenInterface
}

func (j *JabberId) getBytes() []byte {
	buffer := bytes.NewBuffer([]byte{})
	buffer.WriteByte(j.token)
	buffer.Write(j.user.getBytes())
	buffer.Write(j.server.getBytes())

	return buffer.Bytes()
}

func (j JabberId) getString() string {
	if j.user == nil && j.server == nil {
		return ""
	}
	return j.user.getString() + "@" + j.server.getString()
}

func (x *JabberId) getToken() byte {
	return x.token
}

func Int8LengthArrayString2(b byte) *Int8LengthArrayString {
	return &Int8LengthArrayString{
		XMMPToken: NewXMMPToken(b),
		length:    0,
		data:      "",
	}
}

func NewInt8LengthArrayString(s string) *Int8LengthArrayString {
	return &Int8LengthArrayString{
		XMMPToken: NewXMMPToken(0xFC),
		length:    len(s),
		data:      s,
	}
}

type Int8LengthArrayString struct {
	*XMMPToken
	length int
	data   string
}

func (i *Int8LengthArrayString) getBytes() []byte {
	buffer := bytes.NewBuffer([]byte{i.token, byte(len(i.data))})
	buffer.Write([]byte(i.data))
	return buffer.Bytes()
}
func (i *Int8LengthArrayString) getString() string {
	return i.data
}
func (x *Int8LengthArrayString) getToken() byte {
	return x.token
}

func NewInt20LengthArrayString2(b byte) *Int8LengthArrayString {
	return &Int8LengthArrayString{
		XMMPToken: NewXMMPToken(b),
		length:    0,
		data:      "",
	}
}

type Int20LengthArrayString struct {
	*XMMPToken
	length byte
	data   string
}

func (i Int20LengthArrayString) getBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(i.token)
	buffer.WriteByte(byte(len(i.data) >> 16 & 0x0F))
	buffer.WriteByte(byte(len(i.data) >> 8 & 0xFF))
	buffer.WriteByte(byte(len(i.data) >> 0 & 0xFF))
	buffer.Write([]byte(i.data))
	return buffer.Bytes()
}
func (x *Int20LengthArrayString) getToken() byte {
	return x.token
}

func NewInt31LengthArrayString2(b byte) *Int31LengthArrayString {
	return &Int31LengthArrayString{
		XMMPToken: NewXMMPToken(b),
		length:    0,
		data:      "",
	}
}

func NewInt31LengthArrayString(s string) *Int31LengthArrayString {
	return &Int31LengthArrayString{
		XMMPToken: NewXMMPToken(0xFE),
		length:    len(s),
		data:      s,
	}
}

type Int31LengthArrayString struct {
	*XMMPToken
	length int
	data   string
}

func (i *Int31LengthArrayString) getBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(i.token)
	buffer.WriteByte(byte(len(i.data) >> 24 & 0x7F))
	buffer.WriteByte(byte(len(i.data) >> 16 & 0xFF))
	buffer.WriteByte(byte(len(i.data) >> 8 & 0xFF))
	buffer.WriteByte(byte(len(i.data) >> 0 & 0xFF))
	buffer.Write([]byte(i.data))
	return buffer.Bytes()
}
func (x *Int31LengthArrayString) getToken() byte {
	return x.token
}

func NewPackedHex(b byte) *PackedHex {
	return &PackedHex{
		XMMPToken: NewXMMPToken(b),
	}
}

type PackedHex struct {
	*XMMPToken
	length int
	data   string
}

func (p *PackedHex) getBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(p.token)
	buffer.WriteByte(0x00)

	return buffer.Bytes()
}
func (p *PackedHex) unpack2(startByte byte, packed []byte) []byte {
	var i2, i3 int
	if (startByte & 0x80) != 1 {
		i3 = 1
	}
	i5 := (p.length << 1) - i3
	newPacked := make([]byte, i5)
	for i := 0; i < i5; i++ {
		i7 := (1 - (i % 2)) << 2
		i8 := (int(packed[i>>1]) & (0xF << i7)) >> i7
		if i8 >= 0 && i8 <= 9 {
			i2 = i8 + 48
		} else if i8 >= 10 && i8 <= 15 {
			i2 = (i8 - 10) + 65
		}
		newPacked[i] = byte(i2)
	}
	return newPacked
}

func (p *PackedHex) unpack(startByte byte, packed []byte) string {

	builder := strings.Builder{}
	if (startByte & 0x7F) == 0 {
		return ""
	}
	for i := 0; i < int(startByte&0x7F); i++ {
		currByte := packed[i]
		builder.WriteString(p.unpackByte((int(currByte) & 0xF0) >> 4))
		builder.WriteString(p.unpackByte(int(currByte) & 0x0F))
	}
	ret := []byte{}
	if (startByte >> 7) == 0 {
		ret = []byte(builder.String())
		ret = ret[:len(ret)-1]
	}
	return string(ret)
}

func (p *PackedHex) unpackByte(v int) string {
	if v >= 0 && v <= 9 {
		return string(rune('0' + v))
	} else if v == 10 {
		return "-"
	} else if v == 11 {
		return "."
	} else if v == 15 {
		return "\\0"
	}
	panic("invalid nibble to unpack: " + strconv.Itoa(v))
}
func (p *PackedHex) getToken() byte {
	return p.token
}
func (p *PackedHex) getString() string {
	return p.data
}

func NewPackedNibble(b byte) *PackedNibble {
	return &PackedNibble{
		XMMPToken: NewXMMPToken(b),
		length:    0,
		data:      "",
	}
}

type PackedNibble struct {
	*XMMPToken
	length byte
	data   string
}

func (p *PackedNibble) getBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(p.token)
	buffer.WriteByte(0x00)

	return buffer.Bytes()
}

func (p *PackedNibble) unpack(startByte byte, packed []byte) string {
	builder := strings.Builder{}
	if (startByte & 0x7F) == 0 {
		return ""
	}
	for i := 0; i < int(startByte&0x7F); i++ {
		currByte := packed[i]
		builder.WriteString(p.unpackByte((int(currByte) & 0xF0) >> 4))
		builder.WriteString(p.unpackByte(int(currByte) & 0x0F))
	}
	var ret []byte
	if (startByte >> 7) == 0 {
		ret = []byte(builder.String())
		ret = ret[:len(ret)-1]
	}
	return string(ret)
}

func (p *PackedNibble) unpackByte(v int) string {
	if v < 0 || v > 15 {
		panic("invalid hex to unpack: " + strconv.Itoa(v))
	}
	if v < 10 {
		return string(rune('0' + v))
	} else {
		return string(rune('0' + v - 10))
	}
}
func (p *PackedNibble) getToken() byte {
	return p.token
}
func (p *PackedNibble) getString() string {
	return p.data
}
