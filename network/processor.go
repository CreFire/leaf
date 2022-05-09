package network

import "github.com/CreFire/leaf/network/cstruct"

type Processor interface {
	// Route must goroutine safe
	Route(msg *cstruct.RecvMsg, userData interface{}) error
	// Unmarshal must goroutine safe
	Unmarshal(data []byte) (interface{}, error)
	// Marshal must goroutine safe
	Marshal(msg interface{}) ([][]byte, error)
}
