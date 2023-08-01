package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-go/api/vo"
	"ws-go/protocol/db"
	"ws-go/protocol/entity"
	"ws-go/protocol/node"
)

// ScanNumberService
func ScanNumberService(k string, number string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	//添加至redis
	key := fmt.Sprintf("whatsapp:scanNumber:%s", number)
	exists, err := db.Exists(key)
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	resultPresence := entity.PresenceResult{}
	if exists {
		err := db.GETObj(key, &resultPresence)
		if err != nil {
			fmt.Println(err.Error())
			return vo.AnErrorOccurred(err)
		}
		//fmt.Println("cache get is ok")
	} else {
		// sync contacts
		sync := app.SyncAddScanContacts([]string{number})
		result, err := sync.GetResult()
		if err != nil {
			return vo.AnErrorOccurred(err)
		}
		iqResult := result.(entity.IqResult)
		if iqResult.GetErrorEntityResult() != nil {
			return vo.Success(gin.H{
				"status": iqResult.GetErrorEntityResult().Code(),
				"msg":    iqResult.GetErrorEntityResult().Text(),
				"data":   iqResult,
			}, app.GetPlatform(), "no")
		}
		subscribe := app.SendPresencesSubscribe(node.NewJid(number).Jid())
		any, err := subscribe.GetResult()
		if err != nil {
			return vo.AnErrorOccurred(err)
		}
		resultPresence = any.(entity.PresenceResult)
		_ = db.SETExpirationObj(key, resultPresence, 60*60*12*1)
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"data":   resultPresence,
	}, app.GetPlatform(), "ok")
}

// ExistenceService
func ExistenceService(k string, number []string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	_, _ = app.SendNumberExistence(number)
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "ok")
}
