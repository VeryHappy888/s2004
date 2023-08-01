package test

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"testing"
	"ws-go/protocol/crypto/cbc"
)

func TestX(t *testing.T) {
	regIdData := make([]byte, 4)
	binary.BigEndian.PutUint32(regIdData, 371095602)
	fmt.Println(base64.StdEncoding.EncodeToString(regIdData))
}
func TestEncDy(t *testing.T) {
	key := []byte("MySecretSecretSecretSecretKey123")
	//plain := []byte("Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.")
	plain, _ := base64.StdEncoding.DecodeString("MwiQg8fYBBABGpABtFU0sHUGkPcbV094NQmLVZ5JS52Y5h33d6CO5AJS+n5J2JI7HJE7sH/cTfxeF8BpMnRpiGHIRoy31n1Qb5lYHkD9akSI+nqUjFJN2QlLTMjdm/S4BiAgRtFnFfAJ3fB9U6wxJhcf2D1n9s2GjHPYGUlWJC9UEXLQqQj/07SKDAbedajOAoP3w6JUrr/PsJ9tNJFghG/8pclLM1TwBRz3p3K8p+JPJ15Mg9930LDoIrNcUQ9qbwK6ECsKwDjusxSQSeGe5NpFw9XSm0NpP+9vBQ==")
	cipher, err := Encrypt(key, nil, plain)
	if err != nil {
		t.Fail()
	}
	p, err := cbc.Decrypt(key, nil, cipher)
	if err != nil {
		t.Fail()
	}
	//fmt.Println(base64.StdEncoding.EncodeToString(p))
	if !bytes.Equal(plain, p) {
		t.Fail()
	}
}

func TestY(t *testing.T) {
	regIdData := make([]byte, 4)
	binary.BigEndian.PutUint32(regIdData, 1255)
	fmt.Println(regIdData)
}
