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

var test_logger = New()
var test_debug_logger = NewFromOptions(&Options{
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
	return test_logger.w.(*flushBuffer).String()
}

// contains reports whether the string is contained in the log.
func contains(str string, t *testing.T) bool {
	return strings.Contains(test_logger.w.(*flushBuffer).String(), str)
}

// debug_contents returns the specified log value as a string.
func debug_contents() string {
	return test_debug_logger.w.(*flushBuffer).String()
}

// debug_contains reports whether the string is contained in the log.
func debug_contains(str string, t *testing.T) bool {
	return strings.Contains(test_debug_logger.w.(*flushBuffer).String(), str)
}

// Test that Debug does not emit by default.
func TestDebug(t *testing.T) {
	test_logger.w = &flushBuffer{}
	test_logger.Debug("test")
	if contents() != "" {
		t.Errorf("Debug should not be emitted by default: %q", contents())
	}
}

// Test that Debug works if turned on.
func TestDebugOn(t *testing.T) {
	test_debug_logger = NewFromOptions(&Options{
		SyncWriter:   &flushBuffer{},
		IncludeDebug: true,
	})
	test_debug_logger.Debug("test")
	if !debug_contains("D", t) {
		t.Errorf("Info has wrong character: %q", debug_contents())
	}
	if !debug_contains("test", t) {
		t.Error("Info failed")
	}
}

// Test that Info works as advertised.
func TestInfo(t *testing.T) {
	test_logger.w = &flushBuffer{}
	test_logger.Info("test")
	if !contains("I", t) {
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test", t) {
		t.Error("Info failed")
	}
}

// Test that the header has the correct format.
func TestHeader(t *testing.T) {
	test_logger.w = &flushBuffer{}
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	test_logger.Info("test")
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
	test_logger.Info("test")
}

// Test that the header respects DepthDelta.
func TestDepthDelta(t *testing.T) {
	test_logger.w = &flushBuffer{}
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	test_logger.depthDelta = 1 // Should report a line in testing.go which calls this func.
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
	test_logger.depthDelta = 0
}

// Test that an Error log goes to Warning and Info.
// Even in the Info log, the source character will be E, so the data should
// all be identical.
func TestError(t *testing.T) {
	test_logger.w = &flushBuffer{}
	test_logger.Error("test")
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
	test_logger.w = &flushBuffer{}
	test_logger.Warning("test")
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
		buf, _, _ := test_logger.header(infoLog, 0)
		test_logger.putBuffer(buf)
	}
}
