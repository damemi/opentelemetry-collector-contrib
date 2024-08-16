// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package goautoreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/goautoreceiver"

// This file implements factory for Go Auto receiver.

import (
	"context"
	"runtime"
	"time"

	"golang.org/x/sys/unix"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/goautoreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"go.opentelemetry.io/auto"
	otelauto "go.opentelemetry.io/auto/pkg/opentelemetry"
	"go.opentelemetry.io/auto/pkg/probe"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

type Controller struct {
	nextConsumer consumer.Traces
	version      string
	bootTime     int64
}

func NewOpenTelemetryController(nextConsumer consumer.Traces, version string) (otelauto.OpenTelemetryController, error) {
	bt, err := EstimateBootTimeOffset()
	if err != nil {
		return nil, err
	}

	return &Controller{
		nextConsumer: nextConsumer,
		version:      version,
		bootTime:     bt,
	}, nil
}

func (c *Controller) Trace(event *probe.Event) {
	ctx := context.Background()
	traces := ptrace.NewTraces()
	slice := ptrace.NewSpanSlice()
	for _, se := range event.SpanEvents {
		span := slice.AppendEmpty()

		span.SetName(se.SpanName)
		span.SetTraceID(pcommon.TraceID(se.SpanContext.TraceID()))
		span.SetSpanID(pcommon.SpanID(se.SpanContext.SpanID()))
		if se.ParentSpanContext != nil {
			span.SetParentSpanID(pcommon.SpanID(se.ParentSpanContext.SpanID()))
		}

		span.SetStartTimestamp(pcommon.Timestamp(c.convertTime(se.StartTime).Unix()))
		span.SetEndTimestamp(pcommon.Timestamp(c.convertTime(se.EndTime).Unix()))
		span.Status().SetCode(ptrace.StatusCode(se.Status.Code))
		span.Status().SetMessage(se.Status.Description)
		for _, attr := range se.Attributes {
			span.Attributes().PutStr(string(attr.Key), attr.Value.AsString())
		}
	}

	rs := traces.ResourceSpans().AppendEmpty()
	rs.SetSchemaUrl(semconv.SchemaURL)
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("go.opentelemetry.io/auto/" + event.Package)
	ss.Scope().SetVersion(c.version)
	slice.CopyTo(ss.Spans())

	c.nextConsumer.ConsumeTraces(ctx, traces)
}

func (c *Controller) convertTime(t int64) time.Time {
	return time.Unix(0, c.bootTime+t)
}

func createTracesReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	ctrl, err := NewOpenTelemetryController(nextConsumer, auto.Version())
	if err != nil {
		return nil, err
	}
	inst, err := auto.NewInstrumentation(ctx,
		auto.WithOpenTelemetryController(ctrl),
		auto.WithTarget(cfg.(*Config).target),
	)

	return &goAutoReceiver{
		inst: inst,
	}, nil
}

type goAutoReceiver struct {
	inst *auto.Instrumentation
}

func (g *goAutoReceiver) Start(ctx context.Context, _ component.Host) error {
	return g.inst.Run(ctx)
}

func (g *goAutoReceiver) Shutdown(ctx context.Context) error {
	return g.inst.Close()
}

func EstimateBootTimeOffset() (bootTimeOffset int64, err error) {
	// The datapath is currently using ktime_get_boot_ns for the pcap timestamp,
	// which corresponds to CLOCK_BOOTTIME. To be able to convert the the
	// CLOCK_BOOTTIME to CLOCK_REALTIME (i.e. a unix timestamp).

	// There can be an arbitrary amount of time between the execution of
	// time.Now() and unix.ClockGettime() below, especially under scheduler
	// pressure during program startup. To reduce the error introduced by these
	// delays, we pin the current Go routine to its OS thread and measure the
	// clocks multiple times, taking only the smallest observed difference
	// between the two values (which implies the smallest possible delay
	// between the two snapshots).
	var minDiff int64 = 1<<63 - 1
	estimationRounds := 25
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for round := 0; round < estimationRounds; round++ {
		var bootTimespec unix.Timespec

		// Ideally we would use __vdso_clock_gettime for both clocks here,
		// to have as little overhead as possible.
		// time.Now() will actually use VDSO on Go 1.9+, but calling
		// unix.ClockGettime to obtain CLOCK_BOOTTIME is a regular system call
		// for now.
		unixTime := time.Now()
		err = unix.ClockGettime(unix.CLOCK_BOOTTIME, &bootTimespec)
		if err != nil {
			return 0, err
		}

		offset := unixTime.UnixNano() - bootTimespec.Nano()
		diff := offset
		if diff < 0 {
			diff = -diff
		}

		if diff < minDiff {
			minDiff = diff
			bootTimeOffset = offset
		}
	}

	return bootTimeOffset, nil
}
