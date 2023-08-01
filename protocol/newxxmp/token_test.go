package newxxmp

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestPackBytes(t *testing.T) {
	//d, _ := hex.DecodeString("6283827948009")
	t.Log(string([]byte("876283827948009")))
	t.Log(hex.EncodeToString(packBytes(255, []byte("6283827948009"))))

	time.Sleep(time.Second * 1000)
}
