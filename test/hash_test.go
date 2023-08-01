package test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"unsafe"
)

func TestBase64Desc(t *testing.T) {
	c := "OoUPZXppAziqZSi+RS2XweorzdaWkD9/SzHQijZ+nlIWrlzA7YpExMBzpWrZ38jmLo67qLquHtUh1BVY0SlRVr3mS9wILxxEKe04rj0INgqD3aTqUj6HRXkCdv//XNFn"
	kone, _ := base64.URLEncoding.DecodeString(c)
	fmt.Println(Bytes2String(kone))

}
func Bytes2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func TestDecPhone(t *testing.T) {
}
