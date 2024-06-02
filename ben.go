package ben

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const (
	StartDict       = 'd'
	StartInt        = 'i'
	StartList       = 'l'
	EndItemSeq      = 'e'
	LengthDelimiter = ':'
	tenth           = 10
	digitMask       = 0x30
)

var (
	errEndItemSequence = errors.New("end of item sequence")
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

func (d DefEltVal[T]) Bytes() ([]byte, error) {
	return nil, TypeError{"Element does not implement Bytes"}
}

type TypeError struct {
	msg string
}

func (t TypeError) Error() string {
	return fmt.Sprintf("Type assertion error: %s", t.msg)
}

type ElementType int

func (e ElementType) String() string {
	switch e {
	case IntType:
		return "Int"
	case StringType:
		return "String"
	case ListType:
		return "List"
	case DictType:
		return "Dictionary"
	}

	return "[Invalid]"
}

const (
	IntType ElementType = iota
	StringType
	ListType
	DictType
)

type ReadPeeker interface {
	io.ByteReader
	io.Reader
	Peek(int) ([]byte, error)
}

type ElementValues interface {
	Bytes() ([]byte, error)
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
	Decode(ReadPeeker) (T, error)

	Element
}

func Decode[B Bencoder[B]](input ReadPeeker) (B, error) {
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

func (i Integer) Into() int64 {
	return i.Val
}

func (i Integer) Type() ElementType {
	return IntType
}

func (i Integer) Decode(input ReadPeeker) (Integer, error) {
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

	if ch != StartInt {
		return i, InvalidInputError{"input is not Integer"}
	}

	for {
		ch, err = input.ReadByte()
		if err != nil {
			return i, err
		}
		if ch == EndItemSeq {
			break
		}
		if ch == '-' && rInt == 0 {
			minus = true
			continue
		}

		it := int(ch) - digitMask
		if 0 <= it && it <= 9 {
			rInt = rInt*int64(tenth) + int64(it)
			continue
		}

		return i, InvalidInputError{strconv.Itoa(int(ch))}
	}

	if minus {
		rInt = -rInt
	}

	return Int(rInt), nil
}

func (i Integer) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(StartInt)
	buff.WriteString(strconv.FormatInt(i.Val, 10))
	buff.WriteByte(EndItemSeq)
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

func (s String) Bytes() ([]byte, error) {
	return []byte(s.Val), nil
}

func (s String) TryFrom(e Element) (String, error) {
	return e.String()
}

func (s String) Into() string {
	return s.Val
}

func (s String) Type() ElementType {
	return StringType
}

func (s String) Decode(input ReadPeeker) (String, error) {
	var (
		buff   bytes.Buffer
		length []byte
		err    error
		sLen   int64
		ch     byte
	)

	firstToken, err := input.ReadByte()
	if err != nil {
		return s, err
	}

	length = append(length, firstToken)

	for {
		ch, err = input.ReadByte()
		if err != nil {
			return s, err
		}
		if ch == LengthDelimiter {
			break
		}
		length = append(length, ch)
	}

	if sLen, err = strconv.ParseInt(string(length), 10, 64); err != nil {
		return s, err
	}

	if _, err = io.CopyN(&buff, input, sLen); err != nil {
		return s, err
	}

	return Str(buff.String()), nil
}

func (s String) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteString(strconv.Itoa(len(s.Val)))
	buff.WriteByte(LengthDelimiter)
	buff.WriteString(s.Val)
	return buff.Bytes()
}

func InferredTypeDecode(input ReadPeeker) (Element, error) {
	currentToken, err := input.Peek(1)
	if err != nil {
		return nil, err
	}

	switch currentToken[0] {
	case StartDict:
		return Decode[Dictionary](input)
	case StartInt:
		return Decode[Integer](input)
	case StartList:
		return Decode[List](input)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return Decode[String](input)
	case EndItemSeq:
		return nil, errEndItemSequence
	default:
		return nil, InvalidInputError{"should not reach here"}
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

func (l List) Decode(input ReadPeeker) (List, error) {
	var (
		lst []Element
		err error
		ch  byte
	)

	ch, err = input.ReadByte()
	if err != nil {
		return l, err
	}

	if ch != StartList {
		return l, InvalidInputError{"not a list type"}
	}

	for {
		var lmnt Element
		lmnt, err = InferredTypeDecode(input)
		if err != nil && !errors.Is(err, errEndItemSequence) {
			return Lst(lst), err
		}
		if lmnt == nil { // end of sequence
			break
		}
		lst = append(lst, lmnt)
	}
	return Lst(lst), nil
}

func (l List) Type() ElementType {
	return ListType
}

func (l List) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(StartList)
	for _, elm := range l.Val {
		buff.Write(elm.Encode())
	}
	buff.WriteByte(EndItemSeq)
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

func (d Dictionary) Decode(input ReadPeeker) (Dictionary, error) {
	var (
		err      error
		testPeek []byte
		ch       byte
		key      String
		val      Element
	)

	dict := make(map[string]Element)

	ch, err = input.ReadByte()
	if err != nil {
		return d, err
	}

	if ch != StartDict {
		return d, InvalidInputError{"not a dictionary"}
	}

	for {
		testPeek, err = input.Peek(1)
		if err != nil {
			return d, err
		}

		if testPeek[0] == EndItemSeq {
			break
		}

		key, err = Decode[String](input)
		if err != nil {
			return d, err
		}

		val, err = InferredTypeDecode(input)
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

func (d Dictionary) Type() ElementType {
	return DictType
}

func (d Dictionary) Encode() []byte {
	var buff bytes.Buffer
	buff.WriteByte(StartDict)
	for k, v := range d.Val {
		buff.Write(Str(k).Encode())
		buff.Write(v.Encode())
	}
	buff.WriteByte(EndItemSeq)
	return buff.Bytes()
}

type InvalidInputError struct {
	msg string
}

func (err InvalidInputError) Error() string {
	return fmt.Sprintf("Invalid Input Error: %s", err.msg)
}
