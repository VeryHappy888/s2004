package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-go/api/dto"
	"ws-go/api/service"
)

// SetProfilePictureController  //设置头像
func SetProfilePictureController(ctx *gin.Context) {
	picDto := &dto.PictureInfoDto{}
	// Validate JSon data
	if !validateData(ctx, &picDto) {
		return
	}
	//Let the service handle
	resp := service.SetProfilePictureService(ctx.Param("key"), *picDto)
	ctx.JSON(http.StatusOK, &resp)
}

//SetNickNameController 设置名称
func SetNickNameController(ctx *gin.Context) {
	dto := &dto.NickNameDto{}
	if !validateData(ctx, &dto) {
		return
	}
	resp := service.SetNickNameService(ctx.Param("key"), *dto)
	ctx.JSON(http.StatusOK, &resp)
}

// GetProfilePictureController  获取头像，别人的和自己的
func GetProfilePictureController(ctx *gin.Context) {
	picDto := &dto.PictureInfoDto{}
	// Validate JSon data
	if !validateData(ctx, &picDto) {
		return
	}
	//Let the service handle
	resp := service.GetProfilePictureService(ctx.Param("key"), *picDto)
	ctx.JSON(http.StatusOK, &resp)
}

// GetPreviewController 获取头像 小图
func GetPreviewController(ctx *gin.Context) {
	picDto := &dto.PictureInfoDto{}
	// Validate JSon data
	if !validateData(ctx, &picDto) {
		return
	}
	//Let the service handle
	resp := service.GetPreviewService(ctx.Param("key"), *picDto)
	ctx.JSON(http.StatusOK, &resp)
}

// GetProfileController 获取自己的个人信息
func GetProfileController(ctx *gin.Context) {
	resp := service.GetProfileService(ctx.Param("key"))
	ctx.JSON(http.StatusOK, &resp)
}

// GetQrController 获取二维码
func GetQrController(ctx *gin.Context) {
	resp := service.GetQrService(ctx.Param("key"))
	ctx.JSON(http.StatusOK, &resp)
}

// SetQrRevokeController 重置二维码
func SetQrRevokeController(ctx *gin.Context) {
	resp := service.SetQrRevokeService(ctx.Param("key"))
	ctx.JSON(http.StatusOK, &resp)
}

// ScanCodeController 扫描二维码
func ScanCodeController(ctx *gin.Context) {
	scanDto := &dto.ScanCodeDto{}
	// Validate JSon data
	if !validateData(ctx, &scanDto) {
		return
	}
	resp := service.ScanCodeService(ctx.Param("key"), scanDto.Code, scanDto.OpCode)
	ctx.JSON(http.StatusOK, &resp)
}

// SetStateController  上传个性签名
func SetStateController(ctx *gin.Context) {
	picDto := &dto.SetStateDto{}
	// Validate JSon data
	if !validateData(ctx, &picDto) {
		return
	}
	//Let the service handle
	resp := service.SetStateService(ctx.Param("key"), *picDto)
	ctx.JSON(http.StatusOK, &resp)
}

//GetStateController 获取个性签名
func GetStateController(ctx *gin.Context) {
	picDto := &dto.GetStateDto{}
	// Validate JSon data
	if !validateData(ctx, &picDto) {
		return
	}
	//Let the service handle
	resp := service.GetStateService(ctx.Param("key"), *picDto)
	ctx.JSON(http.StatusOK, &resp)
}

//TwoVerifyController  两步验证接口
func TwoVerifyController(ctx *gin.Context) {
	picDto := &dto.TwoVerifyDto{}
	// Validate JSon data
	if !validateData(ctx, &picDto) {
		return
	}
	//Let the service handle
	resp := service.TwoVerifyService(ctx.Param("key"), *picDto)
	ctx.JSON(http.StatusOK, &resp)
}
