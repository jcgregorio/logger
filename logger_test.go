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

// flushBuffer wraps a bytes.Buffer to satisfy SyncWriter.
type flushBuffer struct {
	bytes.Buffer
}

func (f *flushBuffer) Sync() error {
	return nil
}

// contents returns the specified log value as a string.
func contents() string {
	return Logger.w.(*flushBuffer).String()
}

// contains reports whether the string is contained in the log.
func contains(str string, t *testing.T) bool {
	return strings.Contains(Logger.w.(*flushBuffer).String(), str)
}

// Test that Info works as advertised.
func TestInfo(t *testing.T) {
	Logger.w = &flushBuffer{}
	Logger.Info("test")
	if !contains("I", t) {
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test", t) {
		t.Error("Info failed")
	}
}

// Test that the header has the correct format.
func TestHeader(t *testing.T) {
	Logger.w = &flushBuffer{}
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	Logger.Info("test")
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

// Test that an Error log goes to Warning and Info.
// Even in the Info log, the source character will be E, so the data should
// all be identical.
func TestError(t *testing.T) {
	Logger.w = &flushBuffer{}
	Logger.Error("test")
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
	Logger.w = &flushBuffer{}
	Logger.Warning("test")
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
		buf, _, _ := Logger.header(infoLog, 0)
		Logger.putBuffer(buf)
	}
}
