package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-go/api/dto"
	"ws-go/api/vo"
)

//SnsTextService 文字动态
func SnsTextService(k string, dto dto.SnsTextDto) vo.Resp {
	// check parameters
	if isEmpty(dto.Text) || len(dto.Text) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	_, error := app.SendSnsText(dto.Text, dto.Participants)
	if error != nil {
		return vo.Success(gin.H{
			"status": 501,
			"data":   error.Error(),
			"msg":    "发送异常!",
		}, app.GetPlatform(), "ok")
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "ok")
}
