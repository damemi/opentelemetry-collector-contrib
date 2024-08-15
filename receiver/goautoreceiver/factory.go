// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package goautoreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/goautoreceiver"

// This file implements factory for Go Auto receiver.

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/goautoreceiver/internal/metadata"

	"go.opentelemetry.io/auto"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return nil
}

func createTracesReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	_, err := auto.NewInstrumentation(ctx)

	return nil, err
}
