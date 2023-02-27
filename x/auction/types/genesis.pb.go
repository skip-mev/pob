// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pob/auction/v1/genesis.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// GenesisState defines the genesis state of the x/auction module.
type GenesisState struct {
	Params Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_9ed8651e43f855a1, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

// Params defines the parameters of the x/auction module.
type Params struct {
	// max_bundle_size is the maximum number of transactions that can be bundled
	// in a single bundle.
	MaxBundleSize uint32 `protobuf:"varint,1,opt,name=max_bundle_size,json=maxBundleSize,proto3" json:"max_bundle_size,omitempty"`
	// escrow_account_address is the address of the account that will hold the
	// funds for the auctions.
	EscrowAccountAddress string `protobuf:"bytes,2,opt,name=escrow_account_address,json=escrowAccountAddress,proto3" json:"escrow_account_address,omitempty"`
	// reserve_fee specifies a fee that the bidder must pay to enter the auction.
	ReserveFee github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,3,rep,name=reserve_fee,json=reserveFee,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"reserve_fee"`
	// min_buy_in_fee speficies the bid floor for the auction.
	MinBuyInFee github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,4,rep,name=min_buy_in_fee,json=minBuyInFee,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"min_buy_in_fee"`
	// min_bid_increment specifies the minimum amount that the next bid must be
	// greater than the previous bid.
	MinBidIncrement github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,5,rep,name=min_bid_increment,json=minBidIncrement,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"min_bid_increment"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_9ed8651e43f855a1, []int{1}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetMaxBundleSize() uint32 {
	if m != nil {
		return m.MaxBundleSize
	}
	return 0
}

func (m *Params) GetEscrowAccountAddress() string {
	if m != nil {
		return m.EscrowAccountAddress
	}
	return ""
}

func (m *Params) GetReserveFee() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.ReserveFee
	}
	return nil
}

func (m *Params) GetMinBuyInFee() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.MinBuyInFee
	}
	return nil
}

func (m *Params) GetMinBidIncrement() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.MinBidIncrement
	}
	return nil
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "skipmev.pob.auction.v1.GenesisState")
	proto.RegisterType((*Params)(nil), "skipmev.pob.auction.v1.Params")
}

func init() { proto.RegisterFile("pob/auction/v1/genesis.proto", fileDescriptor_9ed8651e43f855a1) }

var fileDescriptor_9ed8651e43f855a1 = []byte{
	// 450 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x92, 0x31, 0x6f, 0x13, 0x31,
	0x14, 0xc7, 0x73, 0xa4, 0x44, 0xc2, 0xa1, 0x54, 0x3d, 0x55, 0x55, 0x28, 0xc8, 0x8d, 0x3a, 0x40,
	0x54, 0xa9, 0xb6, 0x52, 0x60, 0x41, 0x2c, 0x09, 0x12, 0xa8, 0x12, 0x03, 0x4a, 0x37, 0x16, 0xcb,
	0xf6, 0x3d, 0x82, 0x55, 0x6c, 0x1f, 0x67, 0xdf, 0x91, 0x54, 0x7c, 0x02, 0x26, 0x3e, 0x06, 0x62,
	0xea, 0xc7, 0xe8, 0xd8, 0x91, 0x09, 0x50, 0x32, 0x74, 0xe5, 0x23, 0xa0, 0xb3, 0x0f, 0xe8, 0xd0,
	0x35, 0xcb, 0xdd, 0xe9, 0xfd, 0xff, 0xef, 0xfd, 0xde, 0x3b, 0xfd, 0xd1, 0xfd, 0xdc, 0x0a, 0xca,
	0x4b, 0xe9, 0x95, 0x35, 0xb4, 0x1a, 0xd2, 0x29, 0x18, 0x70, 0xca, 0x91, 0xbc, 0xb0, 0xde, 0xa6,
	0xdb, 0xee, 0x44, 0xe5, 0x1a, 0x2a, 0x92, 0x5b, 0x41, 0x1a, 0x17, 0xa9, 0x86, 0x3b, 0x5b, 0x53,
	0x3b, 0xb5, 0xc1, 0x42, 0xeb, 0xaf, 0xe8, 0xde, 0xc1, 0xd2, 0x3a, 0x6d, 0x1d, 0x15, 0xdc, 0x01,
	0xad, 0x86, 0x02, 0x3c, 0x1f, 0x52, 0x69, 0x95, 0x69, 0xf4, 0x4d, 0xae, 0x95, 0xb1, 0x34, 0x3c,
	0x63, 0x69, 0xef, 0x15, 0xba, 0xfd, 0x32, 0x12, 0x8f, 0x3d, 0xf7, 0x90, 0x3e, 0x43, 0x9d, 0x9c,
	0x17, 0x5c, 0xbb, 0x5e, 0xd2, 0x4f, 0x06, 0xdd, 0x43, 0x4c, 0xae, 0xdf, 0x80, 0xbc, 0x0e, 0xae,
	0xf1, 0xda, 0xf9, 0x8f, 0xdd, 0xd6, 0xa4, 0xe9, 0xd9, 0xfb, 0xdd, 0x46, 0x9d, 0x28, 0xa4, 0x0f,
	0xd0, 0x86, 0xe6, 0x33, 0x26, 0x4a, 0x93, 0xbd, 0x07, 0xe6, 0xd4, 0x29, 0x84, 0x89, 0xeb, 0x93,
	0x75, 0xcd, 0x67, 0xe3, 0x50, 0x3d, 0x56, 0xa7, 0x90, 0x3e, 0x46, 0xdb, 0xe0, 0x64, 0x61, 0x3f,
	0x32, 0x2e, 0xa5, 0x2d, 0x8d, 0x67, 0x3c, 0xcb, 0x0a, 0x70, 0xae, 0x77, 0xa3, 0x9f, 0x0c, 0x6e,
	0x4d, 0xb6, 0xa2, 0x3a, 0x8a, 0xe2, 0x28, 0x6a, 0xe9, 0x07, 0xd4, 0x2d, 0xc0, 0x41, 0x51, 0x01,
	0x7b, 0x0b, 0xd0, 0x6b, 0xf7, 0xdb, 0x83, 0xee, 0xe1, 0x5d, 0x12, 0xef, 0x27, 0xf5, 0xfd, 0xa4,
	0xb9, 0x9f, 0x3c, 0xb7, 0xca, 0x8c, 0x9f, 0xd4, 0x6b, 0x7e, 0xfb, 0xb9, 0x3b, 0x98, 0x2a, 0xff,
	0xae, 0x14, 0x44, 0x5a, 0x4d, 0x9b, 0x9f, 0x15, 0x5f, 0x07, 0x2e, 0x3b, 0xa1, 0x7e, 0x9e, 0x83,
	0x0b, 0x0d, 0xee, 0xeb, 0xe5, 0xd9, 0x7e, 0x32, 0x41, 0x0d, 0xe4, 0x05, 0x40, 0x5a, 0xa2, 0x3b,
	0x5a, 0x19, 0x26, 0xca, 0x39, 0x53, 0x26, 0x50, 0xd7, 0x56, 0x44, 0xed, 0x6a, 0x65, 0xc6, 0xe5,
	0xfc, 0xc8, 0xd4, 0xd8, 0x4f, 0x68, 0x33, 0x60, 0x55, 0xc6, 0x94, 0x91, 0x05, 0x68, 0x30, 0xbe,
	0x77, 0x73, 0x45, 0xe4, 0x8d, 0x9a, 0xac, 0xb2, 0xa3, 0xbf, 0xa0, 0xa7, 0xfd, 0xcf, 0x97, 0x67,
	0xfb, 0xf7, 0xae, 0xb4, 0xcc, 0xfe, 0x65, 0xb5, 0x09, 0xc0, 0xe8, 0x7c, 0x81, 0x93, 0x8b, 0x05,
	0x4e, 0x7e, 0x2d, 0x70, 0xf2, 0x65, 0x89, 0x5b, 0x17, 0x4b, 0xdc, 0xfa, 0xbe, 0xc4, 0xad, 0x37,
	0x0f, 0xaf, 0xb0, 0xeb, 0x10, 0x1d, 0x68, 0xa8, 0x68, 0x9d, 0xf6, 0xff, 0x33, 0xc2, 0x02, 0xa2,
	0x13, 0xa2, 0xf8, 0xe8, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x17, 0x29, 0x90, 0x62, 0x0b, 0x03,
	0x00, 0x00,
}

func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.MinBidIncrement) > 0 {
		for iNdEx := len(m.MinBidIncrement) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.MinBidIncrement[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.MinBuyInFee) > 0 {
		for iNdEx := len(m.MinBuyInFee) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.MinBuyInFee[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.ReserveFee) > 0 {
		for iNdEx := len(m.ReserveFee) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.ReserveFee[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.EscrowAccountAddress) > 0 {
		i -= len(m.EscrowAccountAddress)
		copy(dAtA[i:], m.EscrowAccountAddress)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.EscrowAccountAddress)))
		i--
		dAtA[i] = 0x12
	}
	if m.MaxBundleSize != 0 {
		i = encodeVarintGenesis(dAtA, i, uint64(m.MaxBundleSize))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovGenesis(uint64(l))
	return n
}

func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.MaxBundleSize != 0 {
		n += 1 + sovGenesis(uint64(m.MaxBundleSize))
	}
	l = len(m.EscrowAccountAddress)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	if len(m.ReserveFee) > 0 {
		for _, e := range m.ReserveFee {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.MinBuyInFee) > 0 {
		for _, e := range m.MinBuyInFee {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.MinBidIncrement) > 0 {
		for _, e := range m.MinBidIncrement {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxBundleSize", wireType)
			}
			m.MaxBundleSize = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxBundleSize |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EscrowAccountAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.EscrowAccountAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReserveFee", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ReserveFee = append(m.ReserveFee, types.Coin{})
			if err := m.ReserveFee[len(m.ReserveFee)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinBuyInFee", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MinBuyInFee = append(m.MinBuyInFee, types.Coin{})
			if err := m.MinBuyInFee[len(m.MinBuyInFee)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinBidIncrement", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MinBidIncrement = append(m.MinBidIncrement, types.Coin{})
			if err := m.MinBidIncrement[len(m.MinBidIncrement)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)
