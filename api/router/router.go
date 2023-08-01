package router

import (
	"github.com/gin-gonic/gin"
	"ws-go/api/controller"
)

type SetMiddleWare = func(engine *gin.Engine)

func SetUpRouter(middleware SetMiddleWare, debug bool) *gin.Engine {
	//获取Gin实例
	r := gin.Default()
	//设置中间
	if middleware != nil {
		middleware(r)
	}
	//设置静态文件目录
	r.Static("static", "api/static")
	setUpConfig(r)
	setApi_V1(r)
	//setTemplate(r)
	return r
}

// 初始化应用设置
func setUpConfig(router *gin.Engine) {
	// 使用swagger自动生成接口文档
	//router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func setApi_V1(engine *gin.Engine) {
	ver := "/ws"

	//登录
	login := engine.Group(ver)
	{
		login.POST("/WALogin", controller.WALoginController)
		login.GET("/LogOut/:key", controller.LogOutController)
		login.POST("/GetPhoneExists", controller.GetPhoneExistsController)
		login.POST("/GetBusinessPhoneExists", controller.GetBusinessPhoneExistsController)
		login.POST("/SendRegisterSms", controller.SendRegisterSmsController)
		login.POST("/SendBusinessRegisterSms", controller.SendBusinessRegisterSmsController)
		login.POST("/SendRegisterVerify", controller.SendRegisterVerifyController)
		login.POST("/SendBusinessRegisterVerify", controller.SendBusinessRegisterVerifyController)
		login.GET("/GetBusinessCategory/:key", controller.GetBusinessCategoryController)
		login.POST("/SetBusinessCategory/:key", controller.SetBusinessCategoryController)
		login.POST("/SetNetWorkProxy/:key", controller.SetNetWorkProxyController)
		login.POST("/HasUnsentPreKeys/:key", controller.HasUnsentPreKeysController)
	}
	// 消息
	message := engine.Group(ver + "/message")
	{
		message.GET("/syncMessages/:key", controller.SyncNewMessageController)
		message.POST("/SendTextMessage/:key", controller.SendTextMessageController)
		message.POST("/SendImageMessage/:key", controller.SendImageMessageController)
		message.POST("/SendAudioMessage/:key", controller.SendAudioMessageController)
		message.POST("/SendVideoMessage/:key", controller.SendVideoMessageController)
		message.POST("/SendVcardMessage/:key", controller.SendVcardMessageController)
		message.POST("/SendMessageDownload/:key", controller.SendMessageDownloadController)
	}

	// 同步
	sync := engine.Group(ver + "/sync")
	{
		sync.POST("/SyncContacts/:key", controller.SyncContactController)
		sync.POST("/SyncAddOneContacts/:key", controller.SyncAddOneContactsController)
	}
	// 个人中心
	profile := engine.Group(ver + "/profile")
	{
		profile.POST("/SetNickName/:key", controller.SetNickNameController)
		profile.POST("/SetPicture/:key", controller.SetProfilePictureController)
		profile.POST("/GetPicture/:key", controller.GetProfilePictureController)
		profile.POST("/GetPreview/:key", controller.GetPreviewController)
		profile.POST("/SetState/:key", controller.SetStateController)
		profile.POST("/GetState/:key", controller.GetStateController)
		profile.GET("/GetQr/:key", controller.GetQrController)
		profile.GET("/SetQrRevoke/:key", controller.SetQrRevokeController)
		profile.GET("/GetProfile/:key", controller.GetProfileController)
		profile.POST("/ScanCode/:key", controller.ScanCodeController)
		profile.POST("/TwoVerify/:key", controller.TwoVerifyController)
	}

	// 群
	group := engine.Group(ver + "/group")
	{
		group.POST("/CreateGroup/:key", controller.CreateGroupController)
		group.POST("/AddGroupMember/:key", controller.AddGroupMemberController)
		group.POST("/GetGroupCode/:key", controller.GetGroupCodeController)
		group.POST("/SetGroupAdmin/:key", controller.SetGroupAdminController)
		group.POST("/LogOutGroup/:key", controller.LogOutGroupController)
		group.POST("/GetGroupMember/:key", controller.GetGroupMemberController)
		group.POST("/CreateGroupInvite/:key", controller.CreateGroupInviteController)
	}

	//动态
	sns := engine.Group(ver + "/sns")
	{
		sns.POST("/SnsTextPost/:key", controller.SnsTextPostController)
	}
	// task
	task := engine.Group(ver + "/task")
	{
		task.POST("/AddTask/:key", controller.AddTaskController)
	}
	//扫号
	number := engine.Group(ver + "/number")
	{
		number.POST("/scanNumber/:key", controller.ScanNumberController)
		number.POST("/existence/:key", controller.ExistenceController)
	}
}
