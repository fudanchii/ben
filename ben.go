package ben

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	DICT_START       = 'd'
	INT_START        = 'i'
	LIST_START       = 'l'
	SEQ_END          = 'e'
	LENGTH_DELIMITER = ':'
)

type ElementType int

const (
	IntType ElementType = iota
	StringType
	ListType
	DictType
)

type Element interface {
	Type() ElementType
	Encode() []byte
	Val() interface{}
}

func Decode(input io.ByteReader) (Element, error) {
	token, err := input.ReadByte()
	if err != nil {
		return nil, err
	}
	switch token {
	case DICT_START:
		return decodeDict(input)
	case INT_START:
		return decodeInt(input)
	case LIST_START:
		return decodeList(input)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return decodeString(input, token)
	case SEQ_END:
		return nil, nil
	default:
		return nil, InvalidInputError{}
	}
}

type Integer struct {
	data int64
}

func NewInteger(i int64) *Integer {
	return &Integer{i}
}

func decodeInt(input io.ByteReader) (*Integer, error) {
	var (
		ch    byte
		rInt  int64
		err   error
		minus bool
	)
	for {
		ch, err = input.ReadByte()
		if err != nil {
			return nil, err
		}
		if ch == SEQ_END {
			break
		}
		if ch == '-' && rInt == 0 {
			minus = true
			continue
		}

		i := int(ch) - 0x30
		if 0 <= i && i <= 9 {
			rInt = rInt*int64(10) + int64(i)
			continue
		}

		return nil, InvalidInputError{strconv.Itoa(int(ch))}
	}
	if minus {
		rInt = -rInt
	}
	return &Integer{rInt}, err
}

func (bInt *Integer) Type() ElementType {
	return IntType
}

func (bInt *Integer) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(INT_START)
	buff.WriteString(strconv.FormatInt(bInt.data, 10))
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

func (bInt *Integer) Val() interface{} {
	return bInt.data
}

type String struct {
	data string
}

func NewString(input string) *String {
	return &String{input}
}

func decodeString(input io.ByteReader, first_token byte) (*String, error) {
	var (
		buff   bytes.Buffer
		length []byte
		err    error
		sLen   int64
	)
	length = append(length, first_token)
	for {
		ch, err := input.ReadByte()
		if err != nil {
			return nil, err
		}
		if ch == LENGTH_DELIMITER {
			break
		}
		length = append(length, ch)
	}
	if sLen, err = strconv.ParseInt(string(length), 10, 64); err != nil {
		return nil, err
	}
	if _, err := io.CopyN(&buff, input.(io.Reader), sLen); err != nil {
		return nil, err
	}
	return &String{buff.String()}, err
}

func (bStr *String) Type() ElementType {
	return StringType
}

func (bStr *String) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteString(strconv.Itoa(len(bStr.data)))
	buff.WriteByte(LENGTH_DELIMITER)
	buff.WriteString(bStr.data)
	return buff.Bytes()
}

func (bStr *String) Val() interface{} {
	return bStr.data
}

type List struct {
	data []Element
}

func NewList(input []Element) *List {
	return &List{input}
}

func decodeList(input io.ByteReader) (*List, error) {
	var (
		lst []Element
	)
	for {
		lmnt, err := Decode(input)
		if err != nil {
			return nil, err
		}
		if lmnt == nil { // end of sequence
			break
		}
		lst = append(lst, lmnt)
	}
	return &List{lst}, nil
}

func (bLst *List) Type() ElementType {
	return ListType
}

func (bLst *List) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(LIST_START)
	for _, elm := range bLst.data {
		buff.Write(elm.Encode())
	}
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

func (bLst *List) Val() interface{} {
	return bLst.data
}

type Dictionary struct {
	data map[string]Element
}

func decodeDict(input io.ByteReader) (*Dictionary, error) {
	dict := make(map[string]Element)
	for {
		key, err := Decode(input)
		if err != nil {
			return nil, err
		}
		if key == nil {
			break
		}
		if key.Type() != StringType {
			return nil, InvalidInputError{"expecting string type"}
		}

		val, err := Decode(input)
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, InvalidInputError{"key without value"}
		}
		dict[key.Val().(string)] = val
	}
	return &Dictionary{dict}, nil
}

func (bDct *Dictionary) Type() ElementType {
	return DictType
}

func (bDct *Dictionary) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(DICT_START)
	for k, v := range bDct.data {
		buff.Write((&String{k}).Encode())
		buff.Write(v.Encode())
	}
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

func (bDct *Dictionary) Val() interface{} {
	return bDct.data
}

type InvalidInputError struct {
	msg string
}

func (err InvalidInputError) Error() string {
	return fmt.Sprintf("Invalid Input Error: %s", err.msg)
}
