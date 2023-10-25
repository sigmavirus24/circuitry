package log_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sigmavirus24/circuitry/log"
	"github.com/sirupsen/logrus"
)

func newTestLogger() (*bytes.Buffer, *logrus.Logger) {
	var b bytes.Buffer
	return &b, &logrus.Logger{Out: &b, Formatter: &logrus.JSONFormatter{}, Level: logrus.DebugLevel}
}

func TestLogrus(t *testing.T) {
	testCases := map[string]struct {
		testFn   func(l *log.Logrus)
		levelStr string
	}{
		"debug": {
			testFn: func(l *log.Logrus) {
				l.WithError(errors.New("test")).
					WithField("field", "value").
					WithFields(log.Fields{"a": 1, "b": 2.2, "c": "d"}).
					Debug("message")
			},
			levelStr: "debug",
		},
		"info": {
			testFn: func(l *log.Logrus) {
				l.WithError(errors.New("test")).
					WithField("field", "value").
					WithFields(log.Fields{"a": 1, "b": 2.2, "c": "d"}).
					Info("message")
			},
			levelStr: "info",
		},
		"warn": {
			testFn: func(l *log.Logrus) {
				l.WithError(errors.New("test")).
					WithField("field", "value").
					WithFields(log.Fields{"a": 1, "b": 2.2, "c": "d"}).
					Warn("message")
			},
			levelStr: "warning",
		},
		"error": {
			testFn: func(l *log.Logrus) {
				l.WithError(errors.New("test")).
					WithField("field", "value").
					WithFields(log.Fields{"a": 1, "b": 2.2, "c": "d"}).
					Error("message")
			},
			levelStr: "error",
		},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			buf, logrusLogger := newTestLogger()
			logger := log.NewLogrus(logrusLogger)

			tc.testFn(logger)

			logline := buf.Bytes()
			parsedLog := struct {
				A     int     `json:"a"`
				B     float64 `json:"b"`
				C     string  `json:"c"`
				Err   string  `json:"error"`
				Field string  `json:"field"`
				Level string  `json:"level"`
				Msg   string  `json:"msg"`
				Time  string  `json:"time"`
			}{}
			err := json.Unmarshal(logline, &parsedLog)
			if err != nil {
				t.Fatalf("cannot parse log line: %q due to err %v", string(logline), err)
			}
			if parsedLog.Level != tc.levelStr {
				t.Fatalf("invalid log level for message, expected %q, got %q", tc.levelStr, parsedLog.Level)
			}

		})
	}
}
