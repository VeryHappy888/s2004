package register

import (
	"fmt"
)

const AndroidEnvLogTag = "AndroidEnvBase"

type AndroidEnvBase struct {
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

	BUILDVERSION string
}

func (a *AndroidEnvBase) GetToken(phone string) string {
	token, err := androidCalcWAToken(phone, a.KEY, a.MD5CLASSES, a.SIGNATURE)
	if err != nil {
		return ""
	}
	return token
}

func (a *AndroidEnvBase) WAUserAgent() string {
	//WhatsApp/2.21.15.16 Android/11 Device/Google-Pixel_4a_(5G)
	//WhatsApp/2.21.14.24 SMBA/11 Device/Google-Pixel_4a_(5G) //商业版
	return fmt.Sprintf("WhatsApp/%s %s/%s Device/%s-%s", a.VERSION, a.OSNAME, a.OSVERSION, a.MANUFACTURER, a.DEVICENAME)
}
