package define

type MsgStatus int

const (
	Sent MsgStatus = iota // 发送
	Ack  MsgStatus = 1    // 确认送达
	Read MsgStatus = 2    // 已读
)
