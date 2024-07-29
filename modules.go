/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

// Modules DI container
type Modules []interface{}

// Add object to container
func (m Modules) Add(v ...interface{}) Modules {
	for _, mod := range v {
		switch obj := mod.(type) {
		case Modules:
			m = m.Add(obj...)
		default:
			m = append(m, mod)
		}
	}
	return m
}
