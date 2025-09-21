/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import (
	"go.osspkg.com/config"
	"go.osspkg.com/console"
	"go.osspkg.com/events"
	"go.osspkg.com/logx"
	"go.osspkg.com/xc"

	config2 "go.osspkg.com/grape/config"
	"go.osspkg.com/grape/container"
	"go.osspkg.com/grape/env"
	"go.osspkg.com/grape/internal"
	"go.osspkg.com/grape/reflect"
)

type AppName string

type Grape interface {
	Logger(log logx.Logger) Grape
	Modules(modules ...interface{}) Grape
	ConfigResolvers(res ...config.Resolver) Grape
	ConfigFile(filename string) Grape
	ConfigData(data, ext string) Grape
	ConfigModels(configs ...interface{}) Grape
	PidFile(filename string) Grape
	Run()
	Invoke(call interface{})
	Call(call interface{})
	ExitFunc(call func(code int)) Grape
}

type (
	_conf struct {
		FileName string
		Data     string
		DataExt  string
	}
	_grape struct {
		appName     string
		config      _conf
		pidFilePath string
		resolvers   []config.Resolver
		configs     Modules
		modules     Modules
		packages    container.TContainer
		logHandler  *_log
		logger      logx.Logger
		appContext  xc.Context
		exitFunc    func(code int)
	}
)

// New create application
func New(appName string) Grape {
	ctx := xc.New()
	return &_grape{
		appName:    appName,
		resolvers:  make([]config.Resolver, 0, 2),
		modules:    Modules{},
		configs:    Modules{},
		packages:   container.New(ctx),
		appContext: ctx,
		exitFunc:   func(_ int) {},
	}
}

// Logger setup logger
func (a *_grape) Logger(l logx.Logger) Grape {
	a.logger = l
	return a
}

// Modules append object to modules list
func (a *_grape) Modules(modules ...interface{}) Grape {
	for _, mod := range modules {
		switch v := mod.(type) {
		case Modules:
			a.modules = a.modules.Add(v...)
		default:
			a.modules = a.modules.Add(v)
		}
	}
	return a
}

// ConfigFile set config file path
func (a *_grape) ConfigFile(filename string) Grape {
	a.config = _conf{
		FileName: filename,
	}
	return a
}

// ConfigData set config data
func (a *_grape) ConfigData(data, ext string) Grape {
	a.config = _conf{
		Data:    data,
		DataExt: ext,
	}
	return a
}

// ConfigModels set configs models
func (a *_grape) ConfigModels(configs ...interface{}) Grape {
	for _, c := range configs {
		a.configs = a.configs.Add(c)
	}
	return a
}

// ConfigResolvers set configs resolvers
func (a *_grape) ConfigResolvers(crs ...config.Resolver) Grape {
	a.resolvers = append(a.resolvers, crs...)
	return a
}

func (a *_grape) PidFile(filename string) Grape {
	a.pidFilePath = filename
	return a
}

func (a *_grape) ExitFunc(v func(code int)) Grape {
	a.exitFunc = v
	return a
}

// Run application with all dependencies
func (a *_grape) Run() {
	a.prepareConfig(false)

	result := a.steps(
		[]step{
			{
				Message: "Registering dependencies",
				Call:    func() error { return a.packages.Register(a.modules...) },
			},
			{
				Message: "Running dependencies",
				Call:    func() error { return a.packages.Start() },
			},
		},
		func(er bool) {
			if er {
				a.appContext.Close()
				return
			}
			go events.OnStopSignal(a.appContext.Close)
			<-a.appContext.Done()
		},
		[]step{
			{
				Message: "Stop dependencies",
				Call:    func() error { return a.packages.Stop() },
			},
		},
	)
	console.FatalIfErr(a.logHandler.Close(), "close log file")
	if result {
		a.exitFunc(1)
	}
	a.exitFunc(0)
}

// Invoke run application with all dependencies and call function after starting
func (a *_grape) Invoke(call interface{}) {
	a.prepareConfig(true)

	result := a.steps(
		[]step{
			{
				Call: func() error { return a.packages.Register(a.modules...) },
			},
			{
				Call: func() error { return a.packages.Start() },
			},
			{
				Call: func() error { return a.packages.Invoke(call) },
			},
		},
		func(_ bool) {},
		[]step{
			{
				Call: func() error { return a.packages.Stop() },
			},
		},
	)
	console.FatalIfErr(a.logHandler.Close(), "close log file")
	if result {
		a.exitFunc(1)
	}
	a.exitFunc(0)
}

// Call function with dependency and without starting all app
func (a *_grape) Call(call interface{}) {
	a.prepareConfig(true)

	result := a.steps(
		[]step{
			{
				Call: func() error { return a.packages.Register(a.modules...) },
			},
			{
				Call: func() error { return a.packages.Register(call) },
			},
			{
				Call: func() error { return a.packages.BreakPoint(call) },
			},
			{
				Call: func() error { return a.packages.Start() },
			},
		},
		func(_ bool) {},
		[]step{
			{
				Call: func() error { return a.packages.Stop() },
			},
		},
	)
	console.FatalIfErr(a.logHandler.Close(), "close log file")
	if result {
		a.exitFunc(1)
	}
	a.exitFunc(0)
}

func (a *_grape) prepareConfig(interactive bool) {
	var err error
	appConfig := config2.Default()

	// read config file
	resolver := config.New(a.resolvers...)
	switch true {
	case len(a.config.Data) > 0:
		resolver.OpenBlob(a.config.Data, a.config.DataExt)
	case len(a.config.FileName) > 0:
		console.FatalIfErr(resolver.OpenFile(a.config.FileName), "Open config file: %s", a.config)
	}
	console.FatalIfErr(resolver.Build(), "Prepare config file: %s", a.config)
	if !interactive {
		console.FatalIfErr(resolver.Decode(appConfig), "Decode config file: %s", a.config)
	}

	// init logger
	if a.logger == nil {
		a.logger = logx.Default()
	}
	a.logHandler = initGlobalLogger(a.appName, appConfig.Log, a.logger)

	// set env
	a.modules = a.modules.Add(
		env.ENV(appConfig.Env),
	)

	// decode all configs
	var configs []interface{}
	configs, err = reflect.TypingPtr(a.configs, func(c interface{}) error {
		return resolver.Decode(c)
	})
	console.FatalIfErr(err, "Decode config file: %s", a.config)
	a.modules = a.modules.Add(configs...)

	if !interactive && len(a.pidFilePath) > 0 {
		console.FatalIfErr(internal.SavePidToFile(a.pidFilePath), "Create pid file: %s", a.pidFilePath)
	}
	a.modules = a.modules.Add(
		func() logx.Logger { return a.logger },
		func() xc.Context { return a.appContext },
	)
}

type step struct {
	Call    func() error
	Message string
}

func (a *_grape) steps(up []step, wait func(bool), down []step) bool {
	var erc int

	for _, s := range up {
		if len(s.Message) > 0 {
			a.logger.Info(s.Message)
		}
		if err := s.Call(); err != nil {
			a.logger.Error(s.Message, "err", err)
			erc++
			break
		}
	}

	wait(erc > 0)

	for _, s := range down {
		if len(s.Message) > 0 {
			a.logger.Info(s.Message)
		}
		if err := s.Call(); err != nil {
			a.logger.Error(s.Message, "err", err)
			erc++
		}
	}

	return erc > 0
}
