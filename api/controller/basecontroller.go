package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/vo"
)

// 验证并判断,是否在线
func validateData(ctx *gin.Context, model interface{}) bool {
	err := ctx.ShouldBindJSON(&model)
	if err != nil {
		ctx.JSON(http.StatusOK, vo.SubmitDataError())
		ctx.Abort()
		return false
	}
	return true
}
