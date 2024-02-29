// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googleclientauthextension

import (
	"fmt"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/extension/googleclientauthextension"
)

// Config defines configuration for the Google client auth extension.
type Config struct {
	googleclientauthextension.Config `mapstructure:",squash"`
}

func (cfg *Config) Validate() error {
	if err := googleclientauthextension.ValidateConfig(cfg.Config); err != nil {
		return fmt.Errorf("googleclientauth extension settings are invalid :%w", err)
	}
	return nil
}
