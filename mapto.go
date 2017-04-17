package mapto

import (
	"github.com/mitchellh/mapstructure"
	"reflect"
	"errors"
	"time"
	"encoding/json"
)
// methods what enum types(ints..) need to implement to allow converting from/to string
type Enum interface {
	FromString(val string) bool
	String() string
}
// helper funcs to easy implement json unmarshal for enum types
func EnumUnmarshalJSON(e Enum, b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	ok := e.FromString(s)
	if !ok {
		return errors.New("Illegal enum value: " + s)
	}
	return nil
}


// Struct constructors
type Constructor func(Type reflect.Type, val interface{}) (interface{}, bool, error)
var typeConstructors = map[string]Constructor{}

// Simple struct constructor what create specified struct type and then fill fields from raw map
func StructConstructor(structType interface{}) Constructor {
	st := reflect.TypeOf(structType)
	return func(Type reflect.Type, val interface{}) (interface{}, bool, error) {
		if st.AssignableTo(Type) {
			v, b := reflect.New(st.Elem()).Interface(), true
			return v, b, nil
		}
		return val, false, nil
	}
}

func RegisterConstructor(classifier string, constructor Constructor) {
	typeConstructors[classifier] = constructor
}

func getConstructor(classifier string) (Constructor, bool) {
	c, ok := typeConstructors[classifier]
	return c, ok
}

var durationType = reflect.TypeOf(time.Hour)
var enumType = reflect.TypeOf((*Enum)(nil)).Elem()

func decodeHook(a reflect.Type, b reflect.Type, c interface{}) (interface{}, error) {
	obj, ok := c.(map[string]interface{})

	if ok {
		classifierIface, ok := obj["@type"]
		if ok {
			classifier, ok := classifierIface.(string)
			if ok {
				constr, ok := getConstructor(classifier)
				if ok {
					res, fill, err := constr(b, c)

					if err != nil {
						return nil, err
					}
					if fill {
						delete(obj, "@type")
						err = Decode(c, res)
						if err != nil {
							return nil, err
						}
					}
					return res, nil
				} else {
					return nil, errors.New("No constructor for classifier " + classifier)
				}
			}
		}
	} else if str, ok := c.(string); ok {

		if b.AssignableTo(durationType) {
			return time.ParseDuration(str)
		} else if bT := reflect.New(b); bT.Type().Implements(enumType) {
			e, ok := bT.Interface().(Enum)
			if ok {
				ok := e.FromString(str)
				if !ok {
					return nil, errors.New("Invalid enum value: " + str)
				}

				return reflect.ValueOf(e).Elem().Interface(), nil
			}
		}
	}
	return c, nil
}

func Decode(from interface{}, to interface{}) error {
	dc := &mapstructure.DecoderConfig{Metadata: &mapstructure.Metadata{}, DecodeHook: decodeHook, Result: to}
	dec, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}

	return dec.Decode(from)
}