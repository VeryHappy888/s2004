package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

// AddGroupMemberController
func AddGroupMemberController(ctx *gin.Context) {
	groupDto := &dto.GroupDto{}
	// Validate JSon data
	if !validateData(ctx, &groupDto) {
		return
	}
	//Let the service handle
	resp := service.AddGroupMemberService(ctx.Param("key"), *groupDto)
	ctx.JSON(http.StatusOK, &resp)
}

// GetGroupMemberController 获取群成员
func GetGroupMemberController(ctx *gin.Context) {
	dto := &dto.GroupCodeDto{}
	if !validateData(ctx, &dto) {
		return
	}
	resp := service.GetGroupMemberService(ctx.Param("key"), *dto)
	ctx.JSON(http.StatusOK, &resp)
}

// CreateGroupController
func CreateGroupController(ctx *gin.Context) {
	groupDto := &dto.GroupDto{}
	// Validate JSon data
	if !validateData(ctx, &groupDto) {
		return
	}
	//Let the service handle
	resp := service.CreateGroupService(ctx.Param("key"), *groupDto)
	ctx.JSON(http.StatusOK, &resp)
}

// CreateGroupInviteController 通过code进群
func CreateGroupInviteController(ctx *gin.Context) {
	dto := &dto.ScanCodeDto{}
	if !validateData(ctx, &dto) {
		return
	}
	//Let the service handle
	resp := service.CreateGroupInviteService(ctx.Param("key"), dto.Code)
	ctx.JSON(http.StatusOK, &resp)
}

// GetGroupCodeController 获取群二维码
func GetGroupCodeController(ctx *gin.Context) {
	groupDto := &dto.GroupCodeDto{}
	if !validateData(ctx, &groupDto) {
		return
	}
	resp := service.GetGroupCodeService(ctx.Param("key"), *groupDto)
	ctx.JSON(http.StatusOK, &resp)
}

//设置群描述
func SetGroupDescController(ctx *gin.Context) {
	dto := &dto.GroupDescDto{}
	if !validateData(ctx, &dto) {
		return
	}
	resp := service.SetGroupDescService(ctx.Param("key"), *dto)
	ctx.JSON(http.StatusOK, &resp)
}

// SetGroupAdminController 设置群管理
func SetGroupAdminController(ctx *gin.Context) {
	groupDto := &dto.GroupAdminDto{}
	if !validateData(ctx, &groupDto) {
		return
	}
	resp := service.SetGroupAdminService(ctx.Param("key"), *groupDto)
	ctx.JSON(http.StatusOK, &resp)
}

// LogOutGroupController 退出群组
func LogOutGroupController(ctx *gin.Context) {
	groupDto := &dto.GroupCodeDto{}
	if !validateData(ctx, &groupDto) {
		return
	}
	resp := service.SendLogOutGroupService(ctx.Param("key"), *groupDto)
	ctx.JSON(http.StatusOK, &resp)
}
