/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package errs

import "go.osspkg.com/errors"

var (
	ErrDepAlreadyRunning = errors.New("dependencies are already running")
	ErrDepNotRunning     = errors.New("dependencies are not running yet")
	ErrServiceUnknown    = errors.New("unknown service")
	ErrIsTypeError       = errors.New("ERROR")
	ErrBreakPointType    = errors.New("breakpoint can only be a function")
	ErrBreakPointAddress = errors.New("invalid breakpoint address")
)
