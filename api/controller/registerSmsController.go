package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

// SendRegisterSmsController 发送注册验证码
func SendRegisterSmsController(ctx *gin.Context) {
	sendPhoneCodeDto := &dto.SendVerifyCodeDto{}
	// Validate JSon data
	if !validateData(ctx, &sendPhoneCodeDto) {
		return
	}
	resp := service.SendRegisterSmsService(*sendPhoneCodeDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendBusinessRegisterSmsController  发送商业版注册验证码
func SendBusinessRegisterSmsController(ctx *gin.Context) {
	sendPhoneCodeDto := &dto.SendVerifyCodeDto{}
	// Validate JSon data
	if !validateData(ctx, &sendPhoneCodeDto) {
		return
	}
	resp := service.SendBusinessRegisterSmsService(*sendPhoneCodeDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendRegisterVerifyController 验证注册验证码
func SendRegisterVerifyController(ctx *gin.Context) {
	sendPhoneCodeDto := &dto.SendRegisterVerifyDto{}
	// Validate JSon data
	if !validateData(ctx, &sendPhoneCodeDto) {
		return
	}
	resp := service.SendRegisterService(*sendPhoneCodeDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendBusinessRegisterVerifyController 商业版验证注册验证码
func SendBusinessRegisterVerifyController(ctx *gin.Context) {
	sendPhoneCodeDto := &dto.SendRegisterVerifyDto{}
	// Validate JSon data
	if !validateData(ctx, &sendPhoneCodeDto) {
		return
	}
	resp := service.SendBusinessRegisterService(*sendPhoneCodeDto)
	ctx.JSON(http.StatusOK, &resp)
}

// GetPhoneExistsController 查询账号是否存在
func GetPhoneExistsController(ctx *gin.Context) {
	sendPhoneCodeDto := &dto.SendVerifyCodeDto{}
	// Validate JSon data
	if !validateData(ctx, &sendPhoneCodeDto) {
		return
	}
	resp := service.GetPhoneExistService(*sendPhoneCodeDto)
	ctx.JSON(http.StatusOK, &resp)
}

// GetBusinessPhoneExistsController 查询商业版本账号是否存在
func GetBusinessPhoneExistsController(ctx *gin.Context) {
	sendPhoneCodeDto := &dto.SendVerifyCodeDto{}
	// Validate JSon data
	if !validateData(ctx, &sendPhoneCodeDto) {
		return
	}
	resp := service.GetBusinessPhoneExistService(*sendPhoneCodeDto)
	ctx.JSON(http.StatusOK, &resp)
}
