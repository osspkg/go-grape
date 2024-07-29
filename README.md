# Grape
Application module aggregator

## Base config file

***config.yaml***

```yaml
env: dev
level: 3 
log: /var/log/simple.log
pig: /var/run/simple.pid
```

level:

* 0 - error only
* 1 - + warning
* 2 - + info
* 3 - + debug

## Example

```go
package main

import (
	"fmt"

	"go.osspkg.com/grape"
	"go.osspkg.com/logx"
	"go.osspkg.com/xc"
)

type (
	//Simple model
	Simple struct{}
	//Config model
	Config struct {
		Env string `yaml:"env"`
	}
)

//NewSimple init Simple
func NewSimple(_ Config) *Simple {
	fmt.Println("--> call NewSimple")
	return &Simple{}
}

//Up  method for start Simple in DI container
func (s *Simple) Up(_ xc.Context) error {
	fmt.Println("--> call *Simple.Up")
	return nil
}

//Down  method for stop Simple in DI container
func (s *Simple) Down(_ xc.Context) error {
	fmt.Println("--> call *Simple.Down")
	return nil
}

func main() {
	grape.New().
		Logger(logx.Default()).
		ConfigFile(
			"./config.yaml",
			Config{},
		).
		Modules(
			NewSimple,
		).
		Run()
}
```

## HowTo

***Run the app***

```go
grape.New()
    .ConfigFile(<path to config file: string>, <config objects separate by comma: ...interface{}>)
    .Modules(<config objects separate by comma: ...interface{}>)
    .Run()
```

***Supported types for initialization***

* Function that returns an object or interface

*All incoming dependencies will be injected automatically*

```go
type Simple1 struct{}
func NewSimple1(_ *logx.Logger) *Simple1 { return &Simple1{} }
```

*Returns the interface*

```go
type Simple2 struct{}
type Simple2Interface interface{
    Get() string
}
func NewSimple2() Simple2Interface { return &Simple2{} }
func (s2 *Simple2) Get() string { 
    return "Hello world"
}
```

*If the object has the `Up(xc.Context) error` and `Down() error` methods, they will be called `Up(xc.Context) error`
when the app starts, and `Down() error` when it finishes. This allows you to automatically start and stop routine
processes inside the module*

```go
var _ service.IServiceCtx = (*Simple3)(nil)
type Simple3 struct{}
func NewSimple3(_ *Simple4) *Simple3 { return &Simple3{} }
func (s3 *Simple3) Up(_ xc.Context) error { return nil }
func (s3 *Simple3) Down(_ xc.Context) error { return nil }
```

* Named type

```go
type HelloWorld string
```

* Object structure

```go
type Simple4 struct{
    S1 *Simple1
    S2 Simple2Interface
    HW HelloWorld
}
```

* Object reference or type

```go
s1 := &Simple1{}
hw := HelloWorld("Hello!!")
```
