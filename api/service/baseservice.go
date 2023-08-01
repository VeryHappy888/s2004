package service

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/os/gcache"
	"ws-go/protocol/app"
	"ws-go/protocol/entity"
)

var appCache *gcache.Cache

// msgCache
var msgCache *gcache.Cache

func init() {
	appCache = gcache.New()
	msgCache = gcache.New()
}

func RemoveWSApp(k string) {
	appCache.Remove(k)
}

// GetWSApp
func GetWSApp(k string) (*app.WaApp, bool) {
	existence, _ := appCache.Contains(k)
	if existence {
		a, err := appCache.Get(k)
		if err != nil {
			return nil, false
		}
		v := a.(*app.WaApp)
		/*Online // 上线 认证成功后上线(连接成功的)
		Drops  // 掉线 认证成功后掉线 （连接断开）
		AuthFailed      // 认证失败 (连接成功后认证失败)
		Banned          // 被禁止使用
		HandshakeFailed // 握手失败
		Connect         // 连接成功
		Disconnect      // 断开连接*/
		fmt.Println("账号:[", v.GetUserName(), "状态=", v.GetLoginStatus().String(), "]")
		if v.GetLoginStatus() == app.Banned {
			RemoveWSApp(k)
			return nil, false
		}
		if v.GetLoginStatus() == app.Drops {
			RemoveWSApp(k)
			return nil, false
		}
		if v.GetLoginStatus() == app.Disconnect {
			RemoveWSApp(k)
			return nil, false
		}
		if v.GetLoginStatus() != app.Online {
			RemoveWSApp(k)
			return nil, false
		}
		return v, true
	}
	return nil, false
}

// CreateWSApp
func CreateWSApp(info *app.AccountInfo) (*app.WaApp, error) {
	if info == nil {
		return nil, errors.New("Need the basic information of the account")
	}
	// new instance
	waAppCli := app.NewWaAppCli(info)
	// save instance
	_ = appCache.Set(info.GetUserName(), waAppCli, 0)
	return waAppCli, nil
}

// GetNewMessages
func GetNewMessages(k string) []*entity.ChatMessage {
	chatMgsList := make([]*entity.ChatMessage, 0)
	if existence, _ := msgCache.Contains(k); !existence {
		return chatMgsList
	} else {
		v, err := msgCache.Remove(k)
		if err != nil {
			return chatMgsList
		}

		var ok bool
		if chatMgsList, ok = v.([]*entity.ChatMessage); ok {
			return chatMgsList
		}
	}
	return nil
}

// SaveNewMessage 保存新消息
func SaveNewMessage(k string, message *entity.ChatMessage) error {
	/*chatMgsList := make([]*entity.ChatMessage, 0)
	if existence, _ := msgCache.Contains(k); !existence {
		chatMgsList = append(chatMgsList, message)
		_ = msgCache.Set(k, chatMgsList, time.Hour)
	} else {
		v, err := msgCache.Get(k)
		if err != nil {
			return err
		}

		var ok bool
		if chatMgsList, ok = v.([]*entity.ChatMessage); ok {
			chatMgsList = append(chatMgsList, message)
			_, _, _ = msgCache.Update(k, chatMgsList)
		}
	}*/
	return nil
}

// isEmpty
func isEmpty(s string) bool {
	return s == "" || len(s) <= 0
}
