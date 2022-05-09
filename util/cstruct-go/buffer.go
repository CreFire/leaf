package cstruct

import (
	"encoding/binary"
	"io"
	"reflect"
	"strconv"
	"unsafe"
)

const (
	VersionFieldName    string = "Ver" // struct版本字段名，值为正数，一般定义为：Ver uint8，值从1开始；如果新增定义Ver字段，Ver可定义为无符号数据类型（无符号最小值为0），其它字段的tag ver设置从0开始，这样就可以处理所有字段数据
	VersionFieldTagName string = "ver" // struct版本字段tag名，表示定义该字段时的版本号，例如：Data uint32 `ver:"1"`，当 tag ver的值 <= struct版本字段值，本struc字段的数据就会做解码处理；当编码的时候发现 tag ver的值 > struct版本字段值，会panic
)

/*

示例

const VerMaxStRoleOne uint8 = 0

type StRoleOne struct {
	Ver       uint8  //版本号（服务器填充）
	Color     uint32 //染色使用的物品ID，0表示没有
	Id        uint32 //角色id，id为character.txt
	ValidTime int64  //时效，0表示永久
}

const VerMaxStFriend uint8 = 0

type StFriend struct {
	Ver         uint8
	UserID      uint32 `ver:"0"`
	Sex         uint8  `ver:"0"` //玩家性别1男2女
	TimeSongLi  uint8  `ver:"0"` //最近送礼时间，已废弃表示时间，0表示当天没有送过礼，1表示当天已经送过礼
	LvlShiShen  uint8  `ver:"0"` //食神等级
	Reserved    uint8  `ver:"0"` //保留字段
	Reserved2   uint32 `ver:"0"` //保留字段
	TimeDianZan uint32 `ver:"0"` //最近点赞时间
	Reserved3   uint32 `ver:"0"` //保留字段
	Name        string `ver:"0"` //玩家昵称
}

const VerMaxPetOne uint8 = 1

type StPetOne struct {
	Ver           uint8
	Status        uint8    `ver:"0"` //状态，0正常，1受伤
	XuanYong      uint8    `ver:"0"` //0未选用1选用
	RobNum        uint8    `ver:"0"` //已打劫次数
	Lvl           uint8    `ver:"0"` //宠物等级
	Star          uint8    `ver:"0"` //宠物星级，从0开始
	Reserved      uint16   `ver:"0"` //保留字段
	Id            uint32   `ver:"0"` //宠物id，id为 pet_role.txt
	ChengZhangZhi uint32   `ver:"0"` //成长值
	SkillGroupID  []uint16 `ver:"1"` //[技能组ID,技能等级...]，技能等级从1开始
}

*/

type Buffer struct {
	buf   []byte
	index int
}

func NewBuffer(e []byte) *Buffer {
	return &Buffer{buf: e}
}

func (p *Buffer) Reset() {
	p.buf = p.buf[:0]
	p.index = 0
}

// Marshal

func (p *Buffer) Marshal(obj IStruct) error {
	t, base, err := getbase(obj)
	if structPointer_IsNil(base) {
		return ErrNil
	}
	if err == nil {
		props := GetProperties(t.Elem())
		s := p.size_struct(props, base)
		p.buf = make([]byte, s)
		err = p.enc_struct(props, base)
	}
	return err
}

func getbase(obj IStruct) (t reflect.Type, b structPointer, err error) {
	if obj == nil {
		err = ErrNil
		return
	}
	t = reflect.TypeOf(obj)
	value := reflect.ValueOf(obj)
	b = toStructPointer(value)
	return
}

func (o *Buffer) size_struct(prop *StructProperties, base structPointer) int {
	ret := prop.fixedSize
	for _, p := range prop.Prop {
		if p.siz != nil {
			ret += p.siz(o, p, base)
		}
	}
	return ret
}

func (o *Buffer) enc_struct(prop *StructProperties, base structPointer) error {
	var propVer int = -2
	for _, p := range prop.Prop {
		if p.enc != nil {
			if p.ver <= propVer {
				if err := p.enc(o, p, base); err != nil {
					return err
				}
			} else {
				if propVer == -2 && p.ver == -1 {
					switch p.t.Kind() {
					case reflect.Int8, reflect.Uint8: // int8 uint8
						v := (*uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
						propVer = int(*v)
					case reflect.Int16, reflect.Uint16: // int16 uint16
						v := (*uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
						propVer = int(*v)
					case reflect.Int32, reflect.Uint32: // int32 uint32
						v := (*uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
						propVer = int(*v)
					default:
						panic("cstruct: unknow type. field name = " + p.Name + " kind: " + strconv.Itoa(int(p.t.Kind())))
					}
					if err := p.enc(o, p, base); err != nil {
						return err
					}
				} else {
					panic("cstruct: enc_struct ver err: field name = " + p.Name + " p.ver[" + strconv.Itoa(int(p.ver)) + "] > propVer[" + strconv.Itoa(int(propVer)) + "]")
				}
			}
		}
	}
	return nil
}

// Unmarshal

func (p *Buffer) Unmarshal(obj IStruct) error {
	typ, base, err := getbase(obj)
	if err != nil {
		return err
	}

	return p.unmarshalType(typ.Elem(), GetProperties(typ.Elem()), base)
}

func (o *Buffer) unmarshalType(st reflect.Type, prop *StructProperties, base structPointer) error {
	var propVer int = -2
	for _, p := range prop.Prop {
		if p.dec != nil {
			if p.ver <= propVer {
				if err := p.dec(o, p, base); err != nil {
					return err
				}
			} else {
				if propVer == -2 && p.ver == -1 {
					if err := p.dec(o, p, base); err != nil {
						return err
					}
					switch p.t.Kind() {
					case reflect.Int8, reflect.Uint8: // int8 uint8
						v := (*uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
						propVer = int(*v)
					case reflect.Int16, reflect.Uint16: // int16 uint16
						v := (*uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
						propVer = int(*v)
					case reflect.Int32, reflect.Uint32: // int32 uint32
						v := (*uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
						propVer = int(*v)
					default:
						panic("cstruct: unknow type. field name = " + p.Name + " kind: " + strconv.Itoa(int(p.t.Kind())))
					}
				}
			}
		}
	}
	return nil
}

// bool
func (o *Buffer) enc_bool(p *Properties, base structPointer) error {
	v := structPointer_BoolVal(base, p.field)
	x := 0
	if *v {
		x = 1
	}
	o.buf[o.index] = uint8(x)
	o.index++
	return nil
}

func (o *Buffer) dec_bool(p *Properties, base structPointer) error {
	i := o.index + 1
	if i < 0 || i > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	o.index = i
	u := uint8(o.buf[i-1])
	v := structPointer_BoolVal(base, p.field)
	*v = (u != 0)
	return nil
}

// uint8
func (o *Buffer) enc_uint8(p *Properties, base structPointer) error {
	v := (*uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	o.buf[o.index] = *v
	o.index++
	return nil
}

func (o *Buffer) dec_uint8(p *Properties, base structPointer) error {
	i := o.index + 1
	if i < 0 || i > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	o.index = i
	u := uint8(o.buf[i-1])
	v := (*uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	*v = u
	return nil
}

// uint16
func (o *Buffer) enc_uint16(p *Properties, base structPointer) error {
	v := (*uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	binary.LittleEndian.PutUint16(o.buf[o.index:], *v)
	o.index += 2
	return nil
}

func (o *Buffer) dec_uint16(p *Properties, base structPointer) error {
	u, err := o.readUInt16()
	if err != nil {
		return err
	}
	v := (*uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	*v = u
	return nil
}

// uint32
func (o *Buffer) enc_uint32(p *Properties, base structPointer) error {
	v := (*uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	binary.LittleEndian.PutUint32(o.buf[o.index:], *v)
	o.index += 4
	return nil
}

func (o *Buffer) dec_uint32(p *Properties, base structPointer) error {
	v := (*uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	i := o.index + 4
	if i < 0 || i > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	o.index = i
	*v = binary.LittleEndian.Uint32(o.buf[i-4:])
	return nil
}

// uint64
func (o *Buffer) enc_uint64(p *Properties, base structPointer) error {
	v := (*uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	binary.LittleEndian.PutUint64(o.buf[o.index:], *v)
	o.index += 8
	return nil
}

func (o *Buffer) dec_uint64(p *Properties, base structPointer) error {
	v := (*uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	i := o.index + 8
	if i < 0 || i > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	o.index = i
	*v = binary.LittleEndian.Uint64(o.buf[i-8:])
	return nil
}

// string
func (o *Buffer) enc_string(p *Properties, base structPointer) error {
	v := structPointer_StringVal(base, p.field)
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	if ln > 0 {
		copy(o.buf[o.index:], *v)
		o.index += ln
	}
	return nil
}

func (o *Buffer) dec_string(p *Properties, base structPointer) error {
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}

	end := o.index + int(nb)
	if end < o.index || end > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	buf := o.buf[o.index:end]
	o.index = end

	v := structPointer_StringVal(base, p.field)
	*v = string(buf)
	return nil
}

func (o *Buffer) size_string(p *Properties, base structPointer) int {
	v := structPointer_StringVal(base, p.field)
	return len(*v)
}

// []byte
func (o *Buffer) enc_slice_byte(p *Properties, base structPointer) error {
	v := structPointer_Bytes(base, p.field)
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	if ln > 0 {
		copy(o.buf[o.index:], *v)
		o.index += ln
	}
	return nil
}

func (o *Buffer) dec_slice_byte(p *Properties, base structPointer) error {
	v := structPointer_Bytes(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}

	end := o.index + int(nb)
	if end < o.index || end > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	*v = append((*v)[:0], o.buf[o.index:end]...)
	o.index = end
	return nil
}

func (o *Buffer) size_slice_byte(p *Properties, base structPointer) int {
	v := structPointer_Bytes(base, p.field)
	return len(*v)
}

// struct ptr
func (o *Buffer) enc_substruct_ptr(p *Properties, base structPointer) error {
	v := structPointer_GetStructPointer(base, p.field)
	if v == nil {
		o.buf[o.index] = uint8(0)
		o.index++
		return nil
	} else {
		o.buf[o.index] = uint8(1)
		o.index++
		return o.enc_struct(p.sprop, v)
	}
}
func (o *Buffer) dec_substruct_ptr(p *Properties, base structPointer) error {
	bas := structPointer_GetStructPointer(base, p.field)
	i := o.index + 1
	if i < 0 || i > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	o.index = i
	flag := uint8(o.buf[i-1])
	if flag == 0 {
		return nil
	}
	if structPointer_IsNil(bas) {
		bas = toStructPointer(reflect.New(p.stype))
		structPointer_SetStructPointer(base, p.field, bas)
	}
	return o.unmarshalType(p.stype, p.sprop, bas)
}
func (o *Buffer) size_substruct_ptr(p *Properties, base structPointer) int {
	v := structPointer_GetStructPointer(base, p.field)
	if v == nil {
		return 1
	}
	return o.size_struct(p.sprop, v) + 1
}

// struct
func (o *Buffer) enc_substruct(p *Properties, base structPointer) error {
	return o.enc_struct(p.sprop, structPointer(unsafe.Pointer(uintptr(base)+uintptr(p.field))))
}
func (o *Buffer) dec_substruct(p *Properties, base structPointer) error {
	bas := structPointer(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	return o.unmarshalType(p.stype, p.sprop, bas)
}
func (o *Buffer) size_substruct(p *Properties, base structPointer) int {
	return o.size_struct(p.sprop, structPointer(unsafe.Pointer(uintptr(base)+uintptr(p.field))))
}

// []bool
func (o *Buffer) enc_slice_bool(p *Properties, base structPointer) error {
	v := structPointer_BoolSlice(base, p.field)
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		x := 0
		if (*v)[i] {
			x = 1
		}
		o.buf[o.index] = uint8(x)
		o.index += 1
	}
	return nil
}

func (o *Buffer) dec_slice_bool(p *Properties, base structPointer) error {
	v := structPointer_BoolSlice(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	end := o.index + int(nb)
	if end < o.index || end > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	*v = make([]bool, int(nb))
	for i := 0; i < int(nb); i++ {
		u := uint8(o.buf[o.index+i])
		(*v)[i] = (u != 0)
	}
	o.index = end
	return nil
}

func (o *Buffer) size_slice_bool(p *Properties, base structPointer) int {
	v := structPointer_BoolSlice(base, p.field)
	return len(*v)
}

// []uint16
func (o *Buffer) enc_slice_uint16(p *Properties, base structPointer) error {
	v := (*[]uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		binary.LittleEndian.PutUint16(o.buf[o.index:], (*v)[i])
		o.index += 2
	}
	return nil
}

func (o *Buffer) dec_slice_uint16(p *Properties, base structPointer) error {
	v := (*[]uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	end := o.index + int(nb)*2
	if end < o.index || end > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	*v = make([]uint16, int(nb))
	for i := 0; i < int(nb); i++ {
		(*v)[i] = binary.LittleEndian.Uint16(o.buf[o.index+i*2:])
	}
	o.index = end
	return nil
}

func (o *Buffer) size_slice_uint16(p *Properties, base structPointer) int {
	v := (*[]uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	return len(*v) * 2
}

// []uint32
func (o *Buffer) enc_slice_uint32(p *Properties, base structPointer) error {
	v := (*[]uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		binary.LittleEndian.PutUint32(o.buf[o.index:], (*v)[i])
		o.index += 4
	}
	return nil
}

func (o *Buffer) dec_slice_uint32(p *Properties, base structPointer) error {
	v := (*[]uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	end := o.index + int(nb)*4
	if end < o.index || end > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	*v = make([]uint32, int(nb))
	for i := 0; i < int(nb); i++ {
		(*v)[i] = binary.LittleEndian.Uint32(o.buf[o.index+i*4:])
	}
	o.index = end
	return nil
}

func (o *Buffer) size_slice_uint32(p *Properties, base structPointer) int {
	v := (*[]uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	return len(*v) * 4
}

// []uint64
func (o *Buffer) enc_slice_uint64(p *Properties, base structPointer) error {
	v := (*[]uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		binary.LittleEndian.PutUint64(o.buf[o.index:], (*v)[i])
		o.index += 8
	}
	return nil
}

func (o *Buffer) dec_slice_uint64(p *Properties, base structPointer) error {
	v := (*[]uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	end := o.index + int(nb)*8
	if end < o.index || end > len(o.buf) {
		return io.ErrUnexpectedEOF
	}
	*v = make([]uint64, int(nb))
	for i := 0; i < int(nb); i++ {
		(*v)[i] = binary.LittleEndian.Uint64(o.buf[o.index+i*8:])
	}
	o.index = end
	return nil
}

func (o *Buffer) size_slice_uint64(p *Properties, base structPointer) int {
	v := (*[]uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	return len(*v) * 8
}

// []string
func (o *Buffer) enc_slice_string(p *Properties, base structPointer) error {
	v := (*[]string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		ln2 := len((*v)[i])
		binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln2))
		o.index += 2
		if ln2 > 0 {
			copy(o.buf[o.index:], (*v)[i])
			o.index += ln2
		}
	}
	return nil
}

func (o *Buffer) dec_slice_string(p *Properties, base structPointer) error {
	v := (*[]string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	nb0, err0 := o.readUInt16()
	if err0 != nil {
		return err0
	}
	*v = make([]string, int(nb0))
	for i := 0; i < int(nb0); i++ {
		nb, err := o.readUInt16()
		if err != nil {
			return err
		}
		end := o.index + int(nb)
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		(*v)[i] = string(o.buf[o.index:end])
		o.index = end
	}
	return nil
}

func (o *Buffer) size_slice_string(p *Properties, base structPointer) int {
	v := (*[]string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ret := 0
	ln := len(*v)
	for i := 0; i < ln; i++ {
		ret += len((*v)[i]) + 2
	}
	return ret
}

// []struct
func (o *Buffer) enc_slice_substruct(p *Properties, base structPointer) error {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	var ln = sliceHeader.Len
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	itemsize := int(p.stype.Size())
	for i := 0; i < ln; i++ {
		sv := (structPointer)(unsafe.Pointer(sliceHeader.Data + uintptr(i*itemsize)))
		o.enc_struct(p.sprop, sv)
	}
	return nil
}

func (o *Buffer) dec_slice_substruct(p *Properties, base structPointer) error {
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	if nb == 0 {
		return nil
	}
	itemsize := int(p.stype.Size())
	data := reflect.MakeSlice(p.t, int(nb), int(nb))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	sliceHeader.Cap = int(nb)
	sliceHeader.Len = int(nb)
	sliceHeader.Data = data.Pointer()
	for i := 0; i < int(nb); i++ {
		data := (structPointer)(unsafe.Pointer(sliceHeader.Data + uintptr(i*itemsize)))
		o.unmarshalType(p.stype, p.sprop, data)
	}
	return nil
}

func (o *Buffer) size_slice_substruct(p *Properties, base structPointer) int {
	ret := 0
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	var ln = sliceHeader.Len
	itemsize := int(p.stype.Size())
	for i := 0; i < ln; i++ {
		sv := (structPointer)(unsafe.Pointer(sliceHeader.Data + uintptr(i*itemsize)))
		ret += o.size_struct(p.sprop, sv)
	}
	return ret
}

// []struct_ptr
func (o *Buffer) enc_slice_substruct_ptr(p *Properties, base structPointer) error {
	v := structPointer_StructPointerSlice(base, p.field)
	ln := v.Len()
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		sv := (*v)[i]
		if sv == nil {
			o.buf[o.index] = uint8(0)
			o.index++
		} else {
			o.buf[o.index] = uint8(1)
			o.index++
			o.enc_struct(p.sprop, sv)
		}
	}
	return nil
}

func (o *Buffer) dec_slice_substruct_ptr(p *Properties, base structPointer) error {
	v := structPointer_StructPointerSlice(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	for j, i := 0, 0; j < int(nb); j++ {
		i = o.index + 1
		if i < 0 || i > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		o.index = i
		flag := uint8(o.buf[i-1])
		if flag == 0 {
			v.Append(nil)
		} else {
			bas := toStructPointer(reflect.New(p.stype))
			o.unmarshalType(p.stype, p.sprop, bas)
			v.Append(bas)
		}
	}
	return nil
}

func (o *Buffer) size_slice_substruct_ptr(p *Properties, base structPointer) int {
	ret := 0
	v := structPointer_StructPointerSlice(base, p.field)
	ln := v.Len()
	for i := 0; i < ln; i++ {
		sv := (*v)[i]
		if sv == nil {
			ret += 1
		} else {
			ret += o.size_struct(p.sprop, sv) + 1
		}
	}
	return ret
}

// []struct_ptr ignore nil
func (o *Buffer) enc_slice_substruct_ptr_ignore_nil(p *Properties, base structPointer) error {
	v := structPointer_StructPointerSlice(base, p.field)
	ln := v.Len()
	len_index := o.index
	real_ln := 0
	o.index += 2
	for i := 0; i < ln; i++ {
		sv := (*v)[i]
		if sv != nil {
			real_ln++
			o.enc_struct(p.sprop, sv)
		}
	}
	binary.LittleEndian.PutUint16(o.buf[len_index:], uint16(real_ln))
	return nil
}

func (o *Buffer) dec_slice_substruct_ptr_ignore_nil(p *Properties, base structPointer) error {
	v := structPointer_StructPointerSlice(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	for j := 0; j < int(nb); j++ {
		bas := toStructPointer(reflect.New(p.stype))
		o.unmarshalType(p.stype, p.sprop, bas)
		v.Append(bas)
	}
	return nil
}

func (o *Buffer) size_slice_substruct_ptr_ignore_nil(p *Properties, base structPointer) int {
	ret := 0
	v := structPointer_StructPointerSlice(base, p.field)
	ln := v.Len()
	for i := 0; i < ln; i++ {
		sv := (*v)[i]
		if sv != nil {
			ret += o.size_struct(p.sprop, sv)
		}
	}
	return ret
}

// [][]byte
func (o *Buffer) enc_slice_slice_byte(p *Properties, base structPointer) error {
	v := structPointer_BytesSlice(base, p.field)
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		ln2 := len((*v)[i])
		binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln2))
		o.index += 2
		if ln2 > 0 {
			copy(o.buf[o.index:], (*v)[i])
			o.index += ln2
		}
	}
	return nil
}

func (o *Buffer) dec_slice_slice_byte(p *Properties, base structPointer) error {
	v := structPointer_BytesSlice(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	for i := 0; i < int(nb); i++ {
		nb2, err := o.readUInt16()
		if err != nil {
			return err
		}

		end := o.index + int(nb2)
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		buf := o.buf[o.index:end]
		o.index = end

		*v = append(*v, buf)
	}
	return nil
}

func (o *Buffer) size_slice_slice_byte(p *Properties, base structPointer) int {
	v := structPointer_BytesSlice(base, p.field)
	ret := 0
	ln := len(*v)
	for i := 0; i < ln; i++ {
		ret += len((*v)[i]) + 2
	}
	return ret
}

// [][]uint16
func (o *Buffer) enc_slice_slice_uint16(p *Properties, base structPointer) error {
	v := structPointer_Uint16sSlice(base, p.field)
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		ln2 := len((*v)[i])
		binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln2))
		o.index += 2
		if ln2 > 0 {
			for j := 0; j < ln2; j++ {
				binary.LittleEndian.PutUint16(o.buf[o.index:], (*v)[i][j])
				o.index += 2
			}
		}
	}
	return nil
}

func (o *Buffer) dec_slice_slice_uint16(p *Properties, base structPointer) error {
	v := structPointer_Uint16sSlice(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	*v = make([][]uint16, int(nb))
	for i := 0; i < int(nb); i++ {
		nb2, err := o.readUInt16()
		if err != nil {
			return err
		}

		end := o.index + int(nb2)*2
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		v2 := make([]uint16, int(nb2))
		for j := 0; j < int(nb2); j++ {
			v2[j] = binary.LittleEndian.Uint16(o.buf[o.index+j*2:])
		}
		(*v)[i] = v2
		o.index = end
	}
	return nil
}

func (o *Buffer) size_slice_slice_uint16(p *Properties, base structPointer) int {
	v := structPointer_Uint16sSlice(base, p.field)
	ret := 0
	ln := len(*v)
	for i := 0; i < ln; i++ {
		ret += len((*v)[i])*2 + 2
	}
	return ret
}

// [][]uint32
func (o *Buffer) enc_slice_slice_uint32(p *Properties, base structPointer) error {
	v := structPointer_Uint32sSlice(base, p.field)
	ln := len(*v)
	binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln))
	o.index += 2
	for i := 0; i < ln; i++ {
		ln2 := len((*v)[i])
		binary.LittleEndian.PutUint16(o.buf[o.index:], uint16(ln2))
		o.index += 2
		if ln2 > 0 {
			for j := 0; j < ln2; j++ {
				binary.LittleEndian.PutUint32(o.buf[o.index:], (*v)[i][j])
				o.index += 4
			}
		}
	}
	return nil
}

func (o *Buffer) dec_slice_slice_uint32(p *Properties, base structPointer) error {
	v := structPointer_Uint32sSlice(base, p.field)
	nb, err := o.readUInt16()
	if err != nil {
		return err
	}
	*v = make([][]uint32, int(nb))
	for i := 0; i < int(nb); i++ {
		nb2, err := o.readUInt16()
		if err != nil {
			return err
		}

		end := o.index + int(nb2)*4
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		v2 := make([]uint32, int(nb2))
		for j := 0; j < int(nb2); j++ {
			v2[j] = binary.LittleEndian.Uint32(o.buf[o.index+j*4:])
		}
		(*v)[i] = v2
		o.index = end
	}
	return nil
}

func (o *Buffer) size_slice_slice_uint32(p *Properties, base structPointer) int {
	v := structPointer_Uint32sSlice(base, p.field)
	ret := 0
	ln := len(*v)
	for i := 0; i < ln; i++ {
		ret += len((*v)[i])*4 + 2
	}
	return ret
}

func (o *Buffer) readUInt16() (uint16, error) {
	i := o.index + 2
	if i < 0 || i > len(o.buf) {
		return 0, io.ErrUnexpectedEOF
	}
	o.index = i
	return binary.LittleEndian.Uint16(o.buf[i-2:]), nil
}

// [n]byte [n]uint8 [n]int8 [n]bool
func (o *Buffer) enc_array_byte(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		var data []byte
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		copy(o.buf[o.index:], data)
		o.index += ln
	}
	return nil
}

func (o *Buffer) dec_array_byte(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		end := o.index + ln
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		var data []byte
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		copy(data, o.buf[o.index:end])
		o.index = end
	}
	return nil
}

func (o *Buffer) size_array_byte(p *Properties, base structPointer) int {
	return p.t.Len()
}

// [n]uint16 [n]int16
func (o *Buffer) enc_array_uint16(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		var data []uint16
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		for i := 0; i < ln; i++ {
			binary.LittleEndian.PutUint16(o.buf[o.index:], data[i])
			o.index += 2
		}
	}
	return nil
}

func (o *Buffer) dec_array_uint16(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		end := o.index + ln*2
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		var data []uint16
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		for i := 0; i < ln; i++ {
			data[i] = binary.LittleEndian.Uint16(o.buf[o.index+i*2:])
		}
		o.index = end
	}
	return nil
}

func (o *Buffer) size_array_uint16(p *Properties, base structPointer) int {
	return p.t.Len() * 2
}

// [n]uint32 [n]int32 [n]float32
func (o *Buffer) enc_array_uint32(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		var data []uint32
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		for i := 0; i < ln; i++ {
			binary.LittleEndian.PutUint32(o.buf[o.index:], data[i])
			o.index += 4
		}
	}
	return nil
}

func (o *Buffer) dec_array_uint32(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		end := o.index + ln*4
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		var data []uint32
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		for i := 0; i < ln; i++ {
			data[i] = binary.LittleEndian.Uint32(o.buf[o.index+i*4:])
		}
		o.index = end
	}
	return nil
}

func (o *Buffer) size_array_uint32(p *Properties, base structPointer) int {
	return p.t.Len() * 4
}

// [n]uint64 [n]int64 [n]float64
func (o *Buffer) enc_array_uint64(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		var data []uint64
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		for i := 0; i < ln; i++ {
			binary.LittleEndian.PutUint64(o.buf[o.index:], data[i])
			o.index += 8
		}
	}
	return nil
}

func (o *Buffer) dec_array_uint64(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		end := o.index + ln*8
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		var data []uint64
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&data)))
		sliceHeader.Cap = ln
		sliceHeader.Len = ln
		sliceHeader.Data = uintptr(base) + uintptr(p.field)
		for i := 0; i < ln; i++ {
			data[i] = binary.LittleEndian.Uint64(o.buf[o.index+i*8:])
		}
		o.index = end
	}
	return nil
}

func (o *Buffer) size_array_uint64(p *Properties, base structPointer) int {
	return p.t.Len() * 8
}

// [n]struct
func (o *Buffer) enc_array_substruct(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		itemsize := int(p.stype.Size())
		for i := 0; i < ln; i++ {
			data := (structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field) + uintptr(i*itemsize)))
			o.enc_struct(p.sprop, data)
		}
	}
	return nil
}

func (o *Buffer) dec_array_substruct(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		itemsize := o.size_substruct(p, base)
		end := o.index + ln*itemsize
		if end < o.index || end > len(o.buf) {
			return io.ErrUnexpectedEOF
		}
		for i := 0; i < ln; i++ {
			data := (structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field) + uintptr(i*int(p.stype.Size()))))
			o.unmarshalType(p.stype, p.sprop, data)
		}
		o.index = end
	}
	return nil
}

func (o *Buffer) size_array_substruct(p *Properties, base structPointer) int {
	return p.t.Len() * o.size_substruct(p, base)
}

// [n]*struct
func (o *Buffer) enc_array_substruct_ptr(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		for i := 0; i < ln; i++ {
			data := (*structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field) + uintptr(i)*ptrSize))
			if (*data) == nil {
				o.buf[o.index] = uint8(0)
				o.index++
			} else {
				o.buf[o.index] = uint8(1)
				o.index++
				o.enc_struct(p.sprop, *data)
			}
		}
	}
	return nil
}

func (o *Buffer) dec_array_substruct_ptr(p *Properties, base structPointer) error {
	ln := p.t.Len()
	if ln > 0 {
		for j, i := 0, 0; j < ln; j++ {
			i = o.index + 1
			if i < 0 || i > len(o.buf) {
				return io.ErrUnexpectedEOF
			}
			o.index = i
			data := (*structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field) + uintptr(j)*ptrSize))
			flag := uint8(o.buf[i-1])
			if flag == 0 {
				*data = nil
			} else {
				bas := toStructPointer(reflect.New(p.stype))
				o.unmarshalType(p.stype, p.sprop, bas)
				*data = bas
			}
		}
	}
	return nil
}

func (o *Buffer) size_array_substruct_ptr(p *Properties, base structPointer) int {
	ret := 0
	ln := p.t.Len()
	for i := 0; i < ln; i++ {
		data := (*structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field) + uintptr(i)*ptrSize))
		if (*data) == nil {
			ret += 1
		} else {
			ret += o.size_struct(p.sprop, *data) + 1
		}
	}
	return ret
}
