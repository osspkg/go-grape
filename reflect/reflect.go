/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package reflect

import (
	"fmt"
	"reflect"
	"strings"
)

const ErrorName = "error"

var errType = reflect.TypeOf(new(error)).Elem()

// nolint: gocyclo
func GetAddress(t reflect.Type, v interface{}) (string, bool) {
	if t == nil {
		return "nil", false
	}
	switch t.Kind() {
	case reflect.Func:
		if len(t.PkgPath()) > 0 {
			return reflectAddressElem(t), true
		}
		if v == nil {
			return t.String(), false
		}
		p := reflect.ValueOf(v).Pointer()
		return fmt.Sprintf("0x%x.%s", p, t.String()), true
	case reflect.Ptr:
		if t.Implements(errType) {
			return ErrorName, false
		}
		value := reflectAddressElem(t)
		if value == "*" {
			return "*struct{}", false
		}
		return value, true
	case reflect.Map:
		key := reflectAddressElem(t.Key())
		value := reflectAddressElem(t.Elem())
		return fmt.Sprintf("map[%s]%s", key, value), isNotSimple(key) && isNotSimple(value)
	case reflect.Struct:
		value := reflectAddressElem(t)
		if len(value) == 0 {
			return "struct{}", false
		}
		return value, true
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String,
		reflect.Interface:
		value := reflectAddressElem(t)
		return value, isNotSimple(value)
	case reflect.Chan:
		value, _ := GetAddress(t.Elem(), v)
		return fmt.Sprintf("chan %s", value), isNotSimple(value)
	case reflect.Slice:
		value := reflectAddressElem(t.Elem())
		return fmt.Sprintf("[]%s", value), isNotSimple(value)
	case reflect.Array:
		value := reflectAddressElem(t.Elem())
		return fmt.Sprintf("[%d]%s", t.Len(), value), isNotSimple(value)
	}
	return t.String(), false
}

func TypingPtr(vv []interface{}, call func(interface{}) error) ([]interface{}, error) {
	result := make([]interface{}, 0, len(vv))
	for _, v := range vv {
		ref := reflect.TypeOf(v)
		switch ref.Kind() {
		case reflect.Struct:
			in := reflect.New(ref).Interface()
			if err := call(in); err != nil {
				return nil, err
			}
			rv := reflect.ValueOf(in).Elem().Interface()
			result = append(result, rv)
		case reflect.Ptr:
			if err := call(v); err != nil {
				return nil, err
			}
			result = append(result, v)
		default:
			return nil, fmt.Errorf("supported type [%T]", v)
		}
	}
	return result, nil
}

func reflectAddressElem(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return "*" + reflectAddressElem(t.Elem())
	}
	value := t.Name()
	if len(t.PkgPath()) > 0 {
		value = t.PkgPath() + "." + value
	}
	return value
}

func isNotSimple(v string) bool {
	return strings.Contains(v, ".")
}
