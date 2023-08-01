package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/frame/g"
	"io"
	"log"
	"net/http"
	"ws-go/api/errors"
	//_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"ws-go/api/router"
	"ws-go/api/vo"
	"ws-go/protocol/app"
	"ws-go/wslog"
)

func init() {
	g.Cfg().SetFileName("config.toml")
	wslog.InitWsLogger()
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
	}
}

func TLog() {
	dir, _ := filepath.Abs(filepath.Dir(""))
	logFileNmae := time.Now().Format("20060102") + ".log"
	logFileAllPath := dir + "/log/" + logFileNmae
	_, err := os.Stat(logFileAllPath)
	var f *os.File
	if err != nil {
		f, _ = os.Create(logFileAllPath)
	} else {
		//如果存在文件则 追加log
		f, _ = os.OpenFile(logFileAllPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	}
	//os.Stdout 目标日志只输出到控制台
	gin.DefaultWriter = io.MultiWriter(f)
	gin.DefaultErrorWriter = io.MultiWriter(f)
	log.SetOutput(io.MultiWriter(f))
}

const version = "1.2"

func main() {
	//分析
	go func() {
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()
	//TLog()
	gin.SetMode(gin.ReleaseMode)
	appService := router.SetUpRouter(func(engine *gin.Engine) {
		//中间件
		engine.Use(Cors())
		//授权系统
		//engine.Use(service.BasicAuth())
		//中间件需要再创建接口之前完成
		engine.GET("/GetVersion", func(context *gin.Context) {
			context.JSON(http.StatusOK, vo.Success(version, version, "获取版本号"))
		})
		//异常处理防止程序奔溃
		engine.Use(errors.Recover)
	}, false)
	fmt.Println("启动GIN服务成功！", 8001)
	s := &http.Server{
		Addr:           ":8001",
		Handler:        appService,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	app.ServerStart()
	err := s.ListenAndServe()
	if err != nil {
		fmt.Println("启动出现了error", err.Error())
		panic(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	<-quit
}
