package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"time"
	"ws-go/api/dto"
	"ws-go/api/vo"
	"ws-go/protocol/app"
	"ws-go/protocol/db"
	"ws-go/protocol/entity"
	"ws-go/protocol/media"
	"ws-go/protocol/msg"
	"ws-go/protocol/utils/promise"
)

// SyncNewMessage 获取新消息
func SyncNewMessageService(k string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	// get message
	chatMessages := GetNewMessages(k)
	messages := make([]gin.H, 0)
	for _, message := range chatMessages {
		content := message.GetContent()
		content.SKMSG = nil
		messages = append(messages, gin.H{
			"type":        message.ContextType(),
			"form":        message.From(),
			"participant": message.Participant(),
			"t":           message.T(),
			"content":     content,
		})
	}
	return vo.Resp{Code: 0, Data: messages}
}

// SendTextMessageService 发送文本消息Worldwide
func SendTextMessageService(k string, dto dto.MessageDto) vo.Resp {
	var subscribeAny promise.Any
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	// check parameter
	if isEmpty(dto.RecipientId) || isEmpty(dto.Content) {
		return vo.IncompleteParameters()
	}
	subscribeAny = dto.Subscribe
	//首次发送需要订阅&发送身份获取认证
	key := fmt.Sprintf("whatsapp:subscribe:%v:%s", k, dto.RecipientId)
	exists, errs := db.Exists(key)
	if errs != nil {
		fmt.Println(errs.Error())
		return vo.AnErrorOccurred(errs)
	}
	//如果不存在,保存&发送订阅消息
	if !exists {
		_ = db.SETExpirationObj(key, "subscribe", 60*60*5*1)
		app.SendPresencesSubscribe(dto.RecipientId)
		app.SendEncrypt(dto.RecipientId)
	}
	// chat state
	if dto.ChatState {
		// 发送聊天状态 在输入文字时候发送
		app.SendChatState(dto.RecipientId, dto.SentGroup, false)
		time.Sleep(time.Second)
		// 发送聊天状态 在输入文字时候发送
		app.SendChatState(dto.RecipientId, dto.SentGroup, true)
		time.Sleep(time.Second)
		return vo.Success(nil, app.GetPlatform(), "")
	}

	// init error
	var (
		err   error
		myMsg *msg.MySendMsg
	)
	// sent group
	if dto.SentGroup {
		// send group text message
		myMsg, err = app.SendGroupTextMessage(dto.RecipientId, dto.Content, dto.At, dto.StanzaId, dto.Participant, dto.Conversation)
	} else {
		// 发送正在输入
		if app.GetUserName() == "" {
			// 发送聊天状态 在输入文字时候发送
			app.SendChatState(dto.RecipientId, dto.SentGroup, false)
			time.Sleep(time.Second)
			app.SendChatState(dto.RecipientId, dto.SentGroup, true)
			time.Sleep(time.Second)
		}
		//send text message
		myMsg, err = app.SendTextMessage(dto.RecipientId, dto.Content, dto.At, dto.StanzaId, dto.Participant, dto.Conversation)
	}
	// handler err
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{"status": 200, "subscribe": subscribeAny, "msg": myMsg}, app.GetPlatform(), "Sent successfully！")
}

// 获取cdn
func getCdnInfo(app *app.WaApp) (*entity.MediaConn, error) {
	sync := app.SendMediaConIq()
	result, err := sync.GetResult()
	if err != nil {
		return nil, err
	}
	iqResult := result.(entity.IqResult)
	return iqResult.GetMediaConn(), nil
}

//发送图片消息
func SendImageMessage(k string, dto dto.MessageImageDto) vo.Resp {
	var subscribeAny promise.Any
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	result, _ := getCdnInfo(app)
	//上传操作
	fileByte, errs := base64.StdEncoding.DecodeString(dto.ImageBase64)
	if errs != nil {
		return vo.AnErrorOccurred(fmt.Errorf("base64转码失败 failed: %v", errs))
	}
	proxy := app.GetNetWorkProxy()
	directPath, url, mediaKey, fileEncSha256, fileSha256, fileLength, errs := media.UploadFor(proxy, ioutil.NopCloser(bytes.NewReader(fileByte)), media.MediaImage, result.HostName(), result.Auth())
	if errs != nil {
		return vo.AnErrorOccurred(fmt.Errorf("上传文件失败 failed: %v", errs))
	}
	subscribeAny = dto.Subscribe
	// PresencesSubscribe
	/*if dto.Subscribe {
		subscribe := app.SendPresencesSubscribe(dto.RecipientId)
		fmt.Println(subscribe)
		any, err := subscribe.GetResult()
		if err != nil {
			return vo.AnErrorOccurred(err)
		}
		subscribeAny = any
	}*/
	// init error
	var (
		err   error
		myMsg *msg.MySendMsg
	)

	//首次发送需要订阅
	key := fmt.Sprintf("whatsapp:subscribe:%v", k)
	exists, errs := db.Exists(key)
	if errs != nil {
		fmt.Println(errs.Error())
		return vo.AnErrorOccurred(errs)
	}
	//如果不存在,保存&发送订阅消息
	if !exists {
		_ = db.SETExpirationObj(key, "subscribe", 60*60*24*1)
		go app.SendPresencesSubscribe(dto.RecipientId)
	}
	// sent group
	if dto.SentGroup {
		// send group text message
		myMsg, err = app.SendGroupImageMessage(dto.RecipientId, dto.ImageBase64, url, directPath, mediaKey, fileEncSha256, fileSha256, fileLength)
	} else {
		//send text message
		myMsg, err = app.SendImageMessage(dto.RecipientId, dto.ImageBase64, url, directPath, mediaKey, fileEncSha256, fileSha256, fileLength)
	}
	// handler err
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{"status": 200, "subscribe": subscribeAny, "msg": myMsg}, app.GetPlatform(), "successfully！")
}

// SendAudioMessageService 发送语音消息
func SendAudioMessageService(k string, dto dto.MessageAudioDto) vo.Resp {
	var subscribeAny promise.Any
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	result, _ := getCdnInfo(app)
	//上传操作
	fileByte, errs := base64.StdEncoding.DecodeString(dto.AudioBase64)
	if errs != nil {
		return vo.AnErrorOccurred(fmt.Errorf("base64转码失败 failed: %v", errs))
	}
	proxy := app.GetNetWorkProxy()
	directPath, url, mediaKey, fileEncSha256, fileSha256, fileLength, errs := media.UploadFor(proxy, ioutil.NopCloser(bytes.NewReader(fileByte)), media.MediaAudio, result.HostName(), result.Auth())
	if errs != nil {
		return vo.AnErrorOccurred(fmt.Errorf("上传文件失败 failed: %v", errs))
	}
	subscribeAny = dto.Subscribe
	var (
		err   error
		myMsg *msg.MySendMsg
	)
	//首次发送需要订阅
	key := fmt.Sprintf("whatsapp:subscribe:%v", k)
	exists, errs := db.Exists(key)
	if errs != nil {
		fmt.Println(errs.Error())
		return vo.AnErrorOccurred(errs)
	}
	//如果不存在,保存&发送订阅消息
	if !exists {
		_ = db.SETExpirationObj(key, "subscribe", 60*60*24*1)
		go app.SendPresencesSubscribe(dto.RecipientId)
	}
	// sent group
	if dto.SentGroup {
		// send group text message
		myMsg, err = app.SendGroupAudioMessage(dto.RecipientId, dto.AudioBase64, url, directPath, mediaKey, fileEncSha256, fileSha256, fileLength)
	} else {
		//send text message
		myMsg, err = app.SendAudioMessage(dto.RecipientId, dto.AudioBase64, url, directPath, mediaKey, fileEncSha256, fileSha256, fileLength)
	}
	// handler err
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{"status": 200, "subscribe": subscribeAny, "msg": myMsg}, app.GetPlatform(), "successfully！")
}

// SendVideoMessageService 发送视频消息
func SendVideoMessageService(k string, dto dto.MessageVideoDto) vo.Resp {
	var subscribeAny promise.Any
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	result, _ := getCdnInfo(app)
	//上传操作
	fileByte, errs := base64.StdEncoding.DecodeString(dto.VideoBase64)
	if errs != nil {
		return vo.AnErrorOccurred(fmt.Errorf("base64转码失败 failed: %v", errs))
	}
	proxy := app.GetNetWorkProxy()
	directPath, url, mediaKey, fileEncSha256, fileSha256, fileLength, errs := media.UploadFor(proxy, ioutil.NopCloser(bytes.NewReader(fileByte)), media.MediaVideo, result.HostName(), result.Auth())
	if errs != nil {
		return vo.AnErrorOccurred(fmt.Errorf("上传文件失败 failed: %v", errs))
	}
	subscribeAny = dto.Subscribe
	var (
		err   error
		myMsg *msg.MySendMsg
	)
	//首次发送需要订阅
	key := fmt.Sprintf("whatsapp:subscribe:%v", k)
	exists, errs := db.Exists(key)
	if errs != nil {
		fmt.Println(errs.Error())
		return vo.AnErrorOccurred(errs)
	}
	//如果不存在,保存&发送订阅消息
	if !exists {
		_ = db.SETExpirationObj(key, "subscribe", 60*60*24*1)
		go app.SendPresencesSubscribe(dto.RecipientId)
	}
	// sent group
	if dto.SentGroup {
		// send group text message
		myMsg, err = app.SendGroupVideoMessage(dto.RecipientId, dto.ThumbnailBase64, url, directPath, mediaKey, fileEncSha256, fileSha256, fileLength)
	} else {
		//send text message
		myMsg, err = app.SendVideoMessage(dto.RecipientId, dto.ThumbnailBase64, url, directPath, mediaKey, fileEncSha256, fileSha256, fileLength)
	}
	// handler err
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{"status": 200, "subscribe": subscribeAny, "msg": myMsg}, app.GetPlatform(), "successfully！")
}

// SendVcardMessageService 发送名片消息
func SendVcardMessageService(k string, dto dto.VcardDto) vo.Resp {
	var subscribeAny promise.Any
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	subscribeAny = dto.Subscribe
	var (
		err   error
		myMsg *msg.MySendMsg
	)
	//首次发送需要订阅
	key := fmt.Sprintf("whatsapp:subscribe:%v", k)
	exists, errs := db.Exists(key)
	if errs != nil {
		fmt.Println(errs.Error())
		return vo.AnErrorOccurred(errs)
	}
	//如果不存在,保存&发送订阅消息
	if !exists {
		_ = db.SETExpirationObj(key, "subscribe", 60*60*24*1)
		go app.SendPresencesSubscribe(dto.RecipientId)
	}
	// sent group
	if dto.SentGroup {
		// send group text message
		//myMsg, err = app.SendGroupVideoMessage(dto.RecipientId)
	} else {
		//send text message
		myMsg, err = app.SendVcardMessage(dto.RecipientId, dto.Tel, dto.VcardName)
	}
	// handler err
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{"status": 200, "subscribe": subscribeAny, "msg": myMsg}, app.GetPlatform(), "successfully！")
}

// SendMessageDownloadService 下载
func SendMessageDownloadService(k string, dto dto.DownloadMessageDto) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	key, err := base64.StdEncoding.DecodeString(dto.MediaKey)
	if err != nil {
		return vo.AnErrorOccurred(fmt.Errorf("验证key: %v\n", err))
	}
	typeMedia := media.MediaImage
	switch dto.MediaType {
	case "MediaImage":
		typeMedia = media.MediaImage
		break
	case "MediaVideo":
		typeMedia = media.MediaVideo
		break
	case "MediaAudio":
		typeMedia = media.MediaAudio
		break
	case "MediaDocument":
		typeMedia = media.MediaDocument
		break
	}
	proxy := app.GetNetWorkProxy()
	resp, err := media.Download(proxy, dto.Url, key, typeMedia, dto.FileLength)
	if err != nil {
		return vo.AnErrorOccurred(fmt.Errorf("下载文件失败: %v\n", err))
	}
	return vo.Success(gin.H{"status": 200, "msg": "ok", "data": base64.StdEncoding.EncodeToString(resp)}, app.GetPlatform(), "successfully！")
}
