// Go support for leveled logs, analogous to https://code.google.com/p/google-glog/
//
// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

var testLogger = New()
var testDebugLogger = NewFromOptions(&Options{
	IncludeDebug: true,
})

// flushBuffer wraps a bytes.Buffer to satisfy SyncWriter.
type flushBuffer struct {
	bytes.Buffer
}

func (f *flushBuffer) Sync() error {
	return nil
}

// contents returns the specified log value as a string.
func contents() string {
	return testLogger.w.(*flushBuffer).String()
}

// contains reports whether the string is contained in the log.
func contains(str string, t *testing.T) bool {
	return strings.Contains(testLogger.w.(*flushBuffer).String(), str)
}

// debugContents returns the specified log value as a string.
func debugContents() string {
	return testDebugLogger.w.(*flushBuffer).String()
}

// debugContains reports whether the string is contained in the log.
func debugContains(str string, t *testing.T) bool {
	return strings.Contains(testDebugLogger.w.(*flushBuffer).String(), str)
}

func newTestLogger() {
	testLogger = New()
	testLogger.w = &flushBuffer{}
}

// Test that Debug does not emit by default.
func TestDebug(t *testing.T) {
	newTestLogger()
	testLogger.Debug("test")
	if contents() != "" {
		t.Errorf("Debug should not be emitted by default: %q", contents())
	}
}

// Test that Debug works if turned on.
func TestDebugOn(t *testing.T) {
	testDebugLogger = NewFromOptions(&Options{
		SyncWriter:   &flushBuffer{},
		IncludeDebug: true,
	})
	testDebugLogger.Debug("test")
	if !debugContains("D", t) {
		t.Errorf("Info has wrong character: %q", debugContents())
	}
	if !debugContains("test", t) {
		t.Error("Info failed")
	}
}

// Test that Info works as advertised.
func TestInfo(t *testing.T) {
	newTestLogger()
	testLogger.Info("test")
	if !contains("I", t) {
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test", t) {
		t.Error("Info failed")
	}
}

func TestInfof(t *testing.T) {
	newTestLogger()
	testLogger.Infof("test-%d", 100)
	if !contains("I", t) {
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test-100", t) {
		t.Error("Info failed")
	}
}

func TestRaw(t *testing.T) {
	newTestLogger()
	testLogger.Raw("test")
	if contents() != "test\n" {
		t.Error("Raw failed")
	}
}

func TestMultiLineInfo(t *testing.T) {
	newTestLogger()

	testLogger.Info("foo\nbar")
	if !contains("I", t) {
		t.Errorf("Info has wrong character: %q", contents())
	}

	lines := strings.Split(contents(), "\n")
	if len(lines) != 3 {
		t.Errorf("Wrong number of lines, got: %d want: 3", len(lines))
	}
	if len(lines[2]) != 0 {
		t.Error("Expected last line to be empty.")
	}

	if !strings.Contains(lines[0], "] foo") {
		t.Error("Failed to format first line 'foo'.")
	}
	if !strings.Contains(lines[1], "] bar") {
		t.Error("Failed to format second line 'bar'.")
	}
	if blen := testLogger.bufferCacheLen(); blen != 2 {
		t.Errorf("Wrong buffer length, got %d want 2", blen)
	}
}

func TestFatal(t *testing.T) {
	newTestLogger()

	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
	}
	testLogger.Fatal("foo")
	if !exitCalled {
		t.Error("Failed to call os.Exit on Fatal error.")
	}
	lines := strings.Split(contents(), "\n")
	if len(lines) < 10 {
		t.Errorf("Wrong number of lines, got: %d want: > 10", len(lines))
	}
}

// Test that the header has the correct format.
func TestHeader(t *testing.T) {
	testLogger.w = &flushBuffer{}
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	testLogger.Info("test")
	var line int
	format := "I0102 15:04:05.067890    1234 logger_test.go:%d] test\n"
	n, err := fmt.Sscanf(contents(), format, &line)
	if n != 1 || err != nil {
		t.Errorf("log format error: %d elements, error %s:\n%s", n, err, contents())
	}
	// Scanf treats multiple spaces as equivalent to a single space,
	// so check for correct space-padding also.
	want := fmt.Sprintf(format, line)
	if contents() != want {
		t.Errorf("log format error: got:\n\t%q\nwant:\t%q", contents(), want)
	}
}

func logFromADepth() {
	testLogger.Info("test")
}

// Test that the header respects DepthDelta.
func TestDepthDelta(t *testing.T) {
	testLogger.w = &flushBuffer{}
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	testLogger.depthDelta = 1 // Should report a line in testing.go which calls this func.
	logFromADepth()
	var line int
	format := "I0102 15:04:05.067890    1234 logger_test.go:%d] test\n"
	n, err := fmt.Sscanf(contents(), format, &line)
	if n != 1 || err != nil {
		t.Errorf("log format error: %d elements, error %s:\n%s", n, err, contents())
	}
	// Scanf treats multiple spaces as equivalent to a single space,
	// so check for correct space-padding also.
	want := fmt.Sprintf(format, line)
	if contents() != want {
		t.Errorf("log format error: got:\n\t%q\nwant:\t%q", contents(), want)
	}
	testLogger.depthDelta = 0
}

// Test that an Error log goes to Warning and Info.
// Even in the Info log, the source character will be E, so the data should
// all be identical.
func TestError(t *testing.T) {
	testLogger.w = &flushBuffer{}
	testLogger.Error("test")
	if !contains("E", t) {
		t.Errorf("Error has wrong character: %q", contents())
	}
	if !contains("test", t) {
		t.Error("Error failed")
	}
	str := contents()
	if !contains(str, t) {
		t.Error("Warning failed")
	}
	if !contains(str, t) {
		t.Error("Info failed")
	}
}

// Test that a Warning log goes to Info.
// Even in the Info log, the source character will be W, so the data should
// all be identical.
func TestWarning(t *testing.T) {
	testLogger.w = &flushBuffer{}
	testLogger.Warning("test")
	if !contains("W", t) {
		t.Errorf("Warning has wrong character: %q", contents())
	}
	if !contains("test", t) {
		t.Error("Warning failed")
	}
	str := contents()
	if !contains(str, t) {
		t.Error("Info failed")
	}
}

func BenchmarkHeader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf, _, _ := testLogger.header(infoLog, 0)
		testLogger.putBuffer(buf)
	}
}
