/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape_test

import (
	"os"
	"testing"

	"go.osspkg.com/casecheck"
	"go.osspkg.com/grape"
	"go.osspkg.com/logx"
	"go.osspkg.com/xc"
)

func TestUnit_NewAppSyslog(t *testing.T) {
	t.Skip()
	configData := `
env: dev
log:
  level: 4
  format: syslog
`
	os.WriteFile("/tmp/TestUnit_NewApp.yaml", []byte(configData), 0755)
	app := grape.New("testapp")
	app.ConfigFile("/tmp/TestUnit_NewApp.yaml")
	app.Modules(func(ctx xc.Context) {
		logx.Info("hello")
		ctx.Close()
	})
	app.Run()
}

func TestUnit_AppInvoke(t *testing.T) {
	out := ""
	call1 := func(s *Struct1) {
		s.Do(&out)
		out += "Done"
	}
	grape.New("testapp").Modules(
		&Struct1{}, &Struct2{},
	).Invoke(call1)
	casecheck.Equal(t, "[Struct1.Do]Done", out)

	out = ""
	call1 = func(s *Struct1) {
		s.Do2(&out)
		out += "Done"
	}
	grape.New("testapp").ExitFunc(func(code int) {
		t.Log("Exit Code", code)
		casecheck.Equal(t, 0, code)
	}).Modules(
		NewStruct1, &Struct2{},
	).Invoke(call1)
	casecheck.Equal(t, "[Struct1.Do][Struct2.Do]Done", out)
}

type Struct1 struct{ s *Struct2 }

func NewStruct1(s2 *Struct2) *Struct1 {
	return &Struct1{s: s2}
}
func (*Struct1) Do(v *string) { *v += "[Struct1.Do]" }
func (s *Struct1) Do2(v *string) {
	*v += "[Struct1.Do]"
	s.s.Do(v)
}

type Struct2 struct{}

func (*Struct2) Do(v *string) { *v += "[Struct2.Do]" }
