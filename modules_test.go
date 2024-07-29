/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package grape_test

import (
	"testing"

	"go.osspkg.com/casecheck"
	"go.osspkg.com/grape"
)

func TestUnit_Modules(t *testing.T) {
	tmp1 := grape.Modules{8, 9, "W"}
	tmp2 := grape.Modules{18, 19, "aW", tmp1}
	main := grape.Modules{1, 2, "qqq"}.Add(tmp2).Add(99)

	casecheck.Equal(t, grape.Modules{1, 2, "qqq", 18, 19, "aW", 8, 9, "W", 99}, main)
}
