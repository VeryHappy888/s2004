package wslog

import (
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/glog"
)

var wslog = glog.New()

func InitWsLogger() {
	wslog = g.Log()
	wslog.SetLevel(glog.LEVEL_INFO | glog.LEVEL_ERRO)
	wslog.SetFlags(glog.F_TIME_STD | glog.F_FILE_SHORT)
	//wslog.SetStdoutPrint(false)
}

// GetLogger
func GetLogger() *glog.Logger {
	return wslog
}

// Println
func Println(v ...interface{}) {
	wslog.Skip(1).Println(v...)
}

func Error(err error) {
	wslog.Skip(1).Error(err)
}

// Info
func Info(v ...interface{}) {
	wslog.Skip(1).Info(v...)
}
