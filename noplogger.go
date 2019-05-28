package logger

import "github.com/jcgregorio/slog"

// NopLogger implements slog.Logger and does nothing.
//
type NopLogger struct{}

// NewNopLogger returns an initialized *NopLogger.
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

// Fatal logs a fatal log and then exits the program.
// Arguments are handled in the manner of fmt.Print.
func (n *NopLogger) Fatal(args ...interface{}) {}

// Fatalf logs a fatal log and then exits the program.
// Arguments are handled in the manner of fmt.Printf.
func (*NopLogger) Fatalf(format string, args ...interface{}) {}

// Error logs error logs.
// Arguments are handled in the manner of fmt.Print.
func (*NopLogger) Error(args ...interface{}) {}

// Errorf logs error logs.
// Arguments are handled in the manner of fmt.Printf.
func (*NopLogger) Errorf(format string, args ...interface{}) {}

// Warning logs warning logs.
// Arguments are handled in the manner of fmt.Print.
func (*NopLogger) Warning(args ...interface{}) {}

// Warning logs warning logs.
// Arguments are handled in the manner of fmt.Printf.
func (*NopLogger) Warningf(format string, args ...interface{}) {}

// Info logs informational logs.
// Arguments are handled in the manner of fmt.Print.
func (*NopLogger) Info(args ...interface{}) {}

// Infof logs informational logs.
// Arguments are handled in the manner of fmt.Printf.
func (*NopLogger) Infof(format string, args ...interface{}) {}

// Debug logs debugging logs.
// Arguments are handled in the manner of fmt.Print.
func (*NopLogger) Debug(args ...interface{}) {}

// Debugf logs debugging logs.
// Arguments are handled in the manner of fmt.Printf.
func (*NopLogger) Debugf(format string, args ...interface{}) {}

// Raw sends the string s to the logs without any additional formatting.
func (*NopLogger) Raw(s string) {}

// Assert that we implement the slog.Logger interface:
var _ slog.Logger = (*NopLogger)(nil)
