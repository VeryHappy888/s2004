package test

import (
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
)

func TestID(t *testing.T) {
	b := make([]byte, 12)
	rand.Read(b)
	var ss []string
	for _, v := range b {
		ss = append(ss, fmt.Sprintf("%%%02X", v))
	}

	fmt.Println(strings.Join(ss, ""))
}
