package testutil

import (
	"bytes"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/kmio11/cslog"
)

type AssertHandler struct {
	buf *bytes.Buffer
	slog.Handler
}

type AssertHandlerOpts struct {
	Json       bool
	RemoveTime bool
}

func NewAssertHandler(t *testing.T, opts AssertHandlerOpts) *AssertHandler {
	t.Helper()

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		ret := a
		if opts.RemoveTime {
			ret = RemoveTime(groups, a)
		}
		return ret
	}

	buf := new(bytes.Buffer)

	var innerHandler slog.Handler
	if opts.Json {
		innerHandler = slog.NewJSONHandler(
			buf,
			&slog.HandlerOptions{
				ReplaceAttr: replaceAttr,
				AddSource:   false,
				Level:       cslog.LogLevel(),
			},
		)
	} else {
		innerHandler = slog.NewTextHandler(
			buf,
			&slog.HandlerOptions{
				ReplaceAttr: replaceAttr,
				AddSource:   false,
				Level:       cslog.LogLevel(),
			},
		)
	}

	return &AssertHandler{
		buf:     buf,
		Handler: innerHandler,
	}
}

func (h *AssertHandler) Buf(t *testing.T) *bytes.Buffer {
	t.Helper()
	return h.buf
}

func (h *AssertHandler) ResetBuf(t *testing.T) {
	t.Helper()
	h.buf.Reset()
}

func (h *AssertHandler) Check(t *testing.T, wantRegexp string) {
	t.Helper()
	checkLogOutput(t, h.buf.String(), wantRegexp)
	h.ResetBuf(t)
}

func checkLogOutput(t *testing.T, got, wantRegexp string) {
	t.Helper()
	got = clean(got)
	wantRegexp = "^" + wantRegexp + "$"
	matched, err := regexp.MatchString(wantRegexp, got)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("\ngot  %s\nwant %s", got, wantRegexp)
	}
}

// clean prepares log output for comparison.
func clean(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return strings.ReplaceAll(s, "\n", "~")
}
