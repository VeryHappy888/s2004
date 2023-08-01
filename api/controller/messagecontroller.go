package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

// SyncNewMessageController
func SyncNewMessageController(ctx *gin.Context) {
	resp := service.SyncNewMessageService(ctx.Param("key"))
	ctx.JSON(http.StatusOK, &resp)
}

// SendTextMessageController
func SendTextMessageController(ctx *gin.Context) {
	MsgDto := &dto.MessageDto{}
	// Validate JSon data
	if !validateData(ctx, &MsgDto) {
		return
	}
	//Let the service handle
	resp := service.SendTextMessageService(ctx.Param("key"), *MsgDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendImageMessageController
func SendImageMessageController(ctx *gin.Context) {
	MsgDto := &dto.MessageImageDto{}
	// Validate JSon data
	if !validateData(ctx, &MsgDto) {
		return
	}

	resp := service.SendImageMessage(ctx.Param("key"), *MsgDto)

	ctx.JSON(http.StatusOK, &resp)
}

// SendMessageDownloadController
func SendMessageDownloadController(ctx *gin.Context) {
	DownloadMessageDto := &dto.DownloadMessageDto{}
	// Validate JSon data
	if !validateData(ctx, &DownloadMessageDto) {
		return
	}
	resp := service.SendMessageDownloadService(ctx.Param("key"), *DownloadMessageDto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendAudioMessageController
func SendAudioMessageController(ctx *gin.Context) {
	Dto := &dto.MessageAudioDto{}
	// Validate JSon data
	if !validateData(ctx, &Dto) {
		return
	}
	resp := service.SendAudioMessageService(ctx.Param("key"), *Dto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendVideoMessageController
func SendVideoMessageController(ctx *gin.Context) {
	Dto := &dto.MessageVideoDto{}
	// Validate JSon data
	if !validateData(ctx, &Dto) {
		return
	}
	resp := service.SendVideoMessageService(ctx.Param("key"), *Dto)
	ctx.JSON(http.StatusOK, &resp)
}

// SendVcardMessageController
func SendVcardMessageController(ctx *gin.Context) {
	Dto := &dto.VcardDto{}
	// Validate JSon data
	if !validateData(ctx, &Dto) {
		return
	}
	resp := service.SendVcardMessageService(ctx.Param("key"), *Dto)
	ctx.JSON(http.StatusOK, &resp)
}
