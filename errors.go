/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape

import "go.osspkg.com/errors"

var (
	errDepAlreadyRunning = errors.New("dependencies are already running")
	errDepNotRunning     = errors.New("dependencies are not running yet")
	errServiceUnknown    = errors.New("unknown service")
	errIsTypeError       = errors.New("ERROR")
	errBreakPointType    = errors.New("breakpoint can only be a function")
	errBreakPointAddress = errors.New("invalid breakpoint address")
)
