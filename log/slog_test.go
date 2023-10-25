//go:build go1.21

// +bulid go1.21
package log_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/sigmavirus24/circuitry/log"
)

type testHandler struct {
	records []slog.Record
	attrs   []slog.Attr
}

func (h *testHandler) clear() {
	h.records = make([]slog.Record, 0)
	h.attrs = make([]slog.Attr, 0)
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)
	return h
}

func (h *testHandler) WithGroup(_ string) slog.Handler { return h }

func newHandler() *testHandler { return &testHandler{make([]slog.Record, 0), make([]slog.Attr, 0)} }

func TestSLogWithField(t *testing.T) {
	handler := newHandler()
	slogger := slog.New(handler)
	logger := log.NewSLog(slogger)

	_ = logger.WithField("field", "value")
	if numAttrs := len(handler.attrs); numAttrs != 1 {
		t.Fatalf("expected there to be exactly 1 attr, but there were %d", numAttrs)
	}
	attr := handler.attrs[0]
	if attr.Key != "field" || !attr.Value.Equal(slog.StringValue("value")) {
		t.Fatalf("expected attr.Key = 'field', attr.Value = 'value'; got %+v", attr)
	}
}

func TestSLogWithFields(t *testing.T) {
	handler := newHandler()
	slogger := slog.New(handler)
	logger := log.NewSLog(slogger)

	_ = logger.WithFields(log.Fields{
		"afield":   "avalue",
		"bfield":   "bvalue",
		"cfield":   "cvalue",
		"intfield": 1234,
		"duration": 5 * time.Second,
		"enabled":  false,
	})
	expected := map[string]slog.Value{
		"afield":   slog.StringValue("avalue"),
		"bfield":   slog.StringValue("bvalue"),
		"cfield":   slog.StringValue("cvalue"),
		"intfield": slog.IntValue(1234),
		"duration": slog.DurationValue(5 * time.Second),
		"enabled":  slog.BoolValue(false),
	}
	if numAttrs := len(handler.attrs); numAttrs != 6 {
		t.Fatalf("expected there to be exactly 6 attrs, but there were %d", numAttrs)
	}
	for _, attr := range handler.attrs {
		value, ok := expected[attr.Key]
		if !ok {
			t.Fatalf("got attr with unexpected key = %s", attr.Key)
		}
		if !value.Equal(attr.Value) {
			t.Fatalf("expected attr with key = %s to have value = %v; got %v", attr.Key, value, attr.Value)
		}
	}
}

func TestSLogWithError(t *testing.T) {
	handler := newHandler()
	slogger := slog.New(handler)
	logger := log.NewSLog(slogger)

	expectedErr := errors.New("test slog.WithError")
	logger.WithError(expectedErr)

	if numAttrs := len(handler.attrs); numAttrs != 1 {
		t.Fatalf("expected there to be 1 attr, but there were %d", numAttrs)
	}
	attr := handler.attrs[0]
	if attr.Key != "err" {
		t.Fatalf("expected WithError to create attr.Key = \"err\"; got \"%s\"", attr.Key)
	}
	if !attr.Value.Equal(slog.AnyValue(expectedErr)) {
		t.Fatalf("expected attr.Value to be the error, got %+v", attr.Value)
	}

}

func TestSLogLevels(t *testing.T) {
	handler := newHandler()
	slogger := slog.New(handler)
	logger := log.NewSLog(slogger)

	logger.WithField("field", "value").Debug("message")
	if numAttrs := len(handler.attrs); numAttrs != 1 {
		t.Fatalf("expected only one attribute, found %d", numAttrs)
	}
	if numRecords := len(handler.records); numRecords != 1 {
		t.Fatalf("expected only one record, found %d", numRecords)
	}
	r := handler.records[0]
	if r.Level != slog.LevelDebug || r.Message != "message" {
		t.Fatalf("expected debug level record, got %+v", r)
	}

	handler.clear()
	logger.Info("message")
	if numRecords := len(handler.records); numRecords != 1 {
		t.Fatalf("expected only one record, found %d", numRecords)
	}
	r = handler.records[0]
	if r.Level != slog.LevelInfo || r.Message != "message" {
		t.Fatalf("expected info level record, got %+v", r)
	}

	handler.clear()
	logger.Warn("message")
	if numRecords := len(handler.records); numRecords != 1 {
		t.Fatalf("expected only one record, found %d", numRecords)
	}
	r = handler.records[0]
	if r.Level != slog.LevelWarn || r.Message != "message" {
		t.Fatalf("expected warn level record, got %+v", r)
	}

	handler.clear()
	logger.Error("message")
	if numRecords := len(handler.records); numRecords != 1 {
		t.Fatalf("expected only one record, found %d", numRecords)
	}
	r = handler.records[0]
	if r.Level != slog.LevelError || r.Message != "message" {
		t.Fatalf("expected error level record, got %+v", r)
	}
}
