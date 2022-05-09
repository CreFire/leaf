package cstruct

import "errors"

var (
	ErrNil = errors.New("cstruct: Marshal called with nil")
)

type IStruct interface {
}

// OptionSliceIgnoreNil slice 元素类型为指针时，是否忽略nil
var OptionSliceIgnoreNil = false

// OptionSliceStructPointer slice 元素类型为结构体时，是否要求为结构体指针
var OptionSliceStructPointer = true
