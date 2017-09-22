// Code generated by protoc-gen-gogo.
// source: subpkg/subproto.proto
// DO NOT EDIT!

/*
Package subpkg is a generated protocol buffer package.

It is generated from these files:
	subpkg/subproto.proto

It has these top-level messages:
	SubObject
*/
package subpkg

import "github.com/gogo/protobuf/proto"
import "fmt"
import "math"
import _ "github.com/gogo/protobuf/gogoproto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type SubObject struct {
	XXX_unrecognized []byte `json:"-"`
}

func (m *SubObject) Reset()                    { *m = SubObject{} }
func (m *SubObject) String() string            { return proto.CompactTextString(m) }
func (*SubObject) ProtoMessage()               {}
func (*SubObject) Descriptor() ([]byte, []int) { return fileDescriptorSubproto, []int{0} }

func init() {
	proto.RegisterType((*SubObject)(nil), "subpkg.SubObject")
}

func init() { proto.RegisterFile("subpkg/subproto.proto", fileDescriptorSubproto) }

var fileDescriptorSubproto = []byte{
	// 88 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x2d, 0x2e, 0x4d, 0x2a,
	0xc8, 0x4e, 0xd7, 0x07, 0x51, 0x45, 0xf9, 0x25, 0xf9, 0x7a, 0x60, 0x52, 0x88, 0x0d, 0x22, 0x2c,
	0xa5, 0x9b, 0x9e, 0x59, 0x92, 0x51, 0x9a, 0xa4, 0x97, 0x9c, 0x9f, 0xab, 0x9f, 0x9e, 0x9f, 0x9e,
	0xaf, 0x0f, 0x96, 0x4e, 0x2a, 0x4d, 0x03, 0xf3, 0xc0, 0x1c, 0x30, 0x0b, 0xa2, 0x4d, 0x89, 0x9b,
	0x8b, 0x33, 0xb8, 0x34, 0xc9, 0x3f, 0x29, 0x2b, 0x35, 0xb9, 0x04, 0x10, 0x00, 0x00, 0xff, 0xff,
	0x4e, 0x38, 0xf3, 0x28, 0x5b, 0x00, 0x00, 0x00,
}
