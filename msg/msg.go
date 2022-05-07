package msg

import (
	"reflect"

	"github.com/CreFire/leaf/chanrpc"
	"github.com/CreFire/leaf/log"
	"github.com/CreFire/leaf/module"
	"github.com/CreFire/leaf/network"
	"github.com/CreFire/leaf/network/cstruct"
)

var (
	// Processor = protobuf.NewProcessor()
	Processor = cstruct.NewProcessor()
	MsgParser = network.NewMsgParser()
	skeleton  *module.Skeleton
)

func GetMsgData(recv *cstruct.RecvMsg, mainCmdID uint16, subCmdID uint16, msg interface{}) []byte {
	data, err := Processor.Marshal(recv, mainCmdID, subCmdID, msg)
	if err != nil {
		log.Error("GetMsgData Marshal message %v error: %v", reflect.TypeOf(msg), err)
		return nil
	}

	result, err := MsgParser.Pack(data...)
	if err != nil {
		log.Error("GetMsgData pack message error: %v", err)
		return nil
	}

	return result
}
func GetMsgDataRawBody(recv *cstruct.RecvMsg, mainCmdID uint16, subCmdID uint16, body []byte) []byte {
	header, err := Processor.MarshalCmd(recv, mainCmdID, subCmdID)
	if err != nil {
		log.Error("GetMsgDataRawBody MarshalCmd error: %v", err)
		return nil
	}

	result, err := MsgParser.Pack(header, body)
	if err != nil {
		log.Error("GetMsgDataRawBody pack message error: %v", err)
		return nil
	}

	return result
}
func GetMsgBodyData(msg interface{}) []byte {
	data, err := Processor.MarshalBody(msg)
	if err != nil {
		log.Error("MarshalBody message %v error: %v", reflect.TypeOf(msg), err)
		return nil
	}

	return data
}
func GetMsgObject(mainCmdID uint16, subCmdID uint16, data []byte) interface{} {
	msg, err := Processor.UnmarshalBody(mainCmdID, subCmdID, data)
	if err != nil {
		log.Error("GetMsgObject UnmarshalBody id [%v,%v] message %v error: %v", mainCmdID, subCmdID, reflect.TypeOf(msg), err)
		return nil
	}

	return msg
}

func SetMsgSkeleton(skeleto *module.Skeleton) {
	skeleton = skeleto
}

func Register(mainCmdID uint16, subCmdID uint16, msg interface{}, msgRouter *chanrpc.Server, h interface{}) {
	Processor.Register(mainCmdID, subCmdID, msg)
	Processor.SetRouter(mainCmdID, subCmdID, msgRouter)
	m := cstruct.MakeDWORD(mainCmdID, subCmdID)
	skeleton.RegisterChanRPC(m, h)
}
