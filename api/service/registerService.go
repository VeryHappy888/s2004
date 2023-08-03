package service

import (
	"fmt"
	"strconv"
	"ws-go/api/dto"
	"ws-go/api/vo"
	"ws-go/protocol/db"
	"ws-go/protocol/register"
)

// SendRegisterSmsService 发送注册验证码
func SendRegisterSmsService(dto dto.SendVerifyCodeDto) vo.Resp {
	if dto.Cc == 0 || dto.Phone == "" {
		return vo.IncompleteParameters()
	}
	r := &register.WaRegistration{
		Lc:       dto.Lc,
		Lg:       dto.Lg,
		WAId:     dto.Phone,
		Proxy:    dto.Socks5,
		DeEnv:    register.Version(dto.Platform),
		DeConfig: register.GenerateWAConfig(dto.Lc),
	}
	cc := strconv.Itoa(int(dto.Cc))
	//添加参数
	_, err := r.ExistsRequest(cc, dto.Phone)
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	key := fmt.Sprintf("whatsapp:sms:%v%v", cc, dto.Phone)
	r.DeConfig = r.DeConfig.SetRegistrationVal()
	_ = db.SETExpirationObj(key, r.DeConfig, 60*60*24*1)
	Method := register.SMS
	if dto.Method == 1 {
		Method = register.VOICE
	}
	resp, err := r.RequestVerifyCode(cc, dto.Phone, Method)
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(resp, register.GetPlatform(dto.Platform), "ok")
}

// SendBusinessRegisterSmsService 发送商业版注册验证码
func SendBusinessRegisterSmsService(dto dto.SendVerifyCodeDto) vo.Resp {
	if dto.Cc == 0 || dto.Phone == "" {
		return vo.IncompleteParameters()
	}
	r := &register.WaRegistration{
		Lc:       dto.Lc,
		Lg:       dto.Lg,
		WAId:     dto.Phone,
		Proxy:    dto.Socks5,
		DeEnv:    register.Version(dto.Platform),
		DeConfig: register.GenerateWAConfig(dto.Lc),
	}
	cc := strconv.Itoa(int(dto.Cc))
	//添加参数
	_, err := r.BusinessExistRequest(cc, dto.Phone)
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	key := fmt.Sprintf("whatsapp:business-sms:%v%v", cc, dto.Phone)
	r.DeConfig = r.DeConfig.SetRegistrationVal()
	_ = db.SETExpirationObj(key, r.DeConfig, 60*60*24*1)
	Method := register.SMS
	if dto.Method == 1 {
		Method = register.VOICE
	}
	resp, err := r.RequestVerifyCode(cc, dto.Phone, Method)
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(resp, register.GetPlatform(dto.Platform), "ok")
}

// SendRegisterVerify 注册验证
func SendRegisterService(dto dto.SendRegisterVerifyDto) vo.Resp {
	if dto.Cc == 0 || dto.Phone == "" || dto.Code == "" {
		return vo.IncompleteParameters()
	}
	r := &register.WaRegistration{
		Lc:    dto.Lc,
		Lg:    dto.Lg,
		WAId:  dto.Phone,
		Proxy: dto.Socks5,
		DeEnv: register.Version(dto.Platform),
	}
	v := &register.WAConfig{}
	cc := strconv.Itoa(int(dto.Cc))
	key := fmt.Sprintf("whatsapp:sms:%v%v", cc, dto.Phone)
	exists, err := db.Exists(key)
	if err != nil {
		fmt.Println(err.Error())
		return vo.AnErrorOccurred(err)
	}
	if exists {
		err := db.GETObj(key, v)
		if err != nil {
			fmt.Println(err.Error())
			return vo.AnErrorOccurred(err)
		}
		r.DeConfig = v
		fmt.Println("cache get is ok")
	} else {
		r.DeConfig = r.DeConfig.SetRegistrationVal()
	}
	resp, err := r.FinishRegistration(cc, dto.Phone, dto.Code)
	if err != nil {
		fmt.Println(err.Error())
		return vo.AnErrorOccurred(err)
	}
	if resp.Status == "ok" {
		return vo.Success(r.DeConfig.GenConfigJson(resp.EdgeRoutingInfo), register.GetPlatform(dto.Platform), "ok")
	}
	return vo.Success(resp, register.GetPlatform(dto.Platform), "ok")
}

// SendBusinessRegisterService 商业版注册验证
func SendBusinessRegisterService(dto dto.SendRegisterVerifyDto) vo.Resp {
	if dto.Cc == 0 || dto.Phone == "" || dto.Code == "" {
		return vo.IncompleteParameters()
	}
	r := &register.WaRegistration{
		Lc:    dto.Lc,
		Lg:    dto.Lg,
		WAId:  dto.Phone,
		Proxy: dto.Socks5,
		DeEnv: register.Version(dto.Platform),
	}
	v := &register.WAConfig{}
	cc := strconv.Itoa(int(dto.Cc))
	key := fmt.Sprintf("whatsapp:business-sms:%v%v", cc, dto.Phone)
	exists, err := db.Exists(key)
	if err != nil {
		fmt.Println(err.Error())
		return vo.AnErrorOccurred(err)
	}
	if exists {
		err := db.GETObj(key, v)
		if err != nil {
			fmt.Println(err.Error())
			return vo.AnErrorOccurred(err)
		}
		r.DeConfig = v
		fmt.Println("cache get is ok")
	} else {
		r.DeConfig = r.DeConfig.SetRegistrationVal()
	}
	resp, err := r.FinishBusinessRegistration(cc, dto.Phone, dto.Code)
	if err != nil {
		fmt.Println(err.Error())
		return vo.AnErrorOccurred(err)
	}
	if resp.Status == "ok" {
		return vo.Success(r.DeConfig.GenConfigJson(resp.EdgeRoutingInfo), register.GetPlatform(dto.Platform), "ok")
	}
	return vo.Success(resp, register.GetPlatform(dto.Platform), "ok")
}

// SendRegisterVerify 查询账号是否存在
func GetPhoneExistService(dto dto.SendVerifyCodeDto) vo.Resp {
	if dto.Cc == 0 || dto.Phone == "" {
		return vo.IncompleteParameters()
	}
	r := &register.WaRegistration{
		Lc:       "US",
		Lg:       "en",
		WAId:     dto.Phone,
		Proxy:    dto.Socks5,
		DeEnv:    register.Version(dto.Platform),
		DeConfig: register.GenerateWAConfig("US"),
	}
	r.DeConfig = r.DeConfig.SetRegistrationVal()
	//添加参数
	cc := strconv.Itoa(int(dto.Cc))
	resp, err := r.ExistsRequest(cc, dto.Phone)
	if err != nil {
		fmt.Println(err.Error())
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(resp, register.GetPlatform(dto.Platform), "ok")
}

// GetBusinessPhoneExistService 查询商业版账号是否存在
func GetBusinessPhoneExistService(dto dto.SendVerifyCodeDto) vo.Resp {
	if dto.Cc == 0 || dto.Phone == "" {
		return vo.IncompleteParameters()
	}
	r := &register.WaRegistration{
		Lc:       "US",
		Lg:       "en",
		WAId:     dto.Phone,
		Proxy:    dto.Socks5,
		DeEnv:    register.Version(dto.Platform),
		DeConfig: register.GenerateWAConfig("US"),
	}
	r.DeConfig = r.DeConfig.SetRegistrationVal()
	//添加参数
	cc := strconv.Itoa(int(dto.Cc))
	resp, err := r.BusinessExistRequest(cc, dto.Phone)
	if err != nil {
		fmt.Println(err.Error())
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(resp, register.GetPlatform(dto.Platform), "ok")
}
