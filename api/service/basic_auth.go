package service

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"time"
	"ws-go/protocol/db"
)

/***
认证
*/

func RsqK(c *gin.Context, msg string) {
	c.AbortWithStatus(http.StatusCreated)
	respVo := &RespVo{
		Code:    http.StatusOK,
		Message: msg,
		Data:    http.StatusCreated,
	}
	c.JSON(http.StatusOK, respVo)
}

type Auth struct {
	Code int32
	Data int32
	Msg  string
}

// {"code":0,"message":"","data":""}
type RespVo struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 发送GET请求
// url：         请求地址
// response：    请求返回的内容
func Get(url string) string {
	// 超时时间：15秒
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}
	return result.String()
}

// 添加授权缓存
func CacheAuthAdd(key string, val *Auth) {
	cache := db.GetCaChe()
	isContains, err := cache.Contains(key)
	if err != nil {
		panic("cache errr 1")
	}
	if !isContains {
		cache.Set(key, val, time.Hour*1)
		return
	}
}

// 查询授权缓存
func GetCacheAuth(key string) *Auth {
	cache := db.GetCaChe()
	isContains, err := cache.Contains(key)
	if err != nil {
		panic("cache errr 2")
	}
	if !isContains {
		return nil
	}
	resp, err := cache.Get(key)
	if err != nil {
		panic("cache errr 3")
	}
	return resp.(*Auth)
}
