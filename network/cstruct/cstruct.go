package cstruct

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/CreFire/leaf/util/cstruct-go"
	"google.golang.org/protobuf/proto"
	"math"

	"reflect"

	"github.com/CreFire/leaf/chanrpc"
	"github.com/CreFire/leaf/log"
)

const (
	MSG_TYPE_NONE uint8 = 0    // 默认的一般消息类型
	MSG_TYPE_RPC  uint8 = 0x01 // rpc
	// MSG_TYPE_NONE uint8 = 0x02 //
	// MSG_TYPE_NONE uint8 = 0x04 //
	// MSG_TYPE_NONE uint8 = 0x08 //
	// MSG_TYPE_NONE uint8 = 0x10 //
)

type RecvMsg struct {
	RpcCallId uint32
	MsgId     uint32
	Msg       interface{}
	MsgType   uint8
}

var DefaultRecvMsg = &RecvMsg{0, 0, nil, MSG_TYPE_NONE}

func FlagSet(value uint8, flag uint8) uint8 {
	return value | flag
}

func FlagUnset(value uint8, flag uint8) uint8 {
	return value & (^flag)
}

func FlagGet(value uint8, flag uint8) bool {
	return (value & flag) != 0
}

func MakeDWORD(mainCmdID uint16, subCmdID uint16) uint32 {
	return uint32(mainCmdID) | uint32(subCmdID)<<16
}

func GetCmd(CmdID uint32) (mainCmdID, subCmdID uint16) {
	mainCmdID = uint16(CmdID)
	subCmdID = uint16(CmdID >> 16)
	return
}

// -------------------------
// | id | protobuf message |
// -------------------------
type Processor struct {
	littleEndian bool
	msgInfo      map[uint32]*MsgInfo
	// msgID        map[reflect.Type]uint32
	// msgType      map[uint32]reflect.Type
}

type MsgInfo struct {
	msgType       reflect.Type
	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
	bProtoBuf     bool // true表示消息体是proto buffer，false表示消息体是cstruct
}

type MsgHandler func([]interface{})

type MsgRaw struct {
	msgID      uint32
	msgRawData []byte
}

func NewProcessor() *Processor {
	cstruct.OptionSliceIgnoreNil = true

	log.Debug("NewProcessor")
	p := new(Processor)
	p.littleEndian = true
	p.msgInfo = make(map[uint32]*MsgInfo)
	// p.msgID = make(map[reflect.Type]uint32)
	// p.msgType = make(map[uint32]reflect.Type)

	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(mainCmdID uint16, subCmdID uint16, msg interface{}) uint32 {
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)
	msgInfo := new(MsgInfo)
	msgInfo.bProtoBuf = false

	if nil == msg {
		msgInfo.msgType = nil
		p.msgInfo[id] = msgInfo
		// p.msgType[id] = nil
		return id
	}

	_, ok := msg.(proto.Message)
	if ok {
		msgInfo.bProtoBuf = true
	}

	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("protobuf message pointer required")
	}
	// if _, ok := p.msgID[msgType]; ok {
	// 	log.Fatal("message type %s is already registered", msgType)
	// }
	//
	// if _, ok := p.msgType[id]; ok {
	// 	log.Fatal("message id %s is already registered", msgType)
	// }
	if len(p.msgInfo) >= math.MaxUint16 {
		log.Fatal("too many protobuf messages (max = %v)", math.MaxUint16)
	}

	msgInfo.msgType = msgType
	p.msgInfo[id] = msgInfo
	// p.msgID[msgType] = id
	// p.msgType[id] = msgType
	return id
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(mainCmdID uint16, subCmdID uint16, msgRouter *chanrpc.Server) {
	// msgType := reflect.TypeOf(msg)
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)
	_, ok := p.msgInfo[id]
	if !ok {
		log.Fatal("message %d,%d not registered", mainCmdID, subCmdID)
	}

	p.msgInfo[id].msgRouter = msgRouter
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetHandler(mainCmdID uint16, subCmdID uint16, msgHandler MsgHandler) {
	// msgType := reflect.TypeOf(msg)
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)
	_, ok := p.msgInfo[id]
	if !ok {
		log.Fatal("message %d,%d not registered", mainCmdID, subCmdID)
	}

	p.msgInfo[id].msgHandler = msgHandler
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRawHandler(mainCmdID uint16, subCmdID uint16, msgRawHandler MsgHandler) {
	// msgType := reflect.TypeOf(msg)
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)
	_, ok := p.msgInfo[id]
	if !ok {
		log.Fatal("message %d,%d not registered", mainCmdID, subCmdID)
	}

	p.msgInfo[id].msgRawHandler = msgRawHandler
}

// goroutine safe
func (p *Processor) Route(msg *RecvMsg, userData interface{}) error {
	// raw
	if msgRaw, ok := msg.Msg.(MsgRaw); ok {
		if _, ok := p.msgInfo[msgRaw.msgID]; !ok {
			return fmt.Errorf("message id %v not registered", msgRaw.msgID)
		}
		i := p.msgInfo[msgRaw.msgID]
		if i.msgRawHandler != nil {
			i.msgRawHandler([]interface{}{msgRaw.msgID, msgRaw.msgRawData, userData})
			return nil
		} else {
			return fmt.Errorf("msg not handle")
		}
	}

	// protobuf
	// msgType := reflect.TypeOf(msg)
	// id, ok := p.msgID[msgType]
	// if !ok {
	// 	return fmt.Errorf("message %s not registered", msgType)
	// }
	i := p.msgInfo[msg.MsgId]
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil {
		i.msgRouter.Go(msg.MsgId, msg, userData)
	} else if i.msgHandler == nil {
		return fmt.Errorf("msg not handle")
	}

	return nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (*RecvMsg, error) {
	if len(data) < 5 {
		return &RecvMsg{0, 0, nil, MSG_TYPE_NONE}, errors.New("cstruct data too short")
	}

	var msgType uint8 = uint8(data[0])

	// id
	var id uint32
	if p.littleEndian {
		id = binary.LittleEndian.Uint32(data[1:])
	} else {
		id = binary.BigEndian.Uint32(data[1:])
	}

	// 判断是否有rpc call id字段
	var idx int = 5
	var rpcCallId uint32
	if FlagGet(msgType, MSG_TYPE_RPC) {
		// 有rpc call id字段
		if p.littleEndian {
			rpcCallId = binary.LittleEndian.Uint32(data[idx:])
		} else {
			rpcCallId = binary.BigEndian.Uint32(data[idx:])
		}
		idx += 4
	}

	// 有rpc call id字段的时候，检查异常情况
	if msgType != MSG_TYPE_NONE {
		mainCmdID, subCmdID := GetCmd(id)
		log.Debug("Unmarshal msgType[%d]!=0 id [%d,%d,%d] message", msgType, mainCmdID, subCmdID, rpcCallId)

		if rpcCallId == 0 {
			log.Error("Unmarshal error: msgType[%d]!=0 id but rpcCallId[%d]==0 message [%d,%d] ", msgType, rpcCallId, mainCmdID, subCmdID)
		}
	}

	if _, ok := p.msgInfo[id]; !ok {
		mainCmdID, subCmdID := GetCmd(id)
		return &RecvMsg{rpcCallId, 0, nil, msgType}, fmt.Errorf("message id [%v,%v] not registered", mainCmdID, subCmdID)
	}

	// msg
	i := p.msgInfo[id]
	if i.msgRawHandler != nil {
		return &RecvMsg{rpcCallId, id, MsgRaw{id, data[idx:]}, msgType}, nil
	} else {
		if nil != i.msgType {
			msg := reflect.New(i.msgType.Elem()).Interface()
			var err error
			if i.bProtoBuf {
				err = proto.Unmarshal(data[idx:], msg.(proto.Message))
			} else {
				err = cstruct.Unmarshal(data[idx:], msg)
			}
			if err != nil {
				mainCmdID, subCmdID := GetCmd(id)
				log.Error("Unmarshal id [%v,%v] message %v error: %v", mainCmdID, subCmdID, reflect.TypeOf(msg), err)
			}
			return &RecvMsg{rpcCallId, id, msg, msgType}, err
		} else {
			return &RecvMsg{rpcCallId, id, nil, msgType}, nil
		}
	}
}
func (p *Processor) UnmarshalBody(mainCmdID uint16, subCmdID uint16, data []byte) (interface{}, error) {
	// id
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)

	if _, ok := p.msgInfo[id]; !ok {
		return nil, fmt.Errorf("message id [%v,%v] not registered", mainCmdID, subCmdID)
	}

	// msg
	i := p.msgInfo[id]
	if i.msgRawHandler != nil {
		return MsgRaw{id, data}, nil
	} else {
		if nil != i.msgType {
			msg := reflect.New(i.msgType.Elem()).Interface()
			var err error
			if i.bProtoBuf {
				err = proto.Unmarshal(data, msg.(proto.Message))
			} else {
				err = cstruct.Unmarshal(data, msg)
			}
			if err != nil {
				log.Error("UnmarshalBody id [%v,%v] message %v error: %v", mainCmdID, subCmdID, reflect.TypeOf(msg), err)
			}
			return msg, err
		} else {
			return nil, nil
		}
	}
}

// goroutine safe
func (p *Processor) Marshal(recv *RecvMsg, mainCmdID uint16, subCmdID uint16, msg interface{}) ([][]byte, error) {
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)

	var header []byte

	if !FlagGet(recv.MsgType, MSG_TYPE_RPC) {
		header = make([]byte, 5)
		header[0] = recv.MsgType
		if p.littleEndian {
			binary.LittleEndian.PutUint32(header[1:], id)
		} else {
			binary.BigEndian.PutUint32(header[1:], id)
		}
	} else {
		if recv.RpcCallId == 0 {
			log.Error("Marshal error: msgType[%d]!=0 id but rpcCallId[%d]==0 message [%d,%d] ", recv.MsgType, recv.RpcCallId, mainCmdID, subCmdID)
		}
		// RPC消息
		header = make([]byte, 9)
		header[0] = recv.MsgType
		if p.littleEndian {
			binary.LittleEndian.PutUint32(header[1:], id)
			binary.LittleEndian.PutUint32(header[5:], recv.RpcCallId)
		} else {
			binary.BigEndian.PutUint32(header[1:], id)
			binary.BigEndian.PutUint32(header[5:], recv.RpcCallId)
		}
	}

	// data
	if nil == msg {
		return [][]byte{header}, nil
	}

	i := p.msgInfo[id]

	var err error
	var body []byte
	if nil != i {
		if i.bProtoBuf {
			body, err = proto.Marshal(msg.(proto.Message))
		} else {
			body, err = cstruct.Marshal(msg)
		}
	} else {
		_, ok := msg.(proto.Message)
		if ok {
			body, err = proto.Marshal(msg.(proto.Message))
		} else {
			body, err = cstruct.Marshal(msg)
		}
	}
	if err != nil {
		log.Error("Marshal %v error: %v", reflect.TypeOf(msg), err)
	}
	return [][]byte{header, body}, err
}
func (p *Processor) MarshalCmd(recv *RecvMsg, mainCmdID uint16, subCmdID uint16) ([]byte, error) {
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)

	var header []byte

	if !FlagGet(recv.MsgType, MSG_TYPE_RPC) {
		header = make([]byte, 5)
		header[0] = recv.MsgType
		if p.littleEndian {
			binary.LittleEndian.PutUint32(header[1:], id)
		} else {
			binary.BigEndian.PutUint32(header[1:], id)
		}
	} else {
		// RPC消息
		header = make([]byte, 9)
		header[0] = recv.MsgType
		if p.littleEndian {
			binary.LittleEndian.PutUint32(header[1:], id)
			binary.LittleEndian.PutUint32(header[5:], recv.RpcCallId)
		} else {
			binary.BigEndian.PutUint32(header[1:], id)
			binary.BigEndian.PutUint32(header[5:], recv.RpcCallId)
		}
	}

	return header, nil
}
func (p *Processor) MarshalBody(msg interface{}) ([]byte, error) {
	var err error
	var body []byte
	_, ok := msg.(proto.Message)
	if ok {
		body, err = proto.Marshal(msg.(proto.Message))
	} else {
		body, err = cstruct.Marshal(msg)
	}
	if err != nil {
		log.Error("MarshalBody %v error: %v", reflect.TypeOf(msg), err)
	}
	return body, err
}

// goroutine safe
func (p *Processor) Range(f func(id uint16, t reflect.Type)) {
	for id, i := range p.msgInfo {
		f(uint16(id), i.msgType)
	}
}

func (p *Processor) Cmd2Bytes(mainCmdID uint16, subCmdID uint16) []byte {
	var id uint32 = MakeDWORD(mainCmdID, subCmdID)
	cmd := make([]byte, 4)
	if p.littleEndian {
		binary.LittleEndian.PutUint32(cmd, id)
	} else {
		binary.BigEndian.PutUint32(cmd, id)
	}
	return cmd
}
