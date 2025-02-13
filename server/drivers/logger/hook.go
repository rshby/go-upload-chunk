package logger

import (
	"errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"strings"
)

type OtelTraceHook struct {
}

func (h *OtelTraceHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}

func (h *OtelTraceHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}

	span := trace.SpanFromContext(entry.Context)
	if !span.IsRecording() {
		return nil
	}

	// attach log to span event attributes
	attrs := []attribute.KeyValue{
		attribute.String("log.message", entry.Message),
		attribute.String("log.level", OtelSeverityText(entry.Level)),
	}
	span.AddEvent("log", trace.WithAttributes(attrs...))

	// set span status
	if entry.Level <= logrus.ErrorLevel {
		span.SetStatus(codes.Error, entry.Message)
		span.RecordError(errors.New(entry.Message))
	}

	return nil
}

func OtelSeverityText(lv logrus.Level) string {
	s := lv.String()
	if s == "warning" {
		s = "warn"
	}
	return strings.ToUpper(s)
}
