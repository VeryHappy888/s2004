package service

import (
	"fmt"
	"ws-go/api/dto"
	"ws-go/api/tasks"
	"ws-go/api/vo"
	"ws-go/protocol/app"
)

//AddTaskService
func AddTaskService(k string, dto dto.TaskDto) vo.Resp {
	ws, isExist := GetWSApp(k)
	if !isExist {
		if ws == nil {
			return vo.AnErrorOccurred(fmt.Errorf("账号%s不在线,请重新登录", k))
		}
		return vo.FailedStatue(ws.GetLoginStatus(), ws.GetLoginStatus().String())
	}
	// parameter
	if isEmpty(dto.TaskName) {
		return vo.ParameterError("taskName", "需要执行的任务名")
	}
	var resp vo.Resp
	switch dto.TaskName {
	case tasks.TaskNameMassTask:
		resp = massMessageTaskService(ws, dto)
	}
	return resp
}

// massMessageTaskService
func massMessageTaskService(ws *app.WaApp, taskDto dto.TaskDto) vo.Resp {
	if len(taskDto.Numbers) == 0 || isEmpty(taskDto.Content) {
		return vo.ParameterError("Numbers Or Content ", tasks.TaskNameMassTask+"完整参数")
	}
	// create task
	task := tasks.NewMassMessageTask(ws, taskDto.Numbers, taskDto.Content, taskDto.RandomWait)
	err := task.Worker()
	if err != nil {
		return vo.AnErrorOccurred(err)
	}
	return vo.Success(nil, ws.GetPlatform(), "任务开始执行")
}
