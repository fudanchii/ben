package ben

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/fudanchii/infr"
)

type Torrent struct {
	Announce     string     `ben:"announce"`
	Info         Info       `ben:"info"`
	CreatedBy    *string    `ben:"created by,omitempty"`
	CreationDate *time.Time `ben:"creation date,omitempty"`
	Encoding     *string    `ben:"encoding,omitempty"`
}

func (t Torrent) TryFrom(d Dictionary) (Torrent, error) {
	return castFromDictionaryInto[Torrent](d)
}

type Info struct {
	Name        string `ben:"name"`
	Length      int64  `ben:"length"`
	PieceLength int64  `ben:"piece length"`
	Pieces      []SHA1 `ben:"pieces"`
	Files       []File `ben:"files,omitempty"`
}

func (Info) TryFrom(d Dictionary) (Info, error) {
	return castFromDictionaryInto[Info](d)
}

type File struct {
	Length int64    `ben:"length"`
	Path   []string `ben:"path"`
}

func (File) TryFrom(d Dictionary) (File, error) {
	return castFromDictionaryInto[File](d)
}

type SHA1 []byte

type valueSetterMap map[reflect.Kind]valueSetterFunc

type valueSetterFunc func(valueSetterMap, reflect.Value, Element) error

func timeTimeStructSetter(_ valueSetterMap, obj reflect.Value, l Element) error {
	// We will assume all time.Time data was serialized to UNIX epoch time.
	v, err := l.Integer()
	if err != nil {
		return err
	}

	timeVal := time.Unix(v.Into(), 0)

	obj.Set(reflect.ValueOf(timeVal))

	return nil
}

func benSHA1StructSetter(_ valueSetterMap, obj reflect.Value, l Element) error {
	lval, err := l.String()
	if err != nil {
		return err
	}

	hashes := []byte(lval.Into())
	hashesCount := len(hashes) / 20 // 160 / 8 bytes = 20

	start := 0
	if obj.Type().Kind() == reflect.Slice {
		obj.Set(reflect.MakeSlice(obj.Type(), hashesCount, hashesCount))
		for i := range hashesCount {
			obj.Index(i).Set(reflect.ValueOf(hashes[start : start+20]))
			start += 20
		}
	}

	return nil
}

func setStructValue() map[string]valueSetterFunc {
	return map[string]valueSetterFunc{
		"time.Time":                     timeTimeStructSetter,
		"github.com/fudanchii/ben.Info": localStructSetter,
		"github.com/fudanchii/ben.File": localStructSetter,
		"github.com/fudanchii/ben.SHA1": benSHA1StructSetter,
	}
}

func setValueForInt64(_ valueSetterMap, obj reflect.Value, l Element) error {
	val, err := l.Integer()
	if err != nil {
		return err
	}

	obj.SetInt(val.Into())

	return nil
}

func setValueForString(_ valueSetterMap, obj reflect.Value, l Element) error {
	val, err := l.String()
	if err != nil {
		return err
	}

	obj.SetString(val.Into())

	return nil
}

func setValueForStruct(setter valueSetterMap, obj reflect.Value, l Element) error {
	objType := obj.Type()
	fqTypeName := fullyQualifiedTypeName(objType)

	structValueSetter, ok := setStructValue()[fqTypeName]

	if !ok {
		return errTypeNotSupported
	}

	return structValueSetter(setter, obj, l)
}

func setValueForPointer(setter valueSetterMap, obj reflect.Value, l Element) error {
	if obj.IsNil() {
		obj.Set(reflect.New(obj.Type().Elem()))
	}

	fqTypeName := fullyQualifiedTypeName(obj.Type().Elem())
	setterFunc, ok := setStructValue()[fqTypeName]
	if !ok {
		setterFunc, ok = setter[obj.Type().Elem().Kind()]
		if !ok {
			return errTypeNotSupported
		}
	}

	return setterFunc(setter, obj.Elem(), l)
}

func setValueForSlice(setter valueSetterMap, obj reflect.Value, l Element) error {
	if obj.IsNil() {
		obj.Set(reflect.MakeSlice(obj.Type(), 0, 0))
	}

	// slice has a very specific behavior:
	//   - []byte handled as one assignment, treated similar to string
	//   - other elem type will be assigned in an iteration
	//   - nested slice will be recursed, but still assigned under iteration
	fqTypeName := fullyQualifiedTypeName(obj.Type().Elem())
	setterFunc, ok := setStructValue()[fqTypeName]
	if ok {
		return setterFunc(setter, obj, l)
	}

	elemType := obj.Type().Elem().Kind()

	//nolint: exhaustive // already covered by default hand
	switch elemType {
	// []byte
	case reflect.Uint8:
		return setter[reflect.String](setter, obj, l)

	case reflect.Invalid:
		return errors.New("ben/list: unexpected invalid type for list element")

	default:
		list, err := l.List()
		if err != nil {
			return err
		}

		for idx := range list.Val {
			setterErr := setter[elemType](setter, obj.Index(idx), list.Val[idx])
			if setterErr != nil {
				return setterErr
			}
		}
	}

	return nil
}

func setValue() valueSetterMap {
	//nolint: exhaustive // no need to cover all types
	return valueSetterMap{
		reflect.Int64:   setValueForInt64,
		reflect.String:  setValueForString,
		reflect.Struct:  setValueForStruct,
		reflect.Pointer: setValueForPointer,
		reflect.Slice:   setValueForSlice,
	}
}

var (
	errTypeNotSupported = errors.New("this type is not supported")
)

func localStructSetter(_ valueSetterMap, obj reflect.Value, l Element) error {
	dict, err := l.Dictionary()
	if err != nil {
		return err
	}

	info, err := infr.TryFrom[Dictionary, Info](dict).TryInto()
	if err != nil {
		return err
	}

	obj.Set(reflect.ValueOf(info))
	return nil
}

func castFromDictionaryInto[T infr.TryFromType[Dictionary, T]](dict Dictionary) (T, error) {
	var t T

	objType := reflect.TypeOf(t)
	objStruct := reflect.ValueOf(&t).Elem()

	numField := objType.NumField()

	for i := range numField {
		field := objType.Field(i)
		fieldVal := objStruct.Field(i)

		if !field.IsExported() {
			continue
		}

		fqTypeName := fullyQualifiedTypeName(field.Type)
		setterFunc, ok := setStructValue()[fqTypeName]

		if !ok {
			setterFunc, ok = setValue()[fieldVal.Kind()]
			if !ok {
				return t, errTypeNotSupported
			}
		}

		tag := strings.SplitN(field.Tag.Get("ben"), ",", 2)
		val, present := dict.Val[tag[0]]

		if !present && tag[1] == "omitempty" {
			continue
		}

		if err := setterFunc(setValue(), fieldVal, val); err != nil {
			return t, err
		}
	}

	return t, nil
}

func fullyQualifiedTypeName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}
