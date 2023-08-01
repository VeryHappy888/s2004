// Code generated by protoc-gen-go. DO NOT EDIT.
// source: wamessage.proto

package waproto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type WAMessage struct {
	// 文本消息
	CONVERSATION         *string                `protobuf:"bytes,1,opt,name=CONVERSATION" json:"CONVERSATION,omitempty"`
	SKMSG                *SenderKeyGroupMessage `protobuf:"bytes,2,opt,name=SKMSG" json:"SKMSG,omitempty"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *WAMessage) Reset()         { *m = WAMessage{} }
func (m *WAMessage) String() string { return proto.CompactTextString(m) }
func (*WAMessage) ProtoMessage()    {}
func (*WAMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_51c117c53be65f6b, []int{0}
}

func (m *WAMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_WAMessage.Unmarshal(m, b)
}
func (m *WAMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_WAMessage.Marshal(b, m, deterministic)
}
func (m *WAMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_WAMessage.Merge(m, src)
}
func (m *WAMessage) XXX_Size() int {
	return xxx_messageInfo_WAMessage.Size(m)
}
func (m *WAMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_WAMessage.DiscardUnknown(m)
}

var xxx_messageInfo_WAMessage proto.InternalMessageInfo

func (m *WAMessage) GetCONVERSATION() string {
	if m != nil && m.CONVERSATION != nil {
		return *m.CONVERSATION
	}
	return ""
}

func (m *WAMessage) GetSKMSG() *SenderKeyGroupMessage {
	if m != nil {
		return m.SKMSG
	}
	return nil
}

type SenderKeyGroupMessage struct {
	GROUP_ID             *string  `protobuf:"bytes,1,opt,name=GROUP_ID,json=GROUPID" json:"GROUP_ID,omitempty"`
	SENDER_KEY           []byte   `protobuf:"bytes,2,opt,name=SENDER_KEY,json=SENDERKEY" json:"SENDER_KEY,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SenderKeyGroupMessage) Reset()         { *m = SenderKeyGroupMessage{} }
func (m *SenderKeyGroupMessage) String() string { return proto.CompactTextString(m) }
func (*SenderKeyGroupMessage) ProtoMessage()    {}
func (*SenderKeyGroupMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_51c117c53be65f6b, []int{1}
}

func (m *SenderKeyGroupMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SenderKeyGroupMessage.Unmarshal(m, b)
}
func (m *SenderKeyGroupMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SenderKeyGroupMessage.Marshal(b, m, deterministic)
}
func (m *SenderKeyGroupMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SenderKeyGroupMessage.Merge(m, src)
}
func (m *SenderKeyGroupMessage) XXX_Size() int {
	return xxx_messageInfo_SenderKeyGroupMessage.Size(m)
}
func (m *SenderKeyGroupMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_SenderKeyGroupMessage.DiscardUnknown(m)
}

var xxx_messageInfo_SenderKeyGroupMessage proto.InternalMessageInfo

func (m *SenderKeyGroupMessage) GetGROUP_ID() string {
	if m != nil && m.GROUP_ID != nil {
		return *m.GROUP_ID
	}
	return ""
}

func (m *SenderKeyGroupMessage) GetSENDER_KEY() []byte {
	if m != nil {
		return m.SENDER_KEY
	}
	return nil
}

func init() {
	proto.RegisterType((*WAMessage)(nil), "WAMessage")
	proto.RegisterType((*SenderKeyGroupMessage)(nil), "SenderKeyGroupMessage")
}

func init() { proto.RegisterFile("wamessage.proto", fileDescriptor_51c117c53be65f6b) }

var fileDescriptor_51c117c53be65f6b = []byte{
	// 161 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2f, 0x4f, 0xcc, 0x4d,
	0x2d, 0x2e, 0x4e, 0x4c, 0x4f, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x57, 0x8a, 0xe5, 0xe2, 0x0c,
	0x77, 0xf4, 0x85, 0x08, 0x09, 0x29, 0x71, 0xf1, 0x38, 0xfb, 0xfb, 0x85, 0xb9, 0x06, 0x05, 0x3b,
	0x86, 0x78, 0xfa, 0xfb, 0x49, 0x30, 0x2a, 0x30, 0x6a, 0x70, 0x06, 0xa1, 0x88, 0x09, 0xe9, 0x70,
	0xb1, 0x06, 0x7b, 0xfb, 0x06, 0xbb, 0x4b, 0x30, 0x29, 0x30, 0x6a, 0x70, 0x1b, 0x89, 0xe9, 0x05,
	0xa7, 0xe6, 0xa5, 0xa4, 0x16, 0x79, 0xa7, 0x56, 0xba, 0x17, 0xe5, 0x97, 0x16, 0x40, 0x8d, 0x0a,
	0x82, 0x28, 0x52, 0x0a, 0xe4, 0x12, 0xc5, 0x2a, 0x2f, 0x24, 0xc9, 0xc5, 0xe1, 0x1e, 0xe4, 0x1f,
	0x1a, 0x10, 0xef, 0xe9, 0x02, 0xb5, 0x86, 0x1d, 0xcc, 0xf7, 0x74, 0x11, 0x92, 0xe5, 0xe2, 0x0a,
	0x76, 0xf5, 0x73, 0x71, 0x0d, 0x8a, 0xf7, 0x76, 0x8d, 0x04, 0x5b, 0xc3, 0x13, 0xc4, 0x09, 0x11,
	0xf1, 0x76, 0x8d, 0x04, 0x04, 0x00, 0x00, 0xff, 0xff, 0x65, 0x72, 0x27, 0xb9, 0xc3, 0x00, 0x00,
	0x00,
}