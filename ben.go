package ben

import (
	"bufio"
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
}

type Bencoder[T Element] interface {
	Decode(*bufio.Reader) (T, error)
	Encode() []byte
	Type() ElementType
}

func Decode[B Bencoder[B]](input *bufio.Reader) (B, error) {
	var b B
	return b.Decode(input)
}

type Integer int64

func NewInteger(i int64) Integer {
	return Integer(i)
}

func (i Integer) TryFrom(e Element) (Integer, error) {
	if e.Type() != i.Type() {
		return i, fmt.Errorf("not an integer")
	}
	return e.(Integer), nil
}

func (i Integer) Type() ElementType {
	return IntType
}

func (i Integer) Decode(input *bufio.Reader) (Integer, error) {
	var (
		ch    byte
		rInt  int64
		err   error
		minus bool
	)
	ch, err = input.ReadByte()
	if err != nil {
		return i, err
	}

	if ch != INT_START {
		return i, fmt.Errorf("input is not Integer")
	}

	for {
		ch, err = input.ReadByte()
		if err != nil {
			return i, err
		}
		if ch == SEQ_END {
			break
		}
		if ch == '-' && rInt == 0 {
			minus = true
			continue
		}

		it := int(ch) - 0x30
		if 0 <= it && it <= 9 {
			rInt = rInt*int64(10) + int64(it)
			continue
		}

		return i, InvalidInputError{strconv.Itoa(int(ch))}
	}

	if minus {
		rInt = -rInt
	}

	return Integer(rInt), nil
}

func (bInt Integer) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(INT_START)
	buff.WriteString(strconv.FormatInt(int64(bInt), 10))
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

type String string

func NewString(input string) String {
	return String(input)
}

func (s String) TryFrom(e Element) (String, error) {
	if e.Type() != s.Type() {
		return s, fmt.Errorf("not a string")
	}
	return e.(String), nil
}

func (s String) Type() ElementType {
	return StringType
}

func (s String) Decode(input *bufio.Reader) (String, error) {
	var (
		buff   bytes.Buffer
		length []byte
		err    error
		sLen   int64
	)

	first_token, err := input.ReadByte()
	if err != nil {
		return s, err
	}

	length = append(length, first_token)

	for {
		ch, err := input.ReadByte()
		if err != nil {
			return s, err
		}
		if ch == LENGTH_DELIMITER {
			break
		}
		length = append(length, ch)
	}

	if sLen, err = strconv.ParseInt(string(length), 10, 64); err != nil {
		return s, err
	}

	if _, err := io.CopyN(&buff, input, sLen); err != nil {
		return s, err
	}

	return String(buff.String()), nil
}

func (bStr String) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteString(strconv.Itoa(len(bStr)))
	buff.WriteByte(LENGTH_DELIMITER)
	buff.WriteString(string(bStr))
	return buff.Bytes()
}

func InferredTypeDecode(input *bufio.Reader) (Element, error) {
	current_token, err := input.Peek(1)
	if err != nil {
		return nil, err
	}

	switch current_token[0] {
	case DICT_START:
		return Decode[Dictionary](input)
	case INT_START:
		return Decode[Integer](input)
	case LIST_START:
		return Decode[List](input)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return Decode[String](input)
	case SEQ_END:
		return nil, nil
	default:
		return nil, InvalidInputError{}
	}
}

type List []Element

func NewList(input []Element) List {
	return List(input)
}

func (l List) TryFrom(e Element) (List, error) {
	if e.Type() != l.Type() {
		return l, fmt.Errorf("not a list")
	}
	return e.(List), nil
}

func (l List) Decode(input *bufio.Reader) (List, error) {
	var (
		lst []Element
		err error
		ch  byte
	)

	ch, err = input.ReadByte()
	if err != nil {
		return l, err
	}

	if ch != LIST_START {
		return l, fmt.Errorf("not a list type")
	}

	for {
		var lmnt Element
		lmnt, err = InferredTypeDecode(input)
		if err != nil {
			return List(lst), err
		}
		if lmnt == nil { // end of sequence
			break
		}
		lst = append(lst, lmnt)
	}
	return List(lst), err
}

func (bLst List) Type() ElementType {
	return ListType
}

func (bLst List) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(LIST_START)
	for _, elm := range bLst {
		buff.Write(elm.Encode())
	}
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

type Dictionary map[string]Element

func NewDictionary(input map[string]Element) Dictionary {
	return Dictionary(input)
}

func (d Dictionary) TryFrom(e Element) (Dictionary, error) {
	if e.Type() != d.Type() {
		return d, fmt.Errorf("not a dictionary")
	}
	return e.(Dictionary), nil
}

func (d Dictionary) Decode(input *bufio.Reader) (Dictionary, error) {
	dict := make(map[string]Element)
	ch, err := input.ReadByte()
	if err != nil {
		return d, err
	}

	if ch != DICT_START {
		return d, fmt.Errorf("not a dictionary")
	}

	for {
		testPeek, err := input.Peek(1)
		if err != nil {
			return nil, err
		}

		if testPeek[0] == SEQ_END {
			break
		}

		key, err := Decode[String](input)
		if err != nil {
			return nil, err
		}

		val, err := InferredTypeDecode(input)
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, InvalidInputError{"key without value"}
		}
		dict[string(key)] = val
	}

	return Dictionary(dict), nil
}

func (bDct Dictionary) Type() ElementType {
	return DictType
}

func (bDct Dictionary) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(DICT_START)
	for k, v := range bDct {
		buff.Write(String(k).Encode())
		buff.Write(v.Encode())
	}
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

type InvalidInputError struct {
	msg string
}

func (err InvalidInputError) Error() string {
	return fmt.Sprintf("Invalid Input Error: %s", err.msg)
}
