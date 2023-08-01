package db

type TypeEnum int

const (
	//系统消息
	System TypeEnum = 1000
	//业务消息
	Msg TypeEnum = 2000
	//业务消息
	Reads TypeEnum = 4000
	//状态消息
	Status TypeEnum = 3000
)

func (p TypeEnum) Number() int {
	switch p {
	case System:
		return 1000
	case Msg:
		return 2000
	case Status:
		return 3000
	case Reads:
		return 4000
	default:
		return -1
	}
}
