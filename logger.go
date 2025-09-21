/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import (
	"io"
	"log/syslog"
	"net/url"
	"os"
	"sync"

	"go.osspkg.com/console"
	"go.osspkg.com/logx"

	"go.osspkg.com/grape/config"
)

var (
	instance *_log = nil
	mux            = sync.Mutex{}
)

type _log struct {
	file    io.WriteCloser
	handler logx.Logger
	conf    config.LogConfig
}

func initGlobalLogger(tag string, conf config.LogConfig, handler logx.Logger) *_log {
	var err error

	mux.Lock()
	defer mux.Unlock()

	if instance != nil {
		return instance
	}

	instance = &_log{
		conf:    conf,
		handler: handler,
	}

	switch conf.Format {
	case "syslog":
		network, addr := "", ""
		if uri, err0 := url.Parse(conf.FilePath); err0 == nil {
			network, addr = uri.Scheme, uri.Host
		}
		instance.file, err = syslog.Dial(network, addr, syslog.LOG_INFO, tag)
	default:
		instance.file, err = os.OpenFile(conf.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	}

	console.FatalIfErr(err, "open log file: %s %s", conf.Format, conf.FilePath)

	instance.handler.SetOutput(instance.file)
	instance.handler.SetLevel(instance.conf.Level)

	switch instance.conf.Format {
	case "string", "syslog":
		strFmt := logx.NewFormatString()
		strFmt.SetDelimiter(' ')
		instance.handler.SetFormatter(strFmt)
	case "json":
		instance.handler.SetFormatter(logx.NewFormatJSON())
	}

	return instance
}

func (v *_log) Close() error {
	mux.Lock()
	defer mux.Unlock()

	err := v.file.Close()
	instance = nil

	if err != nil {
		return err
	}

	return nil
}
