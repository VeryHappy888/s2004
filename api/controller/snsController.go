package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

//SnsTextPostController 发送文字动态
func SnsTextPostController(ctx *gin.Context) {
	snsTextDto := &dto.SnsTextDto{}
	// Validate JSon data
	if !validateData(ctx, &snsTextDto) {
		return
	}
	//Let the service handle
	resp := service.SnsTextService(ctx.Param("key"), *snsTextDto)
	ctx.JSON(http.StatusOK, &resp)
}
