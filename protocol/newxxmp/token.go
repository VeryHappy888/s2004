package newxxmp

import (
	"bytes"
	"errors"
	"strconv"
	"ws-go/protocol/iface/ixxmp"
	"ws-go/protocol/utils"
)

type xToken struct {
	token byte
}

func (t *xToken) GetTokenByte() byte {
	return t.token
}
func (t *xToken) GetTokenString() string {
	return dictionary[t.token]
}
func (t *xToken) GetTokenBytes() []byte {
	return []byte{t.token}
}

func (t *xToken) NewFrom(b []byte) ixxmp.IToken {
	var err error
	if len(b) == 0 {
		return nil
	}

	//wslog.GetLogger().Debug("token new from ", hex.EncodeToString(b))
	token := b[0]
	if token == 0x02 {
		b, err = utils.ZipDecompression(b[1:])
		if err != nil {
			return nil
		}

	} else if token == 0x0 {
		b = b[1:]
	}
	//fmt.Println("=-=---->" + hex.EncodeToString(b))
	return t.From(bytes.NewBuffer(b))
}

func (t *xToken) From(buffer *bytes.Buffer) ixxmp.IToken {
	if buffer == nil || buffer.Len() == 0 {
		return nil
	}
	token := readToken(buffer)
	if token == 236 || token == 237 || token == 238 || token == 239 {
		t := readToken(buffer)
		newSecondaryToken := NewSecondaryToken(token, t)
		//wslog.GetLogger().Debug("newSecondaryToken", token, t, newSecondaryToken.GetTokenString())
		return newSecondaryToken
	} else if token == 247 {
		u := t.From(buffer)
		s := t.From(buffer)
		return JabberId(u, s)
	} else if token == 248 {
		newShortArray := NewShortArray(token)
		newShortArray.length = int(readToken(buffer))
		for i := 0; i < newShortArray.length; i++ {
			// 递归
			newShortArray.AddItem(t.From(buffer))
		}
		return newShortArray
	} else if token == 249 {
		newLongArray := NewLongArray(token)
		newLongArray.length = (int(readToken(buffer)) << 8) + int(readToken(buffer))
		for i := 0; i < newLongArray.length; i++ {
			// 递归
			newLongArray.AddItem(t.From(buffer))
		}
		return newLongArray
	} else if token == 250 {
		u := t.From(buffer)
		s := t.From(buffer)
		return JabberId(u, s)
	} else if token == 251 {
		i := unpackBytes(int(token), buffer)
		//wslog.GetLogger().Debug("unpackBytes :", token, hex.EncodeToString(i))
		return PackedHex(i)
	} else if token == 252 {
		length := readToken(buffer)
		data := readBytes(buffer, int(length))
		//log.Println("Int8LengthArray", hex.EncodeToString(data))
		return Int8LengthArray(data)
	} else if token == 253 {
		b := readToken(buffer)
		b1 := readToken(buffer)
		b2 := readToken(buffer)
		length := ((int(b) & 0xF) << 16) + (int(b1) & 0xFF << 8) + int(b2)&0xFF
		data := readBytes(buffer, length)
		return Int24LengthArray(data)
	} else if token == 254 {
		length := (int(readToken(buffer)) << 24) | (int(readToken(buffer)) << 16) | int(readToken(buffer))<<8 | int(readToken(buffer))
		data := readBytes(buffer, length)
		return Int32LengthArray(data)
	} else if token == 255 {
		i := unpackBytes(int(token), buffer)
		//wslog.GetLogger().Debug("unpackBytes :", token, hex.EncodeToString(i))
		return PackedNibble(i)
	} else {
		newToken := NewToken(token)
		//log.Println("newToken ", token, newToken.GetTokenString())
		return newToken
	}

}
func readToken(buffer *bytes.Buffer) byte {
	b, err := buffer.ReadByte()
	if err != nil {
		panic(err)
	}
	return b
}
func readInt8(buffer *bytes.Buffer) int {
	data := make([]byte, 1)
	buffer.Write(data)
	return byteToInt(data)
}

func byteToInt(b []byte) int {
	mask := 0xff
	temp := 0
	n := 0
	for i := 0; i < len(b); i++ {
		n <<= 8
		temp = int(b[i]) & mask
		n |= temp
	}
	return n
}

func readBytes(buffer *bytes.Buffer, len int) []byte {
	data := make([]byte, len)
	_, err := buffer.Read(data)
	if err != nil {
		panic(err)
	}
	return data
}

// NewToken
func NewToken(t byte) *xToken {
	return &xToken{token: t}
}

// secondaryToken
type secondaryToken struct {
	*xToken
	secondaryToken byte
}

func SecondaryToken(i int) *secondaryToken {
	return &secondaryToken{
		xToken:         NewToken(byte(236 + i/256)),
		secondaryToken: byte(i % 256),
	}
}

func NewSecondaryToken(i, sToken byte) *secondaryToken {
	return &secondaryToken{
		xToken:         NewToken(i),
		secondaryToken: sToken,
	}
}

func (s secondaryToken) GetTokenBytes() []byte {
	return []byte{s.token, s.secondaryToken}
}

func (s *secondaryToken) GetTokenString() string {
	n := s.token - 236&0xFF
	n2 := s.secondaryToken & 0xFF
	return secondaryDictionary[int(n2)+int(n)*256]
}

type joinToken struct {
	*xToken
	items []ixxmp.IToken
}

func (j *joinToken) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	for _, item := range j.items {
		buffer.Write(item.GetTokenBytes())
	}
	return buffer.Bytes()
}

// JoinToken
func JoinToken(token ...ixxmp.IToken) *joinToken {
	return &joinToken{items: token}
}

// int8LengthArray
type int8LengthArray struct {
	*xToken
	length int
	data   []byte
}

func (i *int8LengthArray) GetTokenByte() byte {
	return i.token
}
func (i *int8LengthArray) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(i.token)
	buffer.WriteByte(byte(i.length & 255))
	buffer.Write(i.data)
	return buffer.Bytes()
}
func (i *int8LengthArray) GetTokenString() string {
	return string(i.data)
}

// Int8LengthArray
func Int8LengthArray(d []byte) *int8LengthArray {
	return &int8LengthArray{
		xToken: NewToken(252),
		length: len(d),
		data:   d,
	}
}

// Int8LengthArray
func Int8LengthArray2(d []byte, l int) *int8LengthArray {
	return &int8LengthArray{
		xToken: NewToken(252),
		length: l,
		data:   d,
	}
}

// int32LengthArray
type int24LengthArray struct {
	*xToken
	length int
	data   []byte
}

func (i *int24LengthArray) GetTokenByte() byte {
	return i.token
}
func (i *int24LengthArray) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(i.token)
	buffer.WriteByte(byte((983040 & i.length) >> 16))
	buffer.WriteByte(byte((i.length & 65280) >> 8))
	buffer.WriteByte(byte(i.length & 255))
	buffer.Write(i.data)
	return buffer.Bytes()
}

// Int24LengthArray
func Int24LengthArray(d []byte) *int24LengthArray {
	return &int24LengthArray{
		xToken: NewToken(253),
		length: len(d),
		data:   d,
	}
}

// int32LengthArray
type int32LengthArray struct {
	*xToken
	length int
	data   []byte
}

func (i *int32LengthArray) GetTokenByte() byte {
	return i.token
}
func (i *int32LengthArray) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(i.token)
	buffer.WriteByte(byte((2130706432 & i.length) >> 24))
	buffer.WriteByte(byte((16711680 & i.length) >> 16))
	buffer.WriteByte(byte((i.length & 65280) >> 8))
	buffer.WriteByte(byte(i.length & 255))
	buffer.Write(i.data)
	return buffer.Bytes()
}

// Int32LengthArray
func Int32LengthArray(d []byte) *int32LengthArray {
	return &int32LengthArray{
		xToken: NewToken(254),
		length: len(d),
		data:   d,
	}
}

// jabberIf
type jabberId struct {
	*xToken
	user, server ixxmp.IToken
}

func (j *jabberId) GetTokenByte() byte {
	return j.token
}
func (j *jabberId) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(j.token)
	buffer.Write(j.user.GetTokenBytes())
	buffer.Write(j.server.GetTokenBytes())
	return buffer.Bytes()
}
func (j *jabberId) GetTokenString() string {
	return j.user.GetTokenString() + "@" + j.server.GetTokenString()
}

func JabberId(u, s ixxmp.IToken) *jabberId {
	return &jabberId{
		xToken: NewToken(250),
		user:   u,
		server: s,
	}
}

type multiJabberId struct {
	*xToken
	user, server ixxmp.IToken
	flag1        int
	flag2        int
}

func (j *multiJabberId) GetTokenByte() byte {
	return j.token
}
func (j *multiJabberId) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(j.token)
	buffer.Write(j.user.GetTokenBytes())
	buffer.Write(j.server.GetTokenBytes())
	return buffer.Bytes()
}

// packedNibble
type packedNibble struct {
	*xToken
	length int
	data   []byte
}

func (p *packedNibble) GetTokenByte() byte {
	return p.token
}
func (p *packedNibble) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(p.token)
	newData := packBytes(int(p.token), p.data)
	buffer.WriteByte(byte(((p.length & 1) << 7) | len(newData)))
	buffer.Write(newData)
	return buffer.Bytes()
}
func (p *packedNibble) GetTokenString() string {
	return string(p.data)
}

// PackedNibble
func PackedNibble(d []byte) *packedNibble {
	return &packedNibble{
		xToken: NewToken(255),
		length: len(d),
		data:   d,
	}
}

type packedHex struct {
	*xToken
	length int
	data   []byte
}

func (p *packedHex) GetTokenBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteByte(p.token)
	newData := packBytes(int(p.token), p.data)
	buffer.WriteByte(byte(((p.length & 1) << 7) | len(newData)))
	buffer.Write(newData)
	return buffer.Bytes()
}
func (p *packedHex) GetTokenString() string {
	return string(p.data)
}
func PackedHex(d []byte) *packedHex {
	return &packedHex{
		xToken: NewToken(251),
		length: len(d),
		data:   d,
	}
}

func packBytes(i int, bArr []byte) []byte {
	//wslog.GetLogger().Debug("packBytes :", i, hex.Dump(bArr))
	var i3, i2 int

	length := len(bArr)
	if length >= 128 {
		return nil
	}
	i4 := (length + 1) >> 1
	bArr2 := make([]byte, i4)
	for i5 := 0; i5 < length; i5++ {
		b := bArr[i5]
		if i == 251 {
			/*if b >= 48 && b <= 57 {
				break
			}*/
			if b >= 48 && b <= 57 {
				i3 = 0
			} else if b >= 65 && b <= 70 {
				i3 = int(b - 65)
			} else {
				i2 = -1
			}
			i2 = i3 + 10
		} else if i == 255 {
			if b >= 45 && b <= 46 {
				i3 = int(b - 45)
				i2 = i3 + 10
			} else if b >= 48 && b <= 57 {
				i2 = int(b - 48)
			} else {
				i2 = -1
			}

			if i2 == -1 {
				return nil
			}
			i6 := i5 >> 1
			bArr2[i6] = byte(i2<<((1-(i5%2))<<2)) | bArr2[i6]
		}
		i2 = -1
	}
	if length%2 == 1 {
		i7 := i4 - 1
		bArr2[i7] = bArr2[i7] | 15
	}

	//fmt.Println("packBytes :", i, hex.EncodeToString(bArr2))

	return bArr2
}

// unpackBytes
func unpackBytes(i int, buffer *bytes.Buffer) []byte {
	var i3, i2 int

	token := readToken(buffer)
	if (token & 128) != 0 {
		i3 = 1
	}
	i4 := token & 127
	bArr := make([]byte, i4)
	_, err := buffer.Read(bArr)
	if err != nil {
		panic(err)
	}
	i5 := (int(i4) << 1) - i3
	bArr2 := make([]byte, i5)
	for i6 := 0; i6 < i5; i6++ {
		i7 := (1 - (i6 % 2)) << 2
		i8 := int((bArr[i6>>1] & (15 << i7)) >> i7)
		if i == 251 {
			if i8 >= 0 && i8 <= 9 {
				i2 = i8 + 48
			} else if i8 >= 10 && i8 <= 15 {
				i2 = (i8 - 10) + 65
			} else {
				panic(errors.New("bad hex " + strconv.Itoa(i8)))
			}
		} else if i == 255 {
			if i8 >= 0 && i8 <= 9 {
				i2 = i8 + 48
			} else if i8 == 10 || i8 == 11 {
				i2 = (i8 - 10) + 45
			} else {
				panic(errors.New("bad nibble " + strconv.Itoa(i8)))

			}
		} else {
			panic(errors.New("bad packed type " + strconv.Itoa(i)))
		}
		bArr2[i6] = byte(i2)
	}
	return bArr2
}
