package mapto

import (
	"encoding/json"
	"errors"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"time"
	"regexp"
	"fmt"
)

// Enum simulation. Methods what golang enum types(ints..) need to implement to allow converting from/to string
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

// Mark structs what should init their fields to some defaults other than zero-values before mapping
type Initializable interface {
	Init() error
}

// Struct constructors
type Constructor func(Type reflect.Type, val interface{}) (interface{}, bool, error)

// Global type qualifier to contructor dictionary
var typeConstructors = map[string]Constructor{}

// Simple struct constructor what create specified struct type and then fill fields from raw map and optionally cal Init method
func StructConstructor(structType interface{}) Constructor {
	st := reflect.TypeOf(structType)
	return func(Type reflect.Type, val interface{}) (interface{}, bool, error) {
		if st.AssignableTo(Type) {
			v, b := reflect.New(st.Elem()).Interface(), true
			initializablev, ok := v.(Initializable)
			if ok {
				err := initializablev.Init()
				if err != nil {
					return nil, false, err
				}
			}
			return v, b, nil
		}
		return val, false, nil
	}
}

// Register type qualifier
func RegisterConstructor(qualifier string, constructor Constructor) {
	typeConstructors[qualifier] = constructor
}

func GetConstructor(qualifier string) (Constructor, bool) {
	c, ok := typeConstructors[qualifier]
	return c, ok
}

var durationType = reflect.TypeOf(time.Hour)
var enumType = reflect.TypeOf((*Enum)(nil)).Elem()
var regexpType = reflect.TypeOf(&regexp.Regexp{})

func decodeHook(a reflect.Type, b reflect.Type, c interface{}) (interface{}, error) {
	obj, ok := c.(map[string]interface{})

	if ok {
		qualifierIface, ok := obj["@type"]
		if ok {
			qualifier, ok := qualifierIface.(string)
			if ok {
				constr, ok := GetConstructor(qualifier)
				if ok {
					res, fill, err := constr(b, c)

					if err != nil {
						return nil, err
					}
					resType := reflect.TypeOf(res)
					if !resType.AssignableTo(b) {
						return nil, errors.New(fmt.Sprintf("Type '%s' from qualifier '%s' not assignable to '%s'", resType.String(), qualifier, b.String()));
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
					return nil, errors.New("No constructor for qualifier " + qualifier)
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
		} else if b.AssignableTo(regexpType) {
			return regexp.Compile(str)
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
