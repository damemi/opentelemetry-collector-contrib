package googlecloudloggingexporter

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/model/pdata"
	"go.uber.org/zap"
)

type exporter struct {
	Config             *Config
	logger             *zap.Logger
	collectorID        string
	cloudLoggingClient *logging.Client
	cloudLogger        *logging.Logger
}

func newCloudLoggingExporter(config *Config, params component.ExporterCreateSettings) (component.LogsExporter, error) {
	loggingExporter, err := newCloudLoggingLogExporter(config, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogsExporter(
		config,
		params,
		loggingExporter.ConsumeLogs,
		exporterhelper.WithQueue(config.enforcedQueueSettings()),
		exporterhelper.WithRetry(config.RetrySettings))
}

func newCloudLoggingLogExporter(config *Config, params component.ExporterCreateSettings) (component.LogsExporter, error) {
	// Validate the passed config.
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Generate a Collector ID.
	collectorIdentifier, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// Read project ID from Metadata if not specified by config.
	if config.ProjectID == "" {
		projectId, err := readProjectIdMetadata()
		if err != nil {
			return nil, fmt.Errorf("failed to read Google Cloud project ID: %v", err)
		}
		config.ProjectID = projectId
	}

	// Create Cloud Logging logger with project ID.
	client, err := logging.NewClient(context.Background(), config.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Logging client: %v", err)
	}
	logger := client.Logger(config.LogName)

	// Create the logging exporter.
	loggingExporter := &exporter{
		Config:             config,
		logger:             params.Logger,
		collectorID:        collectorIdentifier.String(),
		cloudLoggingClient: client,
		cloudLogger:        logger,
	}
	return loggingExporter, nil
}

func (e *exporter) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	logEntries, dropped := logsToEntries(e.logger, ld)
	if len(logEntries) == 0 {
		return nil
	}
	if dropped > 0 {
		e.logger.Debug("Dropped logs", zap.Any("logsDropped", dropped))
	}

	for _, logEntry := range logEntries {
		e.logger.Debug("Adding log entry", zap.Any("entry", logEntry))
		e.cloudLogger.Log(logEntry)
	}
	e.logger.Debug("Log entries successfully buffered")
	err := e.cloudLogger.Flush()
	if err != nil {
		e.logger.Error("error force flushing logs. Skipping to next logPusher.", zap.Error(err))
	}
	return nil
}

func (e *exporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e *exporter) Shutdown(ctx context.Context) error {
	// Flush the remaining logs before shutting down the exporter.
	if e.cloudLogger != nil {
		err := e.cloudLogger.Flush()
		if err != nil {
			return err
		}
	}
	e.cloudLoggingClient.Close()
	return nil
}

func (e *exporter) Start(ctx context.Context, host component.Host) error {
	return nil
}

func logsToEntries(logger *zap.Logger, ld pdata.Logs) ([]logging.Entry, int) {
	entries := []logging.Entry{}
	dropped := 0
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		resourceAttrs := attrsValue(rl.Resource().Attributes())
		ills := rl.InstrumentationLibraryLogs()
		for j := 0; j < ills.Len(); j++ {
			ils := ills.At(j)
			logs := ils.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				entry, err := logToEntry(resourceAttrs, log)
				if err != nil {
					logger.Debug("Failed to convert to Cloud Logging Entry", zap.Error(err))
					dropped++
				} else {
					entries = append(entries, entry)
				}
			}
		}
	}
	return entries, dropped
}

type entryPayload struct {
	Message string `json:"message"`
}

func logToEntry(attributes map[string]interface{}, log pdata.LogRecord) (logging.Entry, error) {
	payload := entryPayload{
		Message: log.Body().AsString(),
	}
	return logging.Entry{
		Payload:   payload,
		Timestamp: log.Timestamp().AsTime(),
		Severity:  logging.Severity(log.SeverityNumber()),
		Trace:     log.TraceID().HexString(),
		SpanID:    log.SpanID().HexString(),
	}, nil
}

func attrsValue(attrs pdata.AttributeMap) map[string]interface{} {
	if attrs.Len() == 0 {
		return nil
	}
	out := make(map[string]interface{}, attrs.Len())
	attrs.Range(func(k string, v pdata.AttributeValue) bool {
		out[k] = attrValue(v)
		return true
	})
	return out
}

func attrValue(value pdata.AttributeValue) interface{} {
	switch value.Type() {
	case pdata.AttributeValueTypeInt:
		return value.IntVal()
	case pdata.AttributeValueTypeBool:
		return value.BoolVal()
	case pdata.AttributeValueTypeDouble:
		return value.DoubleVal()
	case pdata.AttributeValueTypeString:
		return value.StringVal()
	case pdata.AttributeValueTypeMap:
		values := map[string]interface{}{}
		value.MapVal().Range(func(k string, v pdata.AttributeValue) bool {
			values[k] = attrValue(v)
			return true
		})
		return values
	case pdata.AttributeValueTypeArray:
		arrayVal := value.SliceVal()
		values := make([]interface{}, arrayVal.Len())
		for i := 0; i < arrayVal.Len(); i++ {
			values[i] = attrValue(arrayVal.At(i))
		}
		return values
	case pdata.AttributeValueTypeEmpty:
		return nil
	default:
		return nil
	}
}
