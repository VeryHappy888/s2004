package register

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

type EnvBase struct {
	Env
	// app签名
	SIGNATURE string
	// whatsapp logo.png 和 内置秘钥 计算
	KEY string
	// dex Md5
	MD5CLASSES string
	// whatsapp 版本号
	VERSION string
	// 设备类型
	OSNAME string
	// 设备版本
	OSVERSION string
	// 制造商
	MANUFACTURER string
	// 设备名
	DEVICENAME string
	//平台
	PLATFORM string
}

func (a *EnvBase) GetToken(phone string, sign ...string) (string, error) {
	if a.PLATFORM == "Apple" {
		txt := fmt.Sprintf("%v%v%v", a.KEY, a.SIGNATURE, phone)
		h := md5.New()
		h.Write([]byte(txt))
		return hex.EncodeToString(h.Sum(nil)), nil
	}

	if len(phone) == 0 || len(a.KEY) == 0 || len(a.MD5CLASSES) == 0 || len(a.SIGNATURE) == 0 {
		// 缺少重要参数计算token失败
		return "", errors.New("Failed to calculate token due to missing important parameters")
	}
	// key
	_key, err := base64.StdEncoding.DecodeString(a.KEY)
	if err != nil {
		return "", err
	}
	// _md5
	_dexMd5, err := base64.StdEncoding.DecodeString(a.MD5CLASSES)
	if err != nil {
		return "", err
	}
	// _sig
	_sig, err := base64.StdEncoding.DecodeString(a.SIGNATURE)
	if err != nil {
		return "", err
	}

	h := hmac.New(sha1.New, _key[:64])
	h.Write(_sig)
	h.Write(_dexMd5)
	h.Write([]byte(phone))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (a *EnvBase) WAUserAgent() string {
	if a.PLATFORM == "android" {
		//WhatsApp/2.21.15.16 Android/11 Device/Google-Pixel_4a_(5G)
		//WhatsApp/2.21.14.24 SMBA/11 Device/Google-Pixel_4a_(5G) //商业版
		return fmt.Sprintf("WhatsApp/%s %s/%s Device/%s-%s", a.VERSION, a.OSNAME, a.OSVERSION, a.MANUFACTURER, a.DEVICENAME)
	}

	return fmt.Sprintf("WhatsApp/%s %s/%s Device/%s", a.VERSION, a.OSNAME, a.OSVERSION, a.DEVICENAME)
}

func (a *EnvBase) EnvInfo() *EnvBase {
	return a
}
