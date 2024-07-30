/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package container

import (
	"fmt"
	"reflect"

	"go.osspkg.com/algorithms/graph/kahn"
	"go.osspkg.com/errors"
	errors2 "go.osspkg.com/grape/errors"
	reflect2 "go.osspkg.com/grape/reflect"
	"go.osspkg.com/grape/services"
	"go.osspkg.com/syncing"
	"go.osspkg.com/xc"
)

type (
	_container struct {
		kahn   *kahn.Graph
		srv    services.TServices
		store  *objectStorage
		status syncing.Switch
	}

	TContainer interface {
		Start() error
		Register(items ...interface{}) error
		Invoke(item interface{}) error
		BreakPoint(item interface{}) error
		Stop() error
	}
)

func New(ctx xc.Context) TContainer {
	return &_container{
		kahn:   kahn.New(),
		srv:    services.New(ctx),
		store:  newObjectStorage(),
		status: syncing.NewSwitch(),
	}
}

// Stop - stop all services in dependencies
func (v *_container) Stop() error {
	if !v.status.Off() {
		return nil
	}
	return v.srv.Down()
}

// Start - initialize dependencies and start
func (v *_container) Start() error {
	if !v.status.On() {
		return errors2.ErrDepAlreadyRunning
	}
	if err := v.srv.MakeAsUp(); err != nil {
		return err
	}
	if err := v.prepare(); err != nil {
		return err
	}
	if err := v.kahn.Build(); err != nil {
		return errors.Wrapf(err, "dependency graph calculation")
	}
	return v.run()
}

func (v *_container) Register(items ...interface{}) error {
	if v.srv.IsOn() {
		return errors2.ErrDepAlreadyRunning
	}

	for _, item := range items {
		ref := reflect.TypeOf(item)
		rt := asTypeExist
		switch ref.Kind() {
		case reflect.Func, reflect.Struct:
			rt = asTypeNew
		default:
		}
		if err := v.store.Add(ref, item, rt); err != nil {
			return err
		}
	}
	return nil
}

func (v *_container) BreakPoint(item interface{}) error {
	if v.srv.IsOn() {
		return errors2.ErrDepAlreadyRunning
	}
	ref := reflect.TypeOf(item)
	switch ref.Kind() {
	case reflect.Func:
	default:
		return errors2.ErrBreakPointType
	}
	address, ok := reflect2.GetAddress(ref, item)
	if !ok {
		return errors2.ErrBreakPointAddress
	}
	v.kahn.BreakPoint(address)
	return nil
}

const root = "ROOT"

func (v *_container) prepare() error {
	return v.store.Each(func(item *objectStorageItem) error {

		switch item.Kind {

		case reflect.Func:
			if item.ReflectType.NumIn() == 0 {
				v.kahn.Add(root, item.Address)
			}
			for i := 0; i < item.ReflectType.NumIn(); i++ {
				inRefType := item.ReflectType.In(i)
				inAddress, _ := reflect2.GetAddress(inRefType, nil)
				v.kahn.Add(inAddress, item.Address)
			}

			for i := 0; i < item.ReflectType.NumOut(); i++ {
				outRefType := item.ReflectType.Out(i)
				outAddress, _ := reflect2.GetAddress(outRefType, nil)
				v.kahn.Add(item.Address, outAddress)
			}

		case reflect.Struct:
			if item.ReflectType.NumField() == 0 {
				v.kahn.Add(root, item.Address)
			}
			for i := 0; i < item.ReflectType.NumField(); i++ {
				inRefType := item.ReflectType.Field(i).Type
				inAddress, _ := reflect2.GetAddress(inRefType, nil)
				v.kahn.Add(inAddress, item.Address)
			}

		default:
			v.kahn.Add(root, item.Address)
		}

		return nil
	})
}

func (v *_container) Invoke(obj interface{}) error {
	if v.srv.IsOff() {
		return errors2.ErrDepNotRunning
	}
	item, _, err := v.callArgs(obj)
	if err != nil {
		return err
	}
	if item.Service == itDownService {
		if err = v.srv.AddAndUp(item.Value); err != nil {
			return err
		}
	}
	return nil
}

func (v *_container) toStoreItem(obj interface{}) (*objectStorageItem, error) {
	item, ok := obj.(*objectStorageItem)
	if ok {
		return item, nil
	}
	ref := reflect.TypeOf(obj)
	if err := v.store.Add(ref, obj, asTypeNew); err != nil {
		return nil, err
	}
	return v.store.GetByReflect(ref, obj)
}

func (v *_container) callArgs(obj interface{}) (*objectStorageItem, []reflect.Value, error) {
	item, err := v.toStoreItem(obj)
	if err != nil {
		return nil, nil, err
	}

	switch item.Kind {

	case reflect.Func:
		args := make([]reflect.Value, 0, item.ReflectType.NumIn())
		for i := 0; i < item.ReflectType.NumIn(); i++ {
			inRefType := item.ReflectType.In(i)
			inAddress, ok := reflect2.GetAddress(inRefType, item.Value)
			if !ok {
				return nil, nil, fmt.Errorf("dependency [%s] is not supported", inAddress)
			}
			dep, err := v.store.GetByAddress(inAddress)
			if err != nil {
				return nil, nil, err
			}
			args = append(args, reflect.ValueOf(dep.Value))
		}
		args = reflect.ValueOf(item.Value).Call(args)
		for _, arg := range args {
			if err, ok := arg.Interface().(error); ok && err != nil {
				return nil, nil, err
			}
		}
		return item, args, nil

	case reflect.Struct:
		value := reflect.New(item.ReflectType)
		args := make([]reflect.Value, 0, 1)
		for i := 0; i < item.ReflectType.NumField(); i++ {
			inRefType := item.ReflectType.Field(i)
			inAddress, ok := reflect2.GetAddress(inRefType.Type, nil)
			if !ok {
				return nil, nil, fmt.Errorf("dependency [%s] is not supported", inAddress)
			}
			dep, err := v.store.GetByAddress(inAddress)
			if err != nil {
				return nil, nil, err
			}
			value.Elem().FieldByName(inRefType.Name).Set(reflect.ValueOf(dep.Value))
		}
		return item, append(args, value.Elem()), nil

	default:
	}

	return item, []reflect.Value{reflect.ValueOf(item.Value)}, nil
}

// nolint: gocyclo
func (v *_container) run() error {
	names := make(map[string]struct{})
	for _, name := range v.kahn.Result() {
		if name == root {
			continue
		}
		names[name] = struct{}{}
	}

	defer v.srv.IterateOver()

	for _, name := range v.kahn.Result() {
		if name == root || name == reflect2.ErrorName {
			continue
		}
		item, err := v.store.GetByAddress(name)
		if err != nil {
			return err
		}
		if item.RelationType == asTypeExist {
			if item.Service == itDownService {
				if err = v.srv.AddAndUp(item.Value); err != nil {
					return errors.Wrapf(err, "service initialization error [%s]", item.Address)
				}
				item.Service = itUpedService
			}
			delete(names, name)
			continue
		}
		_, args, err := v.callArgs(item)
		if err != nil {
			return errors.Wrapf(err, "initialize error [%s]", name)
		}
		delete(names, name)
		for _, arg := range args {
			if err = v.store.Add(arg.Type(), arg.Interface(), asTypeExist); err != nil {
				return errors.Wrapf(err, "initialize error")
			}
			if item, err = v.store.GetByReflect(arg.Type(), arg.Interface()); err != nil {
				if errors.Is(err, errors2.ErrIsTypeError) {
					continue
				}
				return errors.Wrapf(err, "initialize error")
			}
			delete(names, item.Address)
			if item.Service == itDownService {
				if err = v.srv.AddAndUp(item.Value); err != nil {
					return errors.Wrapf(err, "service initialization error [%s]", item.Address)
				}
				item.Service = itUpedService
			}
		}
	}

	return nil
}
