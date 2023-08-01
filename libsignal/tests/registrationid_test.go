package tests

import (
	"fmt"
	"testing"
	"ws-go/libsignal/util/keyhelper"
)

func TestRegistrationID(t *testing.T) {
	i := 0
	fmt.Println(44484893)
	for {
		regID := keyhelper.GenerateRegistrationID()
		fmt.Println(regID)
		i++
		if i == 100 {
			break
		}
	}

}
