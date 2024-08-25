/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import (
	"io"
	"log/syslog"
	"net/url"
	"os"

	"go.osspkg.com/console"
	"go.osspkg.com/grape/config"
	"go.osspkg.com/logx"
)

type _log struct {
	file    io.WriteCloser
	handler logx.Logger
	conf    config.LogConfig
}

func newLog(tag string, conf config.LogConfig) *_log {
	var err error
	object := &_log{
		conf: conf,
	}
	switch conf.Format {
	case "syslog":
		defer func() {
			if p := recover(); p != nil {
				console.Fatalf("logger panic [type=%s filepath=%s]: %v", conf.Format, conf.FilePath, p)
			}
		}()
		network, addr := "", ""
		if uri, err0 := url.Parse(conf.FilePath); err0 == nil {
			network, addr = uri.Scheme, uri.Host
		}
		object.file, err = syslog.Dial(network, addr, syslog.LOG_INFO, tag)
	default:
		object.file, err = os.OpenFile(conf.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	if err != nil {
		panic(err)
	}
	return object
}

func (v *_log) Handler(l logx.Logger) {
	v.handler = l
	v.handler.SetOutput(v.file)
	v.handler.SetLevel(v.conf.Level)

	switch v.conf.Format {
	case "string", "syslog":
		strFmt := logx.NewFormatString()
		strFmt.SetDelimiter(' ')
		v.handler.SetFormatter(strFmt)
	case "json":
		v.handler.SetFormatter(logx.NewFormatJSON())
	}
}

func (v *_log) Close() error {
	return v.file.Close()
}
