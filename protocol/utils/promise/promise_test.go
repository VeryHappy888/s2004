package promise

import (
	"log"
	"testing"
	"time"
)

func TestNewTimeOut(t *testing.T) {
	promise := NewPromiseTimeOut(time.Second * 10)

	go func() {
		t.Log("Perform time-consuming execution")
		time.Sleep(time.Second * 2)
		promise.SuccessResolve("success")
		t.Log("put success")
	}()

	promise.Then(func(data Any) Any {
		log.Println("迭代器 1", data)
		return "啦啦啦啦啦"
	})

	t.Log(promise.Await())
}
