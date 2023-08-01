package service

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-go/api/dto"
	"ws-go/api/vo"
	app2 "ws-go/protocol/app"
	"ws-go/protocol/entity"
	_interface "ws-go/protocol/iface"
	"ws-go/wslog"
)

// LoginService 登录
func LoginService(dto dto.LoginDto) vo.Resp {
	if dto.StaticPriKey == "" || dto.StaticPubKey == "" {
		return vo.IncompleteParameters()
	}
	// create empty account instance
	emptyAccountInfo := app2.EmptyAccountInfo()
	if dto.AuthBody == nil && isEmpty(dto.AuthHexData) {
		return vo.IncompleteParameters()
	} else if dto.AuthBody != nil {
		if resp := GenAuthDataService(*dto.AuthBody); resp.Code != 0 {
			return resp
		}
		emptyAccountInfo.SetCliPayload(dto.AuthBody.ClientPayload)
	} else if !isEmpty(dto.AuthHexData) {
		// decode hex
		authData, err := hex.DecodeString(dto.AuthHexData)
		if err != nil {
			return vo.ParameterError("AuthHexData", err.Error())
		}
		// set client payload Data pb
		err = emptyAccountInfo.SetCliPayloadData(authData)
		if err != nil {
			return vo.ParameterError("AuthHexData", err.Error())
		}
	}
	if dto.EdgeRouting != "" {
		routingInfo, _ := base64.StdEncoding.DecodeString(dto.EdgeRouting)
		emptyAccountInfo.SetRoutingInfo(routingInfo)
	}
	if dto.IdentityPriKey != "" {
		emptyAccountInfo.SetStaticPriKey(dto.IdentityPriKey)
	}
	if dto.IdentityPubKey != "" {
		emptyAccountInfo.SetStaticPubKey(dto.IdentityPubKey)
	}
	// set client static secret key
	err := emptyAccountInfo.SetStaticHdBase64Keys(dto.StaticPriKey, dto.StaticPubKey)
	if err != nil {
		return vo.ParameterError("StaticPriKey or  StaticPubKey", err.Error())
	}

	var (
		app   *app2.WaApp
		exist bool
	)
	// does it exist
	if app, exist = GetWSApp(emptyAccountInfo.GetUserName()); exist {
		// TODO 重新上线？
		status := app.GetLoginStatus()
		if status == app2.Online {
			// 创建WSApp失败，实例存在！
			return vo.FailedCreate("Failed to create WSApp,Instance exists!")
		}
		// 重新登录
	} else {
		// create whatsApp
		app, err = CreateWSApp(emptyAccountInfo)
		if err != nil {
			return vo.FailedCreate(err.Error())
		}
	}

	// whether to use socks 5
	if dto.Socks5 != "" {
		app.SetNetWorkProxy(dto.Socks5)
	}

	var iPromise _interface.IPromise
	status := app.GetLoginStatus()
	if status == app2.Drops || status == app2.Disconnect {
		iPromise = app.RetryLogin()
	} else {
		// start login
		iPromise = app.WALogin()
	}
	if iPromise == nil {
		iPromise = app.WALogin()
	}
	Any, err := iPromise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// login result
	loginState, ok := Any.(app2.LoginStatus)
	if !ok || loginState != app2.Online {
		RemoveWSApp(emptyAccountInfo.GetUserName())
	} else {
		// set new message notify
		app.SetNewMessageNotify(func(message *entity.ChatMessage) {
			wslog.GetLogger().Ctx(app.Ctx()).Info(
				fmt.Sprintf(
					"Received a new message from:[ %s ] type:[ %s ] Content:[ %s ]",
					message.From(), message.ContextType(), message.GetContent().GetCONVERSATION(),
				),
			)
			// save message
			_ = SaveNewMessage(app.GetUserName(), message)
		})
	}
	msg := "登录失败"
	if loginState == app2.Online {
		msg = "登录成功"
	}
	return vo.SuccessJson(loginState.String(), app.GetPlatform(), msg)
}

// 退出登录
func LogOutService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		return vo.AnErrorOccurred(fmt.Errorf("账号%s已下线", k))
	}
	app.NewtWorkClose()
	app.SetLoginStatusOne(app2.Disconnect)
	RemoveWSApp(app.GetUserName())
	//登录成功开启
	connectMgr := app2.WXServer.GetWXConnectMgr()
	//查询该链接是否存在
	iwxConnect := connectMgr.GetWXConnectByWXID(k)
	if iwxConnect != nil {
		iwxConnect.Stop()
	}
	return vo.Success(gin.H{"status": 200, "msg": fmt.Sprintf("账号%s已下线", k)}, app.GetPlatform(), "success！")
}

// 获取商业版类型
func GetBusinessCategoryService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		return vo.AnErrorOccurred(fmt.Errorf("账号%s已下线", k))
	}
	respQr := app.SendCategories()
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
	resp := iqResult.GetCategories()
	if resp == nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{
		"status":     200,
		"msg":        "ok",
		"categories": resp,
	}, app.GetPlatform(), "ok")
}

// SetBusinessCategoryService 设置商业类型与名称
func SetBusinessCategoryService(k string, dto dto.SetBusinessCategoryDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		return vo.AnErrorOccurred(fmt.Errorf("账号%s已下线", k))
	}
	_ = app.SendBusinessProfile(dto.CategoryId)
	//_=app.SendBusinessPresenceAvailable(dto.Name)
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "设置成功!",
	}, app.GetPlatform(), "ok")
}

// SetNetWorkProxyService 设置代理
func SetNetWorkProxyService(k string, dto dto.SetNetWorkProxyDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		return vo.AnErrorOccurred(fmt.Errorf("账号%s已下线", k))
	}

	app.SetNetWorkProxy("")
	if dto.Socks5 != "" {
		app.SetNetWorkProxy(dto.Socks5)
	}
	app.SetLoginStatus(app2.Drops)
	// 防止重复重新登录
	go app.SetNetWorkProxyLogin()
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "设置代理成功!",
	}, app.GetPlatform(), "ok")
}

// HasUnsentPreKeysService
func HasUnsentPreKeysService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		return vo.AnErrorOccurred(fmt.Errorf("账号%s已下线", k))
	}
	return vo.Success(gin.H{
		"status": 200,
		"data":   app.HasUnsentPreKeys(),
		"msg":    "ok!",
	}, app.GetPlatform(), "ok")
}
