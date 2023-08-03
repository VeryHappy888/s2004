package register

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
)

// 环境
type Env interface {
	GetToken(phone string, sign ...string) (string, error)
	WAUserAgent() string
	DeviceName() string
	EnvInfo() *EnvBase
}

func TestToken(phone, key, dexMd5, sig string) (string, error) {
	return CalcWATokenA(phone, key, dexMd5, sig)
}

func CalcWATokenA(phone, key, dexMd5, sig string) (string, error) {
	var sourceData string
	var err error
	if len(phone) == 0 || len(key) == 0 || len(dexMd5) == 0 || len(sig) == 0 {
		// 缺少重要参数计算token失败
		return "", errors.New("Failed to calculate token due to missing important parameters")
	}
	// key
	_key, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		sourceData = key
		err = fmt.Errorf("Base64 decode failure source %s", sourceData)
		return "", err
	}
	// _md5
	_dexMd5, err := base64.StdEncoding.DecodeString(dexMd5)
	if err != nil {
		sourceData = dexMd5
		err = fmt.Errorf("Base64 decode failure source %s", sourceData)
		return "", err
	}
	// _sig
	_sig, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		sourceData = sig
		err = fmt.Errorf("Base64 decode failure source %s", sourceData)
		return "", err
	}

	h := hmac.New(sha1.New, _key[:64])
	h.Write(_sig)
	h.Write(_dexMd5)
	h.Write([]byte(phone))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
