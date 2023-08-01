package test

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestPrxy(t *testing.T) {
	v := "socks5://209.127.191.180:9279"
	s := strings.Split(v, ":")
	fmt.Println(len(s))
	if len(s) <= 3 {
		ip := strings.ReplaceAll(s[1], "//", "")
		port := s[2]
		fmt.Println(ip, port)
	} else {
		username := strings.ReplaceAll(s[1], "//", "")
		twoArray := strings.Split(s[2], "@")
		pwd := twoArray[0]
		ip := twoArray[1]
		port := s[3]
		fmt.Println(username, pwd, ip, port)
	}
}

func TestKey(t *testing.T) {
	v := "+ImPi/5A+XWiGrAXwiRg+u+pwavHGfYix9Kcb9Zj92+XCxfq7RMAE/3u0hRKiEvVrnIo1o2a+FjK9E8uuFrEAA=="
	_, err := base64.StdEncoding.DecodeString(v)
	if err != nil {

	}
}
func TestPutInt(t *testing.T) {
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, uint32(12207577))
	t.Log(hex.EncodeToString(a))
}

func TestGenId(t *testing.T) {

	last := 8320015 - 1
	maxCount := 812
	minCount := 0
	for true {
		maxCount -= minCount
		if maxCount > 0 {
			if maxCount > 50 {
				minCount = 50
			} else {
				minCount = maxCount
			}
			for i := 0; i < minCount; i++ {
				log.Println(last + i + 1)
			}
			log.Println("---------------")
			last += minCount + 1
		} else {
			return
		}
	}
}
