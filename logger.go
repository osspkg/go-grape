/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import (
	"os"

	"go.osspkg.com/grape/config"
	"go.osspkg.com/logx"
)

type _log struct {
	file    *os.File
	handler logx.Logger
	conf    config.LogConfig
}

func newLog(conf config.LogConfig) *_log {
	file, err := os.OpenFile(conf.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return &_log{file: file, conf: conf}
}

func (v *_log) Handler(l logx.Logger) {
	v.handler = l
	v.handler.SetOutput(v.file)
	v.handler.SetLevel(v.conf.Level)
	switch v.conf.Format {
	case "string":
		v.handler.SetFormatter(logx.NewFormatString())
	case "json":
		v.handler.SetFormatter(logx.NewFormatJSON())
	}
}

func (v *_log) Close() error {
	if v.handler != nil {
		v.handler.Close()
	}
	return v.file.Close()
}
