package dto

import "ws-go/protocol/waproto"

type AuthDataDto struct {
	*waproto.ClientPayload
}

// LoginDto 登录信息
type LoginDto struct {
	// socks 5 代理
	Socks5   string
	AuthBody *AuthDataDto
	// AuthHexData 握手认证数据
	AuthHexData string
	// StaticPriKey 认证私钥
	StaticPriKey string
	// StaticPubKey 认证公钥
	StaticPubKey string
	//注册返回
	ClientStaticKeypair string
	//EdgeRouting
	EdgeRouting string
	//
	IdentityPubKey string
	IdentityPriKey string
	Hash           string
}

// 注册
type SendVerifyCodeDto struct {
	Cc     int32  //区号
	Phone  string //电话
	Method int    // 默认0为短信验证码 1为语音验证码
	Lc     string //国家
	Lg     string //语言
	Socks5 string // socks 5 代理
}

// 注册验证
type SendRegisterVerifyDto struct {
	//区号
	Cc int32
	//电话
	Phone string
	// socks 5 代理
	Socks5 string
	//code
	Code string
	Lc   string //国家
	Lg   string //语言
}

// MessageDto 消息
type MessageDto struct {
	// RecipientId 接收者
	RecipientId string
	// Content 发送内容
	Content string
	// Subscribe 发送消息前发送
	Subscribe bool
	// SentGroup 发送到群组
	SentGroup bool
	// at
	At []string
	//引用消息id
	StanzaId string
	//引用谁的话
	Participant string
	//引用的消息内容
	Conversation string
	// 聊天状态
	ChatState bool
}

type MessageImageDto struct {
	ImageBase64 string
	// RecipientId 接收者
	RecipientId string
	// Subscribe 发送消息前发送
	Subscribe bool
	// SentGroup 发送到群组
	SentGroup bool
}
type MessageAudioDto struct {
	AudioBase64 string
	// RecipientId 接收者
	RecipientId string
	// Subscribe 发送消息前发送
	Subscribe bool
	// SentGroup 发送到群组
	SentGroup bool
}
type MessageVideoDto struct {
	//预览图
	ThumbnailBase64 string
	VideoBase64     string
	// RecipientId 接收者
	RecipientId string
	// Subscribe 发送消息前发送
	Subscribe bool
	// SentGroup 发送到群组
	SentGroup bool
}

type VcardDto struct {
	// RecipientId 接收者
	RecipientId string
	//推荐电话（格式：+60 10-890 8990）
	Tel string
	//卡片名称
	VcardName string
	// Subscribe 发送消息前发送
	Subscribe bool
	// SentGroup 发送到群组
	SentGroup bool
}

/*
*
下载图片
*/
type DownloadMessageDto struct {
	/**
	消息里返回后缀为.enc的url
	*/
	Url string
	/***
	消息里返回的MediaKey
	*/
	MediaKey string
	/**
	消息里返回的长度
	*/
	FileLength int

	/**
	消息类型：MediaImage /MediaVideo /MediaAudio /MediaDocument
	*/
	MediaType string
}

type PictureInfoDto struct {
	Picture []byte
	From    string
}
type NickNameDto struct {
	Name string
}
type SetBusinessCategoryDto struct {
	Name       string
	CategoryId string
}
type SetNetWorkProxyDto struct {
	Socks5 string
}
type ScanCodeDto struct {
	Code   string
	OpCode int32
}

type SetStateDto struct {
	Content string
}
type GetStateDto struct {
	ToWid string
}
type TwoVerifyDto struct {
	Code  string
	Email string
}

// SyncContactDto
type SyncContactDto struct {
	Numbers []string
}
type SyncAddOneContactsDto struct {
	Numbers string
}

// GroupDto
type GroupDto struct {
	// GroupId 群id
	GroupId string
	// Subject 群名称
	Subject string
	// Participants 群成员
	Participants []string
}

// GroupCodeDto
type GroupCodeDto struct {
	GroupId string
}
type GroupDescDto struct {
	GroupId string
	Desc    string
}

// GroupAdminDto 群管理
type GroupAdminDto struct {
	GroupId string
	Opcode  int32
	ToWid   string
}

type TaskDto struct {
	TaskName   string
	Content    string
	Numbers    []string
	RandomWait bool
}

type SnsTextDto struct {
	Text string
	// Participants 可见成员
	Participants []string
}
type ScanNumberDto struct {
	Number string
}
type ExistenceDto struct {
	Number []string
}
