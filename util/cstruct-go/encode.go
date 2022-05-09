package cstruct

func Marshal(obj IStruct) ([]byte, error) {
	p := NewBuffer(nil)
	err := p.Marshal(obj)
	if p.buf == nil && err == nil {
		return []byte{}, nil
	}
	return p.buf, err
}

func GetSize(obj IStruct) (uint16, error) {
	p := NewBuffer(nil)

	t, base, err := getbase(obj)
	if structPointer_IsNil(base) {
		return 0, ErrNil
	}
	if err == nil {
		props := GetProperties(t.Elem())
		s := p.size_struct(props, base)
		return uint16(s), nil
	}
	return 0, err
}
