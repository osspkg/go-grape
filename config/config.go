/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package config

type (
	// Config config model
	Config struct {
		Env string    `yaml:"env"`
		Log LogConfig `yaml:"log"`
	}

	LogConfig struct {
		Level    uint32 `yaml:"level"`
		FilePath string `yaml:"file_path"`
		Format   string `yaml:"format"`
	}
)
