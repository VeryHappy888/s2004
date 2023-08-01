package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

// ScanNumberController  //扫单个号
func ScanNumberController(ctx *gin.Context) {
	dto := &dto.ScanNumberDto{}
	// Validate JSon data
	if !validateData(ctx, &dto) {
		return
	}
	//Let the service handle
	resp := service.ScanNumberService(ctx.Param("key"), dto.Number)
	ctx.JSON(http.StatusOK, &resp)
}

// ExistenceController 查询号码是否存在
func ExistenceController(ctx *gin.Context) {
	dto := &dto.ExistenceDto{}
	// Validate JSon data
	if !validateData(ctx, &dto) {
		return
	}
	//Let the service handle
	resp := service.ExistenceService(ctx.Param("key"), dto.Number)
	ctx.JSON(http.StatusOK, &resp)
}
