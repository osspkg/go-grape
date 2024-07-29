/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import (
	"go.osspkg.com/config"
	"go.osspkg.com/console"
	"go.osspkg.com/events"
	"go.osspkg.com/logx"
	"go.osspkg.com/xc"
)

type (
	Grape interface {
		Logger(log logx.Logger) Grape
		Modules(modules ...interface{}) Grape
		ConfigResolvers(res ...config.Resolver) Grape
		ConfigFile(filename string) Grape
		ConfigModels(configs ...interface{}) Grape
		PidFile(filename string) Grape
		Run()
		Invoke(call interface{})
		Call(call interface{})
		ExitFunc(call func(code int)) Grape
	}

	_grape struct {
		configFilePath string
		pidFilePath    string
		resolvers      []config.Resolver
		configs        Modules
		modules        Modules
		packages       Container
		logHandler     *_log
		log            logx.Logger
		appContext     xc.Context
		exitFunc       func(code int)
	}
)

// New create application
func New() Grape {
	ctx := xc.New()
	return &_grape{
		resolvers:  make([]config.Resolver, 0, 2),
		modules:    Modules{},
		configs:    Modules{},
		packages:   NewContainer(ctx),
		appContext: ctx,
		exitFunc:   func(_ int) {},
	}
}

// Logger setup logger
func (a *_grape) Logger(l logx.Logger) Grape {
	a.log = l
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
	a.configFilePath = filename
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
	for _, r := range crs {
		a.resolvers = append(a.resolvers, r)
	}
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
	if len(a.configFilePath) == 0 {
		a.logHandler = newLog(LogConfig{
			Level:    4,
			FilePath: "/dev/stdout",
			Format:   "string",
		})
		if a.log == nil {
			a.log = logx.Default()
		}
		a.logHandler.Handler(a.log)
	}
	if len(a.configFilePath) > 0 {
		// read config file
		resolver := config.New(a.resolvers...)
		if err = resolver.OpenFile(a.configFilePath); err != nil {
			console.FatalIfErr(err, "open config file: %s", a.configFilePath)
		}
		if err = resolver.Build(); err != nil {
			console.FatalIfErr(err, "prepare config file: %s", a.configFilePath)
		}
		appConfig := &Config{}
		if err = resolver.Decode(appConfig); err != nil {
			console.FatalIfErr(err, "decode config file: %s", a.configFilePath)
		}
		if interactive {
			appConfig.Log.Level = 4
			appConfig.Log.FilePath = "/dev/stdout"
		}

		// init logger
		a.logHandler = newLog(appConfig.Log)
		if a.log == nil {
			a.log = logx.Default()
		}
		a.logHandler.Handler(a.log)
		a.modules = a.modules.Add(
			ENV(appConfig.Env),
		)

		// decode all configs
		var configs []interface{}
		configs, err = typingReflectPtr(a.configs, func(c interface{}) error {
			return resolver.Decode(c)
		})
		if err != nil {
			a.log.WithFields(logx.Fields{
				"err": err.Error(),
			}).Fatalf("Decode config file")
		}
		a.modules = a.modules.Add(configs...)

		if !interactive && len(a.pidFilePath) > 0 {
			if err = pid2file(a.pidFilePath); err != nil {
				a.log.WithFields(logx.Fields{
					"err":  err.Error(),
					"file": a.pidFilePath,
				}).Fatalf("Create pid file")
			}
		}
	}
	a.modules = a.modules.Add(
		func() logx.Logger { return a.log },
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
			a.log.Infof(s.Message)
		}
		if err := s.Call(); err != nil {
			a.log.WithFields(logx.Fields{
				"err": err.Error(),
			}).Errorf(s.Message)
			erc++
			break
		}
	}

	wait(erc > 0)

	for _, s := range down {
		if len(s.Message) > 0 {
			a.log.Infof(s.Message)
		}
		if err := s.Call(); err != nil {
			a.log.WithFields(logx.Fields{
				"err": err.Error(),
			}).Errorf(s.Message)
			erc++
		}
	}

	return erc > 0
}
