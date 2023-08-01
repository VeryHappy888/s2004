package entity

import (
	"errors"
	"fmt"
	"io"
	"ws-go/protocol/waproto"
)

func ParseProtoMessage(msg *waproto.Message) interface{} {
	switch {
	case msg.GetAudioMessage() != nil:
		return getAudioMessage(msg)
	case msg.GetImageMessage() != nil:
		return getImageMessage(msg)

	case msg.GetVideoMessage() != nil:
		return getVideoMessage(msg)

	case msg.GetDocumentMessage() != nil:
		return getDocumentMessage(msg)

	case msg.GetConversation() != "":
		return getTextMessage(msg)

	case msg.GetExtendedTextMessage() != nil:
		return getTextMessage(msg)

	case msg.GetLocationMessage() != nil:
		return GetLocationMessage(msg)

	case msg.GetLiveLocationMessage() != nil:
		return GetLiveLocationMessage(msg)

	case msg.GetStickerMessage() != nil:
		return getStickerMessage(msg)

	case msg.GetContactMessage() != nil:
		return getContactMessage(msg)
	default:
		//cannot match message
		fmt.Println(msg)
		return errors.New("message type not implemented")
	}
}

func getContactMessage(msg *waproto.Message) ContactMessage {
	contact := msg.GetContactMessage()
	contactMessage := ContactMessage{
		//Info: getMessageInfo(msg),
		DisplayName: contact.GetDisplayName(),
		Vcard:       contact.GetVcard(),

		//ContextInfo: getMessageContext(contact.GetContextInfo()),
	}

	return contactMessage
}

/*
ContactMessage表示联系人消息。
*/
type ContactMessage struct {
	Info MessageInfo

	DisplayName string
	Vcard       string

	ContextInfo ContextInfo
}

func getStickerMessage(msg *waproto.Message) StickerMessage {
	sticker := msg.GetStickerMessage()
	stickerMessage := StickerMessage{
		//Info:          getMessageInfo(msg),
		url:           sticker.GetUrl(),
		mediaKey:      sticker.GetMediaKey(),
		Type:          sticker.GetMimetype(),
		fileEncSha256: sticker.GetFileEncSha256(),
		fileSha256:    sticker.GetFileSha256(),
		fileLength:    sticker.GetFileLength(),
		//ContextInfo:   getMessageContext(sticker.GetContextInfo()),
	}

	return stickerMessage
}

/*
StickerMessage represents a sticker message.
*/
type StickerMessage struct {
	Info          MessageInfo
	Type          string
	Content       io.Reader
	url           string
	mediaKey      []byte
	fileEncSha256 []byte
	fileSha256    []byte
	fileLength    uint64

	ContextInfo ContextInfo
}

func GetLiveLocationMessage(msg *waproto.Message) LiveLocationMessage {
	loc := msg.GetLiveLocationMessage()
	liveLocationMessage := LiveLocationMessage{
		//Info:                              getMessageInfo(msg),
		DegreesLatitude:                   loc.GetDegreesLatitude(),
		DegreesLongitude:                  loc.GetDegreesLongitude(),
		AccuracyInMeters:                  loc.GetAccuracyInMeters(),
		SpeedInMps:                        loc.GetSpeedInMps(),
		DegreesClockwiseFromMagneticNorth: loc.GetDegreesClockwiseFromMagneticNorth(),
		Caption:                           loc.GetCaption(),
		SequenceNumber:                    loc.GetSequenceNumber(),
		JpegThumbnail:                     loc.GetJpegThumbnail(),
		//ContextInfo:                       getMessageContext(loc.GetContextInfo()),
	}

	return liveLocationMessage
}

/*
LiveLocationMessage represents a live location message
*/
type LiveLocationMessage struct {
	Info                              MessageInfo
	DegreesLatitude                   float64
	DegreesLongitude                  float64
	AccuracyInMeters                  uint32
	SpeedInMps                        float32
	DegreesClockwiseFromMagneticNorth uint32
	Caption                           string
	SequenceNumber                    int64
	JpegThumbnail                     []byte
	ContextInfo                       ContextInfo
}

func GetLocationMessage(msg *waproto.Message) LocationMessage {
	loc := msg.GetLocationMessage()
	locationMessage := LocationMessage{
		//Info:             getMessageInfo(msg),
		DegreesLatitude:  loc.GetDegreesLatitude(),
		DegreesLongitude: loc.GetDegreesLongitude(),
		Name:             loc.GetName(),
		Address:          loc.GetAddress(),
		Url:              loc.GetUrl(),
		JpegThumbnail:    loc.GetJpegThumbnail(),
		//ContextInfo:      getMessageContext(loc.GetContextInfo()),
	}

	return locationMessage
}

/*
LocationMessage represents a location message
*/
type LocationMessage struct {
	Info             MessageInfo
	DegreesLatitude  float64
	DegreesLongitude float64
	Name             string
	Address          string
	Url              string
	JpegThumbnail    []byte
	ContextInfo      ContextInfo
}

func getTextMessage(msg *waproto.Message) TextMessage {
	text := TextMessage{Text: msg.GetConversation()}
	if msg.GetConversation() == "" {
		text = TextMessage{Text: msg.ExtendedTextMessage.GetText()}
	}
	return text
}

/*
TextMessage represents a text message.
*/
type TextMessage struct {
	Info        MessageInfo
	Text        string
	ContextInfo ContextInfo
}

func getDocumentMessage(msg *waproto.Message) DocumentMessage {
	doc := msg.GetDocumentMessage()

	documentMessage := DocumentMessage{
		//Info:          getMessageInfo(msg),
		Title:         doc.GetTitle(),
		PageCount:     doc.GetPageCount(),
		Type:          doc.GetMimetype(),
		FileName:      doc.GetFileName(),
		Thumbnail:     doc.GetJpegThumbnail(),
		Url:           doc.GetUrl(),
		MediaKey:      doc.GetMediaKey(),
		FileEncSha256: doc.GetFileEncSha256(),
		FileSha256:    doc.GetFileSha256(),
		FileLength:    doc.GetFileLength(),
		//ContextInfo:   getMessageContext(doc.GetContextInfo()),
	}

	return documentMessage
}

/*
DocumentMessage表示文档消息。媒体上传/下载和媒体下载需要未报告的字段

验证。提供io读卡器作为消息发送的内容。
*/
type DocumentMessage struct {
	Info          MessageInfo
	Title         string
	PageCount     uint32
	Type          string
	FileName      string
	Thumbnail     []byte
	Content       io.Reader
	Url           string
	MediaKey      []byte
	FileEncSha256 []byte
	FileSha256    []byte
	FileLength    uint64
	ContextInfo   ContextInfo
}

func getVideoMessage(msg *waproto.Message) VideoMessage {
	vid := msg.GetVideoMessage()

	videoMessage := VideoMessage{
		//Info:          getMessageInfo(msg),
		Caption:       vid.GetCaption(),
		Thumbnail:     vid.GetJpegThumbnail(),
		GifPlayback:   vid.GetGifPlayback(),
		Url:           vid.GetUrl(),
		MediaKey:      vid.GetMediaKey(),
		Length:        vid.GetSeconds(),
		Type:          vid.GetMimetype(),
		FileEncSha256: vid.GetFileEncSha256(),
		FileSha256:    vid.GetFileSha256(),
		FileLength:    vid.GetFileLength(),
		//ContextInfo:   getMessageContext(vid.GetContextInfo()),
	}

	return videoMessage
}

/*
VideoMessage表示视频消息。媒体启动/下载和媒体验证需要未报告的字段。
提供io读卡器作为消息发送的内容。
*/
type VideoMessage struct {
	Info          MessageInfo
	Caption       string
	Thumbnail     []byte
	Length        uint32
	Type          string
	Content       io.Reader
	GifPlayback   bool
	Url           string
	MediaKey      []byte
	FileEncSha256 []byte
	FileSha256    []byte
	FileLength    uint64
	ContextInfo   ContextInfo
}

func getImageMessage(msg *waproto.Message) ImageMessage {
	image := msg.GetImageMessage()
	imageMessage := ImageMessage{
		//Info:          getMessageInfo(msg),
		Caption:       image.GetCaption(),
		Thumbnail:     image.GetJpegThumbnail(),
		Url:           image.GetUrl(),
		MediaKey:      image.GetMediaKey(),
		Type:          image.GetMimetype(),
		FileEncSha256: image.GetFileEncSha256(),
		FileSha256:    image.GetFileSha256(),
		FileLength:    image.GetFileLength(),
		//ContextInfo:   getMessageContext(image.GetContextInfo()),
	}
	return imageMessage
}

/*
ImageMessage表示图像消息。媒体启动/下载和媒体验证需要未报告的字段。
提供io读卡器作为消息发送的内容。
*/
type ImageMessage struct {
	Info          MessageInfo
	Caption       string
	Thumbnail     []byte
	Type          string
	Content       io.Reader
	Url           string
	MediaKey      []byte
	FileEncSha256 []byte
	FileSha256    []byte
	FileLength    uint64
	ContextInfo   ContextInfo
}

func getAudioMessage(msg *waproto.Message) AudioMessage {
	aud := msg.GetAudioMessage()
	audioMessage := AudioMessage{
		//Info:          getMessageInfo(msg),
		Url:           aud.GetUrl(),
		MediaKey:      aud.GetMediaKey(),
		Length:        aud.GetSeconds(),
		Type:          aud.GetMimetype(),
		FileEncSha256: aud.GetFileEncSha256(),
		FileSha256:    aud.GetFileSha256(),
		FileLength:    aud.GetFileLength(),
		//ContextInfo:   getMessageContext(aud.GetContextInfo()),
	}

	return audioMessage
}

/*
ContextInfo表示每条消息的ContextInfo
*/
type ContextInfo struct {
	QuotedMessageID string //StanzaId
	QuotedMessage   *waproto.Message
	Participant     string
	IsForwarded     bool
	ForwardingScore uint32
}

type MessageStatus int

const (
	Error       MessageStatus = 0
	Pending                   = 1
	ServerAck                 = 2
	DeliveryAck               = 3
	Read                      = 4
	Played                    = 5
)

/*
MessageInfo contains general message information. It is part of every of every message type.
*/
type MessageInfo struct {
	Id        string
	RemoteJid string
	SenderJid string
	FromMe    bool
	Timestamp uint64
	PushName  string
	Status    MessageStatus

	Source *waproto.WebMessageInfo
}

/*
AudioMessage表示音频消息。媒体启动/下载和媒体验证需要未报告的字段。
提供io读卡器作为消息发送的内容
*/
type AudioMessage struct {
	Info          MessageInfo
	Length        uint32
	Type          string
	Content       io.Reader
	Ptt           bool
	Url           string
	MediaKey      []byte
	FileEncSha256 []byte
	FileSha256    []byte
	FileLength    uint64
	ContextInfo   ContextInfo
}
