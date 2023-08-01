package service

import (
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"runtime"
	"ws-go/api/dto"
	"ws-go/api/vo"
	"ws-go/protocol/waproto"
	"ws-go/waver"
)

// GenAuthDataService 生成认证数据
func GenAuthDataService(dto dto.AuthDataDto) vo.Resp {
	// check parameters
	if dto.GetUsername() == 0 || isEmpty(dto.GetPushName()) {
		return vo.IncompleteParameters()
	}
	// check user agent
	userAgent := dto.GetUserAgent()
	if userAgent == nil || isEmpty(userAgent.GetDevice()) ||
		isEmpty(userAgent.GetOsBuildNumber()) ||
		isEmpty(userAgent.GetManufacturer()) ||
		isEmpty(userAgent.GetPhoneId()) ||
		isEmpty(userAgent.GetOsVersion()) {
		return vo.IncompleteParameters()
	}
	// set other parameters
	dto.ShortConnect = proto.Bool(false)
	wifi := waproto.ClientPayload_WIFI
	dto.ConnectType = &wifi
	// set app version
	platform := userAgent.GetPlatform().String()
	//如果为安卓普通版
	if platform == "ANDROID" {
		dto.UserAgent.AppVersion.Primary = proto.Uint32(waver.GetVersion42().Primary)
		dto.UserAgent.AppVersion.Secondary = proto.Uint32(waver.GetVersion42().Secondary)
		dto.UserAgent.AppVersion.Tertiary = proto.Uint32(waver.GetVersion42().Tertiary)
		dto.UserAgent.AppVersion.Quaternary = proto.Uint32(waver.GetVersion42().Quaternary)
		dto.Oc = proto.Bool(true)
		dto.Lc = proto.Uint32(1)
		dto.Ok_36 = proto.Uint32(2016)
		dto.Ok_37 = proto.Uint32(256)
	} else if platform == "PLATFORM_10" {
		//安卓企业版
		dto.UserAgent.AppVersion.Primary = proto.Uint32(waver.GetBusinessVersion42().Primary)
		dto.UserAgent.AppVersion.Secondary = proto.Uint32(waver.GetBusinessVersion42().Secondary)
		dto.UserAgent.AppVersion.Tertiary = proto.Uint32(waver.GetBusinessVersion42().Tertiary)
		dto.UserAgent.AppVersion.Quaternary = proto.Uint32(waver.GetBusinessVersion42().Quaternary)
	} else {
		return vo.AnErrorOccurred(fmt.Errorf("platform参数不正确%s", dto.UserAgent.Platform.String()))
	}
	fmt.Sprintln("platform参数=%s", dto.UserAgent.Platform.String())
	// set locale
	if dto.UserAgent.LocaleLanguageIso_639_1 == nil {
		dto.UserAgent.LocaleLanguageIso_639_1 = proto.String("zh")
	}
	if dto.UserAgent.LocaleCountryIso_3166_1Alpha_2 == nil {
		dto.UserAgent.LocaleCountryIso_3166_1Alpha_2 = proto.String("CN")
	}
	sysType := runtime.GOOS
	if sysType == "darwin" {
		return vo.Resp{Code: 0, Data: gin.H{"status": 200,
			"msg": "ok", "AuthHexData": ""}} //hex.EncodeToString(authData)
	}
	// marshal auth data
	authData, err := proto.Marshal(&dto)
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// encode hex
	return vo.Resp{Code: 0, Data: gin.H{"status": 200,
		"msg": "ok", "AuthHexData": hex.EncodeToString(authData)}}
}
