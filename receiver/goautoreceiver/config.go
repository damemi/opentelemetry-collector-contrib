// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package goautoreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/goautoreceiver"

type Config struct {
	target string `mapstructure:"target"`
}

func (cfg *Config) Validate() error {
	return nil
}
