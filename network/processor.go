package network

import "github.com/CreFire/leaf/network/cstruct"

type Processor interface {
	// Route must goroutine safe
	Route(msg *cstruct.RecvMsg, userData interface{}) error
	// Unmarshal must goroutine safe
	Unmarshal(data []byte) (*cstruct.RecvMsg, error)
	// Marshal must goroutine safe
	Marshal(recv *cstruct.RecvMsg, mainCmdID uint16, subCmdID uint16, msg interface{}) ([][]byte, error)
}
