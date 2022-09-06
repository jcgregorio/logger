// Major parts of this implementation are copied verbatim from https://github.com/golang/glog.
//
package logger

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jcgregorio/slog"
)

var (
	pid = os.Getpid()

	// Capture os.Exit to be used for testing.
	osExit = os.Exit
)

type SyncWriter interface {
	Write(p []byte) (n int, err error)
	Sync() error
}

// severity identifies the sort of log: info, warning etc.
type severity int32

// These constants identify the log levels in order of increasing severity.
const (
	debugLog severity = iota
	infoLog
	warningLog
	errorLog
	fatalLog
	numSeverity = 4
)

const severityChar = "DIWEF"

var severityName = []string{
	debugLog:   "DEBUG",
	infoLog:    "INFO",
	warningLog: "WARNING",
	errorLog:   "ERROR",
	fatalLog:   "FATAL",
}

func New() *Logger {
	return &Logger{w: os.Stdout}
}

// Options is passed to NewFromOptions to control some aspects of the created
// Logger.
type Options struct {
	// SyncWriter is the destination to write logs to. If left nil then os.Stdout
	// will be used.
	SyncWriter SyncWriter

	// IncludeDebug is true will emit Debug/Debugf logs, otherwise those logs are ignored.
	IncludeDebug bool

	// DepthDelta is the number of extra stack levels to look up when reporting the calling function.
	//
	// Useful if Logger is going to be wrapped inside another logging module.
	DepthDelta int
}

func NewFromOptions(o *Options) *Logger {
	var w SyncWriter = os.Stdout
	if o.SyncWriter != nil {
		w = o.SyncWriter
	}
	return &Logger{
		w:            w,
		includeDebug: o.IncludeDebug,
		depthDelta:   o.DepthDelta,
	}
}

// Logger collects all the global state of the logging setup.
//
// *Logger implements the slog.Logger interface.
type Logger struct {
	w SyncWriter

	includeDebug bool

	// freeList is a list of byte buffers, maintained under freeListMu.
	freeList *buffer

	// freeListMu maintains the free list. It is separate from the main mutex
	// so buffers can be grabbed and printed to without holding the main lock,
	// for better parallelization.
	freeListMu sync.Mutex

	// DepthDelta is the number of extra stack levels to look up when reporting the calling function.
	depthDelta int
}

// buffer holds a byte Buffer for reuse. The zero value is ready for use.
type buffer struct {
	bytes.Buffer
	tmp  [64]byte // temporary byte array for creating headers.
	next *buffer
}

// getBuffer returns a new, ready-to-use buffer.
func (l *Logger) getBuffer() *buffer {
	l.freeListMu.Lock()
	b := l.freeList
	if b != nil {
		l.freeList = b.next
	}
	l.freeListMu.Unlock()
	if b == nil {
		b = new(buffer)
	} else {
		b.next = nil
		b.Reset()
	}
	return b
}

// putBuffer returns a buffer to the free list.
func (l *Logger) putBuffer(b *buffer) {
	if b.Len() >= 256 {
		// Let big buffers die a natural death.
		return
	}
	l.freeListMu.Lock()
	b.next = l.freeList
	l.freeList = b
	l.freeListMu.Unlock()
}

func (l *Logger) bufferCacheLen() int {
	ret := 0
	b := l.freeList
	for b != nil {
		ret++
		b = b.next
	}
	return ret
}

var timeNow = time.Now // Stubbed out for testing.

/*
header formats a log header as defined by the C++ implementation.
It returns a buffer containing the formatted header and the user's file and line number.
The depth specifies how many stack frames above lives the source line to be identified in the log message.

Log lines have this form:
	Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg...
where the fields are defined as follows:
	L                A single character, representing the log level (eg 'I' for INFO)
	mm               The month (zero padded; ie May is '05')
	dd               The day (zero padded)
	hh:mm:ss.uuuuuu  Time in hours, minutes and fractional seconds
	threadid         The space-padded thread ID as returned by GetTID()
	file             The file name
	line             The line number
	msg              The user-supplied message
*/
func (l *Logger) header(s severity, depth int) (*buffer, string, int) {
	_, file, line, ok := runtime.Caller(3 + depth + l.depthDelta)
	if !ok {
		file = "???"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}
	return l.formatHeader(s, file, line), file, line
}

// formatHeader formats a log header using the provided file name and line number.
func (l *Logger) formatHeader(s severity, file string, line int) *buffer {
	now := timeNow()
	if line < 0 {
		line = 0 // not a real line number, but acceptable to someDigits
	}
	if s > fatalLog {
		s = infoLog // for safety.
	}
	buf := l.getBuffer()

	// Avoid Fprintf, for speed. The format is so simple that we can do it quickly by hand.
	// It's worth about 3X. Fprintf is hard.
	_, month, day := now.Date()
	hour, minute, second := now.Clock()
	// Lmmdd hh:mm:ss.uuuuuu threadid file:line]
	buf.tmp[0] = severityChar[s]
	buf.twoDigits(1, int(month))
	buf.twoDigits(3, day)
	buf.tmp[5] = ' '
	buf.twoDigits(6, hour)
	buf.tmp[8] = ':'
	buf.twoDigits(9, minute)
	buf.tmp[11] = ':'
	buf.twoDigits(12, second)
	buf.tmp[14] = '.'
	buf.nDigits(6, 15, now.Nanosecond()/1000, '0')
	buf.tmp[21] = ' '
	buf.nDigits(7, 22, pid, ' ') // TODO: should be TID
	buf.tmp[29] = ' '
	buf.Write(buf.tmp[:30])
	buf.WriteString(file)
	buf.tmp[0] = ':'
	n := buf.someDigits(1, line)
	buf.tmp[n+1] = ']'
	buf.tmp[n+2] = ' '
	buf.Write(buf.tmp[:n+3])
	return buf
}

// Some custom tiny helper functions to print the log header efficiently.

const digits = "0123456789"

// twoDigits formats a zero-prefixed two-digit integer at buf.tmp[i].
func (buf *buffer) twoDigits(i, d int) {
	buf.tmp[i+1] = digits[d%10]
	d /= 10
	buf.tmp[i] = digits[d%10]
}

// nDigits formats an n-digit integer at buf.tmp[i],
// padding with pad on the left.
// It assumes d >= 0.
func (buf *buffer) nDigits(n, i, d int, pad byte) {
	j := n - 1
	for ; j >= 0 && d > 0; j-- {
		buf.tmp[i+j] = digits[d%10]
		d /= 10
	}
	for ; j >= 0; j-- {
		buf.tmp[i+j] = pad
	}
}

// someDigits formats a zero-prefixed variable-width integer at buf.tmp[i].
func (buf *buffer) someDigits(i, d int) int {
	// Print into the top, then copy down. We know there's space for at least
	// a 10-digit number.
	j := len(buf.tmp)
	for {
		j--
		buf.tmp[j] = digits[d%10]
		d /= 10
		if d == 0 {
			break
		}
	}
	return copy(buf.tmp[i:], buf.tmp[j:])
}

func (l *Logger) print(s severity, args ...interface{}) {
	l.printDepth(s, 1, args...)
}

func (l *Logger) printDepth(s severity, depth int, args ...interface{}) {
	header, _, _ := l.header(s, depth)

	buf := l.getBuffer()

	fmt.Fprint(buf, args...)
	l.emitAsOneOrMoreLogLines(s, buf, header)
	l.putBuffer(buf)
}

func (l *Logger) printf(s severity, format string, args ...interface{}) {
	header, _, _ := l.header(s, 0)
	buf := l.getBuffer()

	fmt.Fprintf(buf, format, args...)

	l.emitAsOneOrMoreLogLines(s, buf, header)
	l.putBuffer(buf)
}

func (l *Logger) emitAsOneOrMoreLogLines(s severity, buf, header *buffer) {
	// At this point buf could contain multiple embedded \n's, so we need to slice it up
	// into multiple lines and emit each line separately.
	l.emitAsOneOrMoreLogLinesImpl(buf, header)

	if s == fatalLog {
		// If this is fatal then grab a strack trace and emit and also fatal
		// error log entries.
		trace := stacks(true)

		buf := l.getBuffer()
		buf.Write(trace)
		l.emitAsOneOrMoreLogLinesImpl(buf, header)

		l.w.Sync()
		osExit(255) // C++ uses -1, which is silly because it's anded with 255 anyway.
	}
}

func (l *Logger) emitAsOneOrMoreLogLinesImpl(buf, header *buffer) {
	// At this point buf could contain multiple embedded \n's, so we need to slice it up
	// into multiple lines and emit each line separately.
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for _, pline := range lines {
		// Don't emit blank lines.
		if len(pline) == 0 {
			continue
		}

		// Writes need to happen as a single call, so concatenate all the data
		// we want to write as a single line and the write that buffer out.
		buf := l.getBuffer()
		buf.Write(header.Bytes())
		buf.Write(pline)
		buf.Write([]byte("\n"))

		l.w.Write(buf.Bytes())

		l.putBuffer(buf)
	}
}

// stacks is a wrapper for runtime.Stack that attempts to recover the data for all goroutines.
func stacks(all bool) []byte {
	// We don't know how big the traces are, so grow a few times if they don't fit. Start large, though.
	n := 10000
	if all {
		n = 100000
	}
	var trace []byte
	for i := 0; i < 5; i++ {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			return trace[:nbytes]
		}
		n *= 2
	}
	return trace
}

func (l *Logger) Debug(args ...interface{}) {
	if l.includeDebug {
		l.print(debugLog, args...)
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.includeDebug {
		l.printf(debugLog, format, args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.print(infoLog, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.printf(infoLog, format, args...)
}

func (l *Logger) Warning(args ...interface{}) {
	l.print(warningLog, args...)
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	l.printf(warningLog, format, args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.print(errorLog, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.printf(errorLog, format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.print(fatalLog, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.printf(fatalLog, format, args...)
}

func (l *Logger) Raw(s string) {
	l.w.Write([]byte(s))
	if s[len(s)-1] != '\n' {
		l.w.Write([]byte{'\n'})
	}
}

// Assert that we implement the slog.Logger interface:
var _ slog.Logger = (*Logger)(nil)
