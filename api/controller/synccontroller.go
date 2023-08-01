package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"ws-go/api/dto"
	"ws-go/api/service"
)

// SyncContactController
func SyncContactController(ctx *gin.Context) {
	syncDto := &dto.SyncContactDto{}
	// Validate JSon data
	if !validateData(ctx, &syncDto) {
		return
	}
	//Let the service handle
	resp := service.SyncContactService(ctx.Param("key"), *syncDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SyncAddOneContactsController -添加单个联系人
func SyncAddOneContactsController(ctx *gin.Context) {
	syncDto := &dto.SyncAddOneContactsDto{}
	// Validate JSon data
	if !validateData(ctx, &syncDto) {
		return
	}
	numbers := strings.ReplaceAll(syncDto.Numbers, "+", "")
	//Let the service handle
	resp := service.SyncAddOneContactsService(ctx.Param("key"), numbers)
	ctx.JSON(http.StatusOK, &resp)
}
