// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googleclientauthextension // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/googleclientauthextension"

import (
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/extension/googleclientauthextension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/googleclientauthextension/internal/metadata"
)

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(component.NewIDWithName(metadata.Type, "").String())
	require.NoError(t, err)
	require.NoError(t, component.UnmarshalConfig(sub, cfg))

	assert.Equal(t, cfg.(*Config), factory.CreateDefaultConfig().(*Config))

	sub, err = cm.Sub(component.NewIDWithName(metadata.Type, "customname").String())
	require.NoError(t, err)
	require.NoError(t, component.UnmarshalConfig(sub, cfg))
	config := &Config{&googleclientauthextension.Config{}}
	config.Project = "my-project"
	config.Scopes = []string{"https://www.something.com/hello", "https://www.something.com/world"}
	config.QuotaProject = "other-project"

	assert.Equal(t, config, cfg)
}

func TestValidate(t *testing.T) {
	assert.NoError(t, NewFactory().CreateDefaultConfig().(*Config).Validate())
}
