/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package internal

import (
	"os"
	"strconv"
	"syscall"
)

func SavePidToFile(filename string) error {
	pid := strconv.Itoa(syscall.Getpid())
	return os.WriteFile(filename, []byte(pid), 0755)
}
