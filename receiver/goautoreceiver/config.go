// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package goautoreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/goautoreceiver"

type Config struct{}

func (cfg *Config) Validate() error {
	return nil
}
