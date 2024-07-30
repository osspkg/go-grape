/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package reflect

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"go.osspkg.com/casecheck"
)

// nolint: lll
func TestUnit_getReflectAddress2(t *testing.T) {
	type (
		aa string
		bb struct{}
		cc interface {
			AA() string
		}
		ff func(_ string) bool
	)
	var (
		fn1 ff = func(_ string) bool { return false }
		fn2    = func(_ string) bool { return false }
	)

	tests := []struct {
		name string
		obj  interface{}
		want string
		ok   bool
	}{
		{name: "Case01", obj: 0, want: "int", ok: false},
		{name: "Case02", obj: "0", want: "string", ok: false},
		{name: "Case03", obj: true, want: "bool", ok: false},
		{name: "Case04", obj: aa("aaa"), want: "go\\.osspkg\\.com\\/grape\\/reflect\\.aa", ok: true},
		{name: "Case05", obj: func() *aa { a := aa("aaa"); return &a }(), want: "\\*go\\.osspkg\\.com\\/grape\\/reflect\\.aa", ok: true},
		{name: "Case06", obj: func() *aa { a := aa("aaa"); return &a }, want: "0x([0-9a-z]+)\\.func\\(\\) \\*reflect\\.aa", ok: true},
		{name: "Case07", obj: fn1, want: "go\\.osspkg\\.com\\/grape\\/reflect\\.ff", ok: true},
		{name: "Case08", obj: fn2, want: "0x([0-9a-z]+)\\.func\\(string\\) bool", ok: true},
		{name: "Case09", obj: 3.14, want: "float64", ok: false},
		{name: "Case10", obj: errors.New(""), want: "error", ok: false},
		{name: "Case11", obj: []string{}, want: "\\[\\]string", ok: false},
		{name: "Case12", obj: []*string{}, want: "\\[\\]\\*string", ok: false},
		{name: "Case13", obj: bb{}, want: "go\\.osspkg\\.com\\/grape\\/reflect\\.bb", ok: true},
		{name: "Case14", obj: &bb{}, want: "\\*go\\.osspkg\\.com\\/grape\\/reflect\\.bb", ok: true},
		{name: "Case15", obj: struct{}{}, want: "struct\\{\\}", ok: false},
		{name: "Case16", obj: &struct{}{}, want: "\\*struct\\{\\}", ok: false},
		{name: "Case17", obj: cc(nil), want: "nil", ok: false},
		{name: "Case18", obj: []cc{}, want: "\\[\\]go\\.osspkg\\.com\\/grape\\/reflect\\.cc", ok: true},
		{name: "Case19", obj: []*cc{}, want: "\\[\\]\\*go\\.osspkg\\.com\\/grape\\/reflect\\.cc", ok: true},
		{name: "Case20", obj: map[cc]cc{}, want: "map\\[go\\.osspkg\\.com\\/grape\\/reflect\\.cc\\]go\\.osspkg\\.com\\/grape\\/reflect\\.cc", ok: true},
		{name: "Case21", obj: map[*cc]*cc{}, want: "map\\[\\*go\\.osspkg\\.com\\/grape\\/reflect\\.cc\\]\\*go\\.osspkg\\.com\\/grape\\/reflect\\.cc", ok: true},
		{name: "Case22", obj: make(chan string), want: "chan string", ok: false},
		{name: "Case23", obj: make(chan *string, 100), want: "chan \\*string", ok: false},
		{name: "Case24", obj: make(chan struct{}, 100), want: "chan struct\\{\\}", ok: false},
		{name: "Case25", obj: [2]int{1}, want: "\\[2\\]int", ok: false},
		{name: "Case26", obj: [2]*cc{}, want: "\\[2\\]\\*go\\.osspkg\\.com\\/grape\\/reflect\\.cc", ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetAddress(reflect.TypeOf(tt.obj), tt.obj)
			fmt.Println(got, ok)
			if !regexp.MustCompile("(U?)^" + tt.want + "$").MatchString(got) {
				t.Errorf("getReflectAddress() = %v, want %v", got, tt.want)
			}
			if ok != tt.ok {
				t.Errorf("getReflectAddress() = %v, want %v", ok, tt.ok)
			}
		})
	}

	var (
		fn3 = func(_ cc) {}
	)
	fn3ref := reflect.TypeOf(fn3)

	fn3ref0 := fn3ref.In(0)
	got, ok := GetAddress(fn3ref0, nil)
	fmt.Println(got, ok)
	casecheck.Equal(t, "go.osspkg.com/grape/reflect.cc", got)
	casecheck.True(t, ok)
}
