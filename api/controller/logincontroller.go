package controller

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
	"ws-go/api/vo"
)

// CAwIBQ==
// WALoginController @Summary 数据登录
func WALoginController(ctx *gin.Context) {
	//fmt.Println("数据登录开始........")
	loginDto := &dto.LoginDto{}
	// Validate JSon data
	if !validateData(ctx, &loginDto) {
		return
	}
	if loginDto.ClientStaticKeypair != "" {
		piarKey, _ := base64.StdEncoding.DecodeString(loginDto.ClientStaticKeypair)
		loginDto.StaticPriKey = base64.StdEncoding.EncodeToString(piarKey[:32])
		loginDto.StaticPubKey = base64.StdEncoding.EncodeToString(piarKey[32:])
	}
	//Let the service handle
	resp := service.LoginService(*loginDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SetNetWorkProxyController 设置代理
func SetNetWorkProxyController(ctx *gin.Context) {
	dto := dto.SetNetWorkProxyDto{}
	if !validateData(ctx, &dto) {
		return
	}
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusOK, vo.IncompleteParameters())
		return
	}
	resp := service.SetNetWorkProxyService(key, dto)
	ctx.JSON(http.StatusOK, &resp)
}

// HasUnsentPreKeysController
func HasUnsentPreKeysController(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusOK, vo.IncompleteParameters())
		return
	}
	resp := service.HasUnsentPreKeysService(key)
	ctx.JSON(http.StatusOK, &resp)
}

// LogOutController  退出登录
func LogOutController(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusOK, vo.IncompleteParameters())
		return
	}
	//Let the service handle
	resp := service.LogOutService(key)
	ctx.JSON(http.StatusOK, &resp)
}

// GetBusinessCategoryController 获取商业类型
func GetBusinessCategoryController(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusOK, vo.IncompleteParameters())
		return
	}
	//Let the service handle
	resp := service.GetBusinessCategoryService(key)
	ctx.JSON(http.StatusOK, &resp)
}

// SetBusinessCategoryController 设置商业类型与商业名称
func SetBusinessCategoryController(ctx *gin.Context) {
	dto := dto.SetBusinessCategoryDto{}
	if !validateData(ctx, &dto) {
		return
	}
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusOK, vo.IncompleteParameters())
		return
	}
	//Let the service handle
	resp := service.SetBusinessCategoryService(key, dto)
	ctx.JSON(http.StatusOK, &resp)
}
