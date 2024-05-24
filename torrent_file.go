package ben

import (
	"errors"
	"reflect"
	"strings"
	"time"
)

type Torrent struct {
	Announce     string     `ben:"announce"`
	Info         Info       `ben:"info"`
	CreatedBy    *string    `ben:"created by,omitempty"`
	CreationDate *time.Time `ben:"creation date,omitempty"`
	Encoding     *string    `ben:"encoding,omitempty"`
}

type Info struct {
	Name        string `ben:"name"`
	Length      int64  `ben:"length"`
	PieceLength int64  `ben:"piece length"`
	Pieces      []SHA1 `ben:"pieces"`
	Files       []File `ben:"files,omitempty"`
}

type File struct {
	Length int64    `ben:"length"`
	Path   []string `ben:"path"`
}

type SHA1 []byte

type valueSetterMap map[reflect.Kind]valueSetterFunc
type valueSetterFunc func(valueSetterMap, reflect.Value, Element) error

var (
	setStructValue = map[string]valueSetterFunc{
		"time.Time": func(setter valueSetterMap, obj reflect.Value, l Element) error {
			// We will assume all time.Time data was serialized to UNIX epoch time.
			v, err := l.Integer()
			if err != nil {
				return err
			}

			timeVal := time.Unix(v.Into(), 0)

			obj.Set(reflect.ValueOf(timeVal))

			return nil
		},

		"github.com/fudanchii/ben.Info": func(setter valueSetterMap, obj reflect.Value, l Element) error {
			return nil
		},
	}

	setValue = valueSetterMap{
		reflect.Int64: func(_ valueSetterMap, obj reflect.Value, l Element) error {
			val, err := l.Integer()
			if err != err {
				return err
			}

			obj.SetInt(val.Into())

			return nil
		},

		reflect.String: func(_ valueSetterMap, obj reflect.Value, l Element) error {
			val, err := l.String()
			if err != nil {
				return err
			}

			obj.SetString(val.Into())

			return nil
		},

		reflect.Struct: func(setter valueSetterMap, obj reflect.Value, l Element) error {
			objType := obj.Type()
			fqTypeName := objType.PkgPath() + "." + objType.Name()

			structValueSetter, ok := setStructValue[fqTypeName]

			if !ok {
				return errTypeNotSupported
			}

			return structValueSetter(setter, obj, l)
		},

		reflect.Pointer: func(setter valueSetterMap, obj reflect.Value, l Element) error {
			if obj.IsNil() {
				obj.Set(reflect.New(obj.Type().Elem()))
			}

			setterFunc, ok := setter[obj.Type().Elem().Kind()]

			if !ok {
				return errTypeNotSupported
			}

			return setterFunc(setter, obj.Elem(), l)
		},
	}

	errTypeNotSupported = errors.New("This type is not supported.")
)

func (t Torrent) TryFrom(d Dictionary) (Torrent, error) {
	torrentType := reflect.TypeOf(t)
	torrentStruct := reflect.ValueOf(&t).Elem()

	numField := torrentType.NumField()

	for i := 0; i < numField; i++ {
		field := torrentType.Field(i)
		fieldVal := torrentStruct.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := strings.SplitN(field.Tag.Get("ben"), ",", 2)
		val := d.Val[tag[0]]

		setterFunc, ok := setValue[fieldVal.Kind()]
		if !ok {
			return t, errTypeNotSupported
		}
		if err := setterFunc(setValue, fieldVal, val); err != nil {
			return t, err
		}
	}

	return t, nil
}
