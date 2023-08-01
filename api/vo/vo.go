package vo

import (
	"fmt"
)

const (
	SysCode                  = -1000
	IncompleteParametersCode = -1
	ParameterErrorCode       = -2
	FailedCreateCode         = -3
	AnErrorOccurredCode      = -4
	SubmitDataErrorCode      = -5
)

type Resp struct {

	//自定义提示状态码
	Code int

	//数据展示
	Data interface{}

	Platform string

	//提示文本
	Msg interface{}
}

// Success 向前端返回成功
func Success(data interface{}, platform, m string) Resp {
	return Resp{
		Code:     0,
		Data:     data,
		Platform: platform,
		Msg:      m,
	}
}

// Success 向前端返回成功
func SuccessJson(data interface{}, platform, m string) Resp {
	return Resp{
		Code:     0,
		Data:     data,
		Platform: platform,
		Msg:      m,
	}
}

func AnErrorOccurred(err error) Resp {
	return Resp{
		Code: AnErrorOccurredCode,
		Data: nil,
		Msg:  "发生错误:" + err.Error(),
	}
}

func SubmitDataError() Resp {
	return Resp{Code: SubmitDataErrorCode, Data: nil, Msg: "提交数据错误"}
}

// FailedCreate 创建失败
func FailedCreate(msg string) Resp {
	return Resp{Code: FailedCreateCode, Data: nil, Msg: msg}
}

func FailedStatue(data interface{}, msg string) Resp {
	return Resp{Code: FailedCreateCode, Data: data, Msg: msg}
}

// ParameterError 参数错误
func ParameterError(name, msg string) Resp {
	return Resp{Code: ParameterErrorCode, Data: nil, Msg: fmt.Sprintf("参数: %s %s", name, msg)}
}

func IncompleteSys(data string) Resp {
	return Resp{Code: SysCode, Data: data, Msg: "系统错误,请联系管理员!"}
}

// IncompleteParameters 参数不完整
func IncompleteParameters() Resp {
	return Resp{Code: IncompleteParametersCode, Data: nil, Msg: "参数不完整"}
}
