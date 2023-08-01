package tasks

import (
	"github.com/gogf/gf/container/glist"
	"github.com/gogf/gf/os/grpool"
)

type TaskStatus int

const (
	TaskRun TaskStatus = iota
	TaskStop
	TaskComplete
	TaskError
)

// ITask task interface
type ITask interface {
	Stop()
	Worker() error
	Result() (interface{}, error)
}

// TaskBase base
type TaskBase struct {
	ITask
	resultError error
	status      TaskStatus
}

// Manager tasks manage
type Manager struct {
	taskList *glist.List
	taskPool *grpool.Pool
}

func NewManager() *Manager {
	return &Manager{
		taskList: glist.New(true),
		taskPool: grpool.New(10),
	}
}

// AddTask `
func (m *Manager) AddTask(task ITask) error {
	return m.taskPool.Add(func() {
		_ = task.Worker()
	})
}
