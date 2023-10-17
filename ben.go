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

type DefEltVal[T any] struct {
	Val T
}

func V[T any](v T) DefEltVal[T] {
	return DefEltVal[T]{v}
}

func (d DefEltVal[T]) String() (String, error) {
	var s String
	return s, TypeError{"Element is not a String"}
}

func (d DefEltVal[T]) Integer() (Integer, error) {
	var i Integer
	return i, TypeError{"Element is not an Integer"}
}

func (d DefEltVal[T]) List() (List, error) {
	var l List
	return l, TypeError{"Element is not a List"}
}

func (d DefEltVal[T]) Dictionary() (Dictionary, error) {
	var di Dictionary
	return di, TypeError{"Element is not a Dictionary"}
}

type TypeError struct {
	msg string
}

func (t TypeError) Error() string {
	return fmt.Sprintf("Type assertion error: %s", t.msg)
}

type ElementType int

const (
	IntType ElementType = iota
	StringType
	ListType
	DictType
)

type ElementValues interface {
	String() (String, error)
	Integer() (Integer, error)
	List() (List, error)
	Dictionary() (Dictionary, error)
}

type Element interface {
	Type() ElementType
	Encode() []byte

	ElementValues
}

type Bencoder[T Element] interface {
	Decode(*bufio.Reader) (T, error)

	Element
}

func Decode[B Bencoder[B]](input *bufio.Reader) (B, error) {
	var b B
	return b.Decode(input)
}

type Integer struct {
	DefEltVal[int64]
}

func Int(v int64) Integer {
	return Integer{V(v)}
}

func (i Integer) Integer() (Integer, error) {
	return i, nil
}

func (i Integer) TryFrom(e Element) (Integer, error) {
	return e.Integer()
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
		return i, InvalidInputError{"input is not Integer"}
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

	return Int(rInt), nil
}

func (bInt Integer) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(INT_START)
	buff.WriteString(strconv.FormatInt(bInt.Val, 10))
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

type String struct {
	DefEltVal[string]
}

func Str(v string) String {
	return String{V(v)}
}

func (s String) String() (String, error) {
	return s, nil
}

func (s String) TryFrom(e Element) (String, error) {
	return e.String()
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

	return Str(buff.String()), nil
}

func (bStr String) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteString(strconv.Itoa(len(bStr.Val)))
	buff.WriteByte(LENGTH_DELIMITER)
	buff.WriteString(bStr.Val)
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

type List struct {
	DefEltVal[[]Element]
}

func Lst(elt []Element) List {
	return List{V(elt)}
}

func (l List) List() (List, error) {
	return l, nil
}

func (l List) TryFrom(e Element) (List, error) {
	return e.List()
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
		return l, InvalidInputError{"not a list type"}
	}

	for {
		var lmnt Element
		lmnt, err = InferredTypeDecode(input)
		if err != nil {
			return Lst(lst), err
		}
		if lmnt == nil { // end of sequence
			break
		}
		lst = append(lst, lmnt)
	}
	return Lst(lst), err
}

func (bLst List) Type() ElementType {
	return ListType
}

func (bLst List) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(LIST_START)
	for _, elm := range bLst.Val {
		buff.Write(elm.Encode())
	}
	buff.WriteByte(SEQ_END)
	return buff.Bytes()
}

type Dictionary struct {
	DefEltVal[map[string]Element]
}

func Dct(m map[string]Element) Dictionary {
	return Dictionary{V(m)}
}

func (d Dictionary) Dictionary() (Dictionary, error) {
	return d, nil
}

func (d Dictionary) TryFrom(e Element) (Dictionary, error) {
	return e.Dictionary()
}

func (d Dictionary) Decode(input *bufio.Reader) (Dictionary, error) {
	dict := make(map[string]Element)
	ch, err := input.ReadByte()
	if err != nil {
		return d, err
	}

	if ch != DICT_START {
		return d, InvalidInputError{"not a dictionary"}
	}

	for {
		testPeek, err := input.Peek(1)
		if err != nil {
			return d, err
		}

		if testPeek[0] == SEQ_END {
			break
		}

		key, err := Decode[String](input)
		if err != nil {
			return d, err
		}

		val, err := InferredTypeDecode(input)
		if err != nil {
			return d, err
		}
		if val == nil {
			return d, InvalidInputError{"key without value"}
		}
		dict[key.Val] = val
	}

	return Dct(dict), nil
}

func (bDct Dictionary) Type() ElementType {
	return DictType
}

func (bDct Dictionary) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(DICT_START)
	for k, v := range bDct.Val {
		buff.Write(Str(k).Encode())
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
