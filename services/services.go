/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package services

import (
	"context"

	"go.osspkg.com/errors"
	errors2 "go.osspkg.com/grape/errors"
	"go.osspkg.com/syncing"
	"go.osspkg.com/xc"
)

type (
	TService interface {
		Up() error
		Down() error
	}
	TServiceXContext interface {
		Up(ctx xc.Context) error
		Down() error
	}
	TServiceContext interface {
		Up(ctx context.Context) error
		Down() error
	}
)

func IsService(v interface{}) bool {
	if _, ok := v.(TServiceContext); ok {
		return true
	}
	if _, ok := v.(TServiceXContext); ok {
		return true
	}
	if _, ok := v.(TService); ok {
		return true
	}
	return false
}

func serviceCallUp(v interface{}, c xc.Context) error {
	if vv, ok := v.(TServiceContext); ok {
		return vv.Up(c.Context())
	}
	if vv, ok := v.(TServiceXContext); ok {
		return vv.Up(c)
	}
	if vv, ok := v.(TService); ok {
		return vv.Up()
	}
	return errors.Wrapf(errors2.ErrServiceUnknown, "service [%T]", v)
}

func serviceCallDown(v interface{}) error {
	if vv, ok := v.(TServiceContext); ok {
		return vv.Down()
	}
	if vv, ok := v.(TServiceXContext); ok {
		return vv.Down()
	}
	if vv, ok := v.(TService); ok {
		return vv.Down()
	}
	return errors.Wrapf(errors2.ErrServiceUnknown, "service [%T]", v)
}

/**********************************************************************************************************************/

type (
	item struct {
		Previous *item
		Current  interface{}
		Next     *item
	}
	_services struct {
		tree   *item
		status syncing.Switch
		ctx    xc.Context
	}
	TServices interface {
		IsOn() bool
		IsOff() bool
		MakeAsUp() error
		IterateOver()
		AddAndUp(v interface{}) error
		Down() error
	}
)

func New(ctx xc.Context) TServices {
	return &_services{
		tree:   nil,
		ctx:    ctx,
		status: syncing.NewSwitch(),
	}
}

func (s *_services) IsOn() bool {
	return s.status.IsOn()
}

func (s *_services) IsOff() bool {
	return s.status.IsOff()
}

func (s *_services) MakeAsUp() error {
	if !s.status.On() {
		return errors2.ErrDepAlreadyRunning
	}
	return nil
}

func (s *_services) IterateOver() {
	if s.tree == nil {
		return
	}
	for s.tree.Previous != nil {
		s.tree = s.tree.Previous
	}
	for {
		if s.tree.Next == nil {
			break
		}
		s.tree = s.tree.Next
	}
	return
}

// AddAndUp - add new service and call up
func (s *_services) AddAndUp(v interface{}) error {
	if s.IsOff() {
		return errors2.ErrDepNotRunning
	}

	if !IsService(v) {
		return errors.Wrapf(errors2.ErrServiceUnknown, "service [%T]", v)
	}

	if s.tree == nil {
		s.tree = &item{
			Previous: nil,
			Current:  v,
			Next:     nil,
		}
	} else {
		n := &item{
			Previous: s.tree,
			Current:  v,
			Next:     nil,
		}
		n.Previous.Next = n
		s.tree = n
	}

	return serviceCallUp(v, s.ctx)
}

// Down - stop all services
func (s *_services) Down() error {
	var err0 error
	if !s.status.Off() {
		return errors2.ErrDepNotRunning
	}
	if s.tree == nil {
		return nil
	}
	for {
		if err := serviceCallDown(s.tree.Current); err != nil {
			err0 = errors.Wrap(err0,
				errors.Wrapf(err, "down [%T] service error", s.tree.Current),
			)
		}
		if s.tree.Previous == nil {
			break
		}
		s.tree = s.tree.Previous
	}
	for s.tree.Next != nil {
		s.tree = s.tree.Next
	}
	return err0
}
