package logging

import (
	"context"
	"encoding/json"
)

type stdoutLogWriter struct{}

func NewStdoutLogWriter() (*stdoutLogWriter, error) {
	return &stdoutLogWriter{}, nil
}

func (l *stdoutLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	return nil
}
