/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import (
	"context"

	"go.osspkg.com/errors"
	"go.osspkg.com/syncing"
	"go.osspkg.com/xc"
)

type (
	ServiceInterface interface {
		Up() error
		Down() error
	}
	ServiceXContextInterface interface {
		Up(ctx xc.Context) error
		Down() error
	}
	ServiceContextInterface interface {
		Up(ctx context.Context) error
		Down() error
	}
)

func isService(v interface{}) bool {
	if _, ok := v.(ServiceContextInterface); ok {
		return true
	}
	if _, ok := v.(ServiceXContextInterface); ok {
		return true
	}
	if _, ok := v.(ServiceInterface); ok {
		return true
	}
	return false
}

func serviceCallUp(v interface{}, c xc.Context) error {
	if vv, ok := v.(ServiceContextInterface); ok {
		return vv.Up(c.Context())
	}
	if vv, ok := v.(ServiceXContextInterface); ok {
		return vv.Up(c)
	}
	if vv, ok := v.(ServiceInterface); ok {
		return vv.Up()
	}
	return errors.Wrapf(errServiceUnknown, "service [%T]", v)
}

func serviceCallDown(v interface{}) error {
	if vv, ok := v.(ServiceContextInterface); ok {
		return vv.Down()
	}
	if vv, ok := v.(ServiceXContextInterface); ok {
		return vv.Down()
	}
	if vv, ok := v.(ServiceInterface); ok {
		return vv.Down()
	}
	return errors.Wrapf(errServiceUnknown, "service [%T]", v)
}

/**********************************************************************************************************************/

type (
	treeItem struct {
		Previous *treeItem
		Current  interface{}
		Next     *treeItem
	}
	serviceTree struct {
		tree   *treeItem
		status syncing.Switch
		ctx    xc.Context
	}
)

func newServiceTree(ctx xc.Context) *serviceTree {
	return &serviceTree{
		tree:   nil,
		ctx:    ctx,
		status: syncing.NewSwitch(),
	}
}

func (s *serviceTree) IsOn() bool {
	return s.status.IsOn()
}

func (s *serviceTree) IsOff() bool {
	return s.status.IsOff()
}

func (s *serviceTree) MakeAsUp() error {
	if !s.status.On() {
		return errDepAlreadyRunning
	}
	return nil
}

func (s *serviceTree) IterateOver() {
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
func (s *serviceTree) AddAndUp(v interface{}) error {
	if s.IsOff() {
		return errDepNotRunning
	}

	if !isService(v) {
		return errors.Wrapf(errServiceUnknown, "service [%T]", v)
	}

	if s.tree == nil {
		s.tree = &treeItem{
			Previous: nil,
			Current:  v,
			Next:     nil,
		}
	} else {
		n := &treeItem{
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
func (s *serviceTree) Down() error {
	var err0 error
	if !s.status.Off() {
		return errDepNotRunning
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
