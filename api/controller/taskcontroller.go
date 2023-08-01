package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

func AddTaskController(ctx *gin.Context) {
	taskDto := &dto.TaskDto{}
	// Validate JSon data
	if !validateData(ctx, &taskDto) {
		return
	}
	//Let the service handle
	resp := service.AddTaskService(ctx.Param("key"), *taskDto)
	ctx.JSON(http.StatusOK, &resp)
}
