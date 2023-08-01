package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-go/api/dto"
	"ws-go/api/vo"
	"ws-go/protocol/app"
	"ws-go/protocol/entity"
	"ws-go/protocol/node"
)

// SetProfilePictureService
func SetProfilePictureService(k string, dto dto.PictureInfoDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}

	if len(dto.Picture) <= 0 {
		return vo.ParameterError("picture", "Accept a Base64 picture")
	}
	// set profile picture
	picture := app.SendProfilePicture(dto.Picture)
	_, err := picture.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	picture = app.GetProfilePicture(k)
	result, err := picture.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	jid := node.NewJid(k)
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	info := iqResult.GetPictureInfo()
	pic := "404"
	if info != nil {
		pic = info.Picture()
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"from":   jid.Jid(),
		"pic":    pic,
	}, app.GetPlatform(), "ok")
}

//SetNickNameService 设置名称
func SetNickNameService(k string, dto dto.NickNameDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	_ = app.SendNickName(dto.Name)
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "ok")
}

func GetProfileService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"info":   app,
	}, app.GetPlatform(), "ok")
}

// 获取个人二维码
func GetQrService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	respQr := app.SendGetQr()
	result, err := respQr.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	qr := iqResult.GetQr()
	if qr == nil || qr.Code() == "" {
		return vo.AnErrorOccurred(err)
	}
	url := fmt.Sprintf("https://wa.me/qr/%s", qr.Code())
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"qrCode": url,
	}, app.GetPlatform(), "ok")
}

// 重置二维码
func SetQrRevokeService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	respQr := app.SetQrRevoke()
	result, err := respQr.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	qr := iqResult.GetQr()
	if qr == nil || qr.Code() == "" {
		return vo.AnErrorOccurred(err)
	}
	url := fmt.Sprintf("https://wa.me/qr/%s", qr.Code())
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"qrCode": url,
	}, app.GetPlatform(), "ok")
}

// ScanCodeService 扫描二维码
func ScanCodeService(k, code string, opCode int32) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	respQr := app.ScanCode(code, opCode)
	result, err := respQr.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)

	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	if opCode == 0 {
		//个人
		qr := iqResult.GetQr()
		if qr == nil || qr.Jid() == "" {
			return vo.AnErrorOccurred(err)
		}
		//发送同步
		list := []string{qr.Jid()}
		go sysContact(app, list)
		return vo.Success(gin.H{
			"status":  200,
			"msg":     "ok",
			"contact": qr.Jid(),
			"type":    qr.TypeNode(),
			"notify":  qr.Notify(),
		}, app.GetPlatform(), "ok")
	} else {
		groupInfo := iqResult.GetGroupInfo()
		if groupInfo == nil {
			return vo.AnErrorOccurred(err)
		}
		participants := make([]gin.H, 0)
		for jid, participant := range groupInfo.Participants() {
			participants = append(participants, gin.H{
				"jid":  jid,
				"attr": participant,
			})
		}
		inviteRsp := app.InviteCode(code, "g.us")
		result, err = inviteRsp.GetResult()
		if err != nil {
			return vo.AnErrorOccurred(err)
		}
		iqResult = result.(entity.IqResult)
		if iqResult.GetErrorEntityResult() != nil {
			return vo.Success(gin.H{
				"status": iqResult.GetErrorEntityResult().Code(),
				"msg":    iqResult.GetErrorEntityResult().Text(),
			}, app.GetPlatform(), "no")
		}
		qr := iqResult.GetGroupInfo()
		if inviteRsp == nil {
			return vo.AnErrorOccurred(err)
		}
		return vo.Success(gin.H{
			"Subject":      groupInfo.Subject(),
			"Id":           groupInfo.Id(),
			"Creation":     groupInfo.Creation(),
			"Creator":      groupInfo.Creator(),
			"so":           groupInfo.SO(),
			"st":           groupInfo.ST(),
			"count":        len(participants),
			"Participants": participants,
			"GroupId":      qr.GroupId(),
			"status":       200,
			"msg":          "ok",
		}, app.GetPlatform(), "ok")
	}
}

//同步
func sysContact(app *app.WaApp, list []string) {
	sync := app.SendSyncContactsAdd(list)
	_, err := sync.GetResult()
	if err == nil {
		app.SendSyncContactsInteractive(list)
	}
}

// GetProfilePictureService
func GetProfilePictureService(k string, dto dto.PictureInfoDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}

	if len(dto.From) <= 0 {
		return vo.ParameterError("from", "Which account avatar to get?")
	}
	// set profile picture
	picture := app.GetProfilePicture(dto.From)
	result, err := picture.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	jid := node.NewJid(dto.From)
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	info := iqResult.GetPictureInfo()
	pic := "404"
	if info != nil {
		pic = info.Picture()
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"from":   jid.Jid(),
		"pic":    pic,
	}, app.GetPlatform(), "")
}

// GetPreviewService
func GetPreviewService(k string, dto dto.PictureInfoDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}

	if len(dto.From) <= 0 {
		return vo.ParameterError("from", "Which account avatar to get?")
	}
	// set profile picture
	picture := app.GetPreviewPicture(dto.From)
	result, err := picture.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	jid := node.NewJid(dto.From)
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	info := iqResult.GetPictureInfo()
	pic := "404"
	if info != nil {
		pic = info.Picture()
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
		"base64": info.Base64(),
		"from":   jid.Jid(),
		"pic":    pic,
	}, app.GetPlatform(), "")
}

// SetState
func SetStateService(k string, dto dto.SetStateDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}

	if len(dto.Content) <= 0 || len(dto.Content) > 118 {
		return vo.ParameterError("from", "Which account avatar to get?")
	}
	stateRsp := app.SendSetState(dto.Content)
	_, err := stateRsp.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(nil, app.GetPlatform(), "设置成功")
}

// GetStateService
func GetStateService(k string, dto dto.GetStateDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	stateRsp := app.SendGetState(dto.ToWid)
	result, err := stateRsp.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	status := iqResult.GetUserStatus()
	return vo.Success(gin.H{
		"status":    200,
		"msg":       "ok",
		"toWid":     status.ToWid(),
		"t":         status.T(),
		"signature": status.Signature(),
	}, app.GetPlatform(), "成功")
}

//TwoVerifyService
func TwoVerifyService(k string, dto dto.TwoVerifyDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	_ = app.TwoVerify(dto.Code, dto.Email)
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "成功")
}
