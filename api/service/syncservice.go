package service

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"ws-go/api/dto"
	"ws-go/api/vo"
	"ws-go/protocol/entity"
)

// SyncContactService 同步联系人
func SyncContactService(k string, dto dto.SyncContactDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	// check dto
	if len(dto.Numbers) <= 0 {
		return vo.AnErrorOccurred(errors.New("No number to sync "))
	}
	// sync contacts
	sync := app.SendSyncContacts(dto.Numbers)
	result, err := sync.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// iq result
	contacts := make([]gin.H, 0)
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	for _, info := range iqResult.GetUSyncContacts() {
		log.Println(info.Jid(), info.Status(), info.Contact())
		contacts = append(contacts, gin.H{
			"jid":     info.Jid(),
			"status":  info.Status(),
			"contact": info.Contact(),
			"type":    info.ContactType(),
		})
		//查头像
		if info.ContactType() == "in" {
			app.GetPreviewPicture(info.Jid())
		}
	}
	return vo.Resp{Code: 0, Data: gin.H{"status": 200,
		"msg": "ok", "contacts": contacts}, Platform: app.GetPlatform()}
}

// SyncAddOneContactsService  添加单个联系人
func SyncAddOneContactsService(k string, number string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	list := []string{number}
	// sync contacts
	sync := app.SyncAddOneContacts(list)
	result, err := sync.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// iq result
	contacts := make([]gin.H, 0)
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	for _, info := range iqResult.GetUSyncContacts() {
		log.Println(info.Jid(), info.Status(), info.Contact())
		contacts = append(contacts, gin.H{
			"jid":     info.Jid(),
			"status":  info.Status(),
			"contact": info.Contact(),
			"type":    info.ContactType(),
		})
	}
	return vo.Resp{Code: 0, Data: gin.H{"status": 200,
		"msg": "ok", "contacts": contacts}, Platform: app.GetPlatform()}
}
