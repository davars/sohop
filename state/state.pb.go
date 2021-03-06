// Code generated by protoc-gen-go.
// source: state.proto
// DO NOT EDIT!

/*
Package state is a generated protocol buffer package.

It is generated from these files:
	state.proto

It has these top-level messages:
	TimeBox
	OAuthState
	Session
*/
package state

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// TimeBox wraps byte string payload with an expiration date
type TimeBox struct {
	NotAfter *google_protobuf.Timestamp `protobuf:"bytes,1,opt,name=not_after,json=notAfter" json:"not_after,omitempty"`
	Payload  []byte                     `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (m *TimeBox) Reset()                    { *m = TimeBox{} }
func (m *TimeBox) String() string            { return proto.CompactTextString(m) }
func (*TimeBox) ProtoMessage()               {}
func (*TimeBox) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *TimeBox) GetNotAfter() *google_protobuf.Timestamp {
	if m != nil {
		return m.NotAfter
	}
	return nil
}

func (m *TimeBox) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

// OAuthState contains data associated with a single oauth flow (currently just the url to redirect the user to after
// authentication completes)
type OAuthState struct {
	RedirectUrl string `protobuf:"bytes,1,opt,name=redirect_url,json=redirectUrl" json:"redirect_url,omitempty"`
}

func (m *OAuthState) Reset()                    { *m = OAuthState{} }
func (m *OAuthState) String() string            { return proto.CompactTextString(m) }
func (*OAuthState) ProtoMessage()               {}
func (*OAuthState) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *OAuthState) GetRedirectUrl() string {
	if m != nil {
		return m.RedirectUrl
	}
	return ""
}

// Session contains data associated with a single user: who that user is and whether they're authenticated & authorized
type Session struct {
	User       string                     `protobuf:"bytes,1,opt,name=user" json:"user,omitempty"`
	ExpiresAt  *google_protobuf.Timestamp `protobuf:"bytes,2,opt,name=expires_at,json=expiresAt" json:"expires_at,omitempty"`
	Authorized bool                       `protobuf:"varint,3,opt,name=authorized" json:"authorized,omitempty"`
}

func (m *Session) Reset()                    { *m = Session{} }
func (m *Session) String() string            { return proto.CompactTextString(m) }
func (*Session) ProtoMessage()               {}
func (*Session) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *Session) GetUser() string {
	if m != nil {
		return m.User
	}
	return ""
}

func (m *Session) GetExpiresAt() *google_protobuf.Timestamp {
	if m != nil {
		return m.ExpiresAt
	}
	return nil
}

func (m *Session) GetAuthorized() bool {
	if m != nil {
		return m.Authorized
	}
	return false
}

func init() {
	proto.RegisterType((*TimeBox)(nil), "state.TimeBox")
	proto.RegisterType((*OAuthState)(nil), "state.OAuthState")
	proto.RegisterType((*Session)(nil), "state.Session")
}

func init() { proto.RegisterFile("state.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 237 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x8e, 0x4f, 0x6b, 0x83, 0x40,
	0x10, 0xc5, 0xb1, 0xff, 0x8c, 0x63, 0x4e, 0x7b, 0x92, 0x1c, 0x5a, 0xeb, 0xc9, 0x93, 0x42, 0x7b,
	0x28, 0x3d, 0xda, 0x2f, 0x50, 0xd8, 0xa4, 0xb7, 0x82, 0x6c, 0xea, 0x24, 0x59, 0x50, 0x47, 0x76,
	0x67, 0xc1, 0xf6, 0xd3, 0x97, 0x6c, 0x5c, 0xe8, 0xa9, 0xb7, 0x99, 0xc7, 0xfb, 0xf1, 0x7e, 0x90,
	0x5a, 0x56, 0x8c, 0xd5, 0x64, 0x88, 0x49, 0xdc, 0xfa, 0x67, 0xf3, 0x70, 0x24, 0x3a, 0xf6, 0x58,
	0xfb, 0x70, 0xef, 0x0e, 0x35, 0xeb, 0x01, 0x2d, 0xab, 0x61, 0xba, 0xf4, 0x8a, 0x4f, 0x88, 0x77,
	0x7a, 0xc0, 0x37, 0x9a, 0xc5, 0x0b, 0x24, 0x23, 0x71, 0xab, 0x0e, 0x8c, 0x26, 0x8b, 0xf2, 0xa8,
	0x4c, 0x9f, 0x36, 0xd5, 0x85, 0xaf, 0x02, 0x5f, 0xed, 0x02, 0x2f, 0x57, 0x23, 0x71, 0x73, 0xee,
	0x8a, 0x0c, 0xe2, 0x49, 0x7d, 0xf7, 0xa4, 0xba, 0xec, 0x2a, 0x8f, 0xca, 0xb5, 0x0c, 0x6f, 0x51,
	0x03, 0xbc, 0x37, 0x8e, 0x4f, 0xdb, 0xb3, 0x8c, 0x78, 0x84, 0xb5, 0xc1, 0x4e, 0x1b, 0xfc, 0xe2,
	0xd6, 0x99, 0xde, 0x6f, 0x24, 0x32, 0x0d, 0xd9, 0x87, 0xe9, 0x8b, 0x19, 0xe2, 0x2d, 0x5a, 0xab,
	0x69, 0x14, 0x02, 0x6e, 0x9c, 0x5d, 0x4c, 0x12, 0xe9, 0x6f, 0xf1, 0x0a, 0x80, 0xf3, 0xa4, 0x0d,
	0xda, 0x56, 0xb1, 0x1f, 0xfb, 0xdf, 0x31, 0x59, 0xda, 0x0d, 0x8b, 0x7b, 0x00, 0xe5, 0xf8, 0x44,
	0x46, 0xff, 0x60, 0x97, 0x5d, 0xe7, 0x51, 0xb9, 0x92, 0x7f, 0x92, 0xfd, 0x9d, 0xc7, 0x9f, 0x7f,
	0x03, 0x00, 0x00, 0xff, 0xff, 0x01, 0x95, 0x8c, 0xac, 0x46, 0x01, 0x00, 0x00,
}
