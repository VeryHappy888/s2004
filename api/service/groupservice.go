package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-go/api/dto"
	"ws-go/api/vo"
	"ws-go/protocol/entity"
)

// AddGroupMemberService 添加成员
func AddGroupMemberService(k string, groupDto dto.GroupDto) vo.Resp {
	// check parameters
	if isEmpty(groupDto.GroupId) || len(groupDto.Participants) <= 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	// add group member
	promise := app.AddGroupMember(groupDto.GroupId, groupDto.Participants...)
	result, err := promise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	addGroupResult := iqResult.GetAddGroupResult()
	members := make([]gin.H, 0)
	for s, attr := range addGroupResult.Members() {
		members = append(members, gin.H{
			"jid":  s,
			"attr": attr,
		})
	}
	return vo.Success(gin.H{
		"status":  200,
		"msg":     "ok",
		"groupId": addGroupResult.GroupId(),
		"members": members,
	}, app.GetPlatform(), "ok")
}

//获取群成员
func GetGroupMemberService(k string, dto dto.GroupCodeDto) vo.Resp {
	// check parameters
	if isEmpty(dto.GroupId) || len(dto.GroupId) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	promise := app.GetGroupMember(dto.GroupId)
	result, err := promise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	groupInfo := iqResult.GetGroupInfo()
	if groupInfo == nil {
		return vo.IncompleteSys("失败")
	}
	participants := make([]gin.H, 0)
	for jid, participant := range groupInfo.Participants() {
		participants = append(participants, gin.H{
			"jid":  jid,
			"attr": participant,
		})
	}
	return vo.Success(gin.H{
		"Subject":      groupInfo.Subject(),
		"Id":           groupInfo.Id(),
		"Creation":     groupInfo.Creation(),
		"Creator":      groupInfo.Creator(),
		"so":           groupInfo.SO(),
		"st":           groupInfo.ST(),
		"count":        len(participants),
		"Participants": participants,
		"status":       200,
		"msg":          "ok",
	}, app.GetPlatform(), "ok")
}

// CreateGroupService 创建群聊
func CreateGroupService(k string, dto dto.GroupDto) vo.Resp {
	// check parameters
	if isEmpty(dto.Subject) || len(dto.Participants) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	// create group
	promise := app.CreateGroup(dto.Subject, dto.Participants)
	result, err := promise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// iq result
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	groupInfo := iqResult.GetGroupInfo()
	participants := make([]gin.H, 0)
	for jid, participant := range groupInfo.Participants() {
		participants = append(participants, gin.H{
			"jid":  jid,
			"attr": participant,
		})
	}
	return vo.Resp{Code: 0, Data: gin.H{
		"status":       200,
		"msg":          "ok",
		"Subject":      groupInfo.Subject(),
		"Id":           groupInfo.Id(),
		"Creation":     groupInfo.Creation(),
		"Creator":      groupInfo.Creator(),
		"Participants": participants,
	}}
}

// GetGroupCodeService 获取群二维码
func GetGroupCodeService(k string, dto dto.GroupCodeDto) vo.Resp {
	// check parameters
	if isEmpty(dto.GroupId) || len(dto.GroupId) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	promise := app.GetGroupCode(dto.GroupId)
	result, err := promise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// iq result
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	if iqResult.GetInviteCode() == "" {
		return vo.AnErrorOccurred(err)
	}
	url := fmt.Sprintf("https://chat.whatsapp.com/%s", iqResult.GetInviteCode())
	return vo.Success(gin.H{
		"data":   url,
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "成功")
}

//SetGroupDescService 设置描述
func SetGroupDescService(k string, dto dto.GroupDescDto) vo.Resp {
	// check parameters
	if isEmpty(dto.GroupId) || len(dto.GroupId) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	app.SetGroupDesc(dto.GroupId, dto.Desc)
	return vo.Success("", app.GetPlatform(), "")
}

// SetGroupAdminService 设置群管理
func SetGroupAdminService(k string, dto dto.GroupAdminDto) vo.Resp {
	// check parameters
	if isEmpty(dto.GroupId) || len(dto.GroupId) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	promise := app.CreateGroupAdmin(dto.GroupId, dto.Opcode, dto.ToWid)
	result, err := promise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// iq result
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "成功")
}

// CreateGroupInviteService 通过code进群
func CreateGroupInviteService(k, code string) vo.Resp {
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	inviteRsp := app.InviteCode(code, "g.us")
	result, err := inviteRsp.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	qr := iqResult.GetGroupInfo()
	if inviteRsp == nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(gin.H{
		"GroupId": qr.GroupId(),
		"status":  200,
		"msg":     "ok",
	}, app.GetPlatform(), "ok")
}

// SendLogOutGroupService 退出群组
func SendLogOutGroupService(k string, dto dto.GroupCodeDto) vo.Resp {
	// check parameters
	if isEmpty(dto.GroupId) || len(dto.GroupId) == 0 {
		return vo.IncompleteParameters()
	}
	// get app
	app, isExist := GetWSApp(k)
	if !isExist {
		if app == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(app.GetLoginStatus(), app.GetLoginStatus().String())
	}
	promise := app.SendLogOutGroup(dto.GroupId)
	result, err := promise.GetResult()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	// iq result
	iqResult := result.(entity.IqResult)
	if iqResult.GetErrorEntityResult() != nil {
		return vo.Success(gin.H{
			"status": iqResult.GetErrorEntityResult().Code(),
			"msg":    iqResult.GetErrorEntityResult().Text(),
		}, app.GetPlatform(), "no")
	}
	return vo.Success(gin.H{
		"status": 200,
		"msg":    "ok",
	}, app.GetPlatform(), "成功")
}
