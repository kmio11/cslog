package testutil

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/kmio11/cslog"
)

type bufHandler struct {
	buf *bytes.Buffer
	slog.Handler
}

type BufTextHandler struct {
	bufHandler
}

type BufJSONHandler struct {
	bufHandler
}

type BufHandlerOpts struct {
	RemoveTime bool
	AddSource  bool
}

// RemoveTime removes the top-level time attribute.
// It is intended to be used as a ReplaceAttr function,
// to make example output deterministic.
func RemoveTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func replaceAttr(t *testing.T, opts BufHandlerOpts) func(groups []string, a slog.Attr) slog.Attr {
	t.Helper()

	return func(groups []string, a slog.Attr) slog.Attr {
		ret := a
		if opts.RemoveTime {
			ret = RemoveTime(groups, a)
		}
		return ret
	}
}

func newBufHandler(t *testing.T, buf *bytes.Buffer, handler slog.Handler) *bufHandler {
	t.Helper()

	return &bufHandler{
		buf:     buf,
		Handler: handler,
	}
}

func NewBufTextHandler(t *testing.T, opts BufHandlerOpts) *BufTextHandler {
	buf := new(bytes.Buffer)

	innerHandler := slog.NewTextHandler(
		buf,
		&slog.HandlerOptions{
			ReplaceAttr: replaceAttr(t, opts),
			AddSource:   opts.AddSource,
			Level:       cslog.LogLevel(),
		},
	)

	return &BufTextHandler{
		bufHandler: *newBufHandler(t, buf, innerHandler),
	}
}

func NewBufJSONHandler(t *testing.T, opts BufHandlerOpts) *BufJSONHandler {
	buf := new(bytes.Buffer)

	innerHandler := slog.NewJSONHandler(
		buf,
		&slog.HandlerOptions{
			ReplaceAttr: replaceAttr(t, opts),
			AddSource:   opts.AddSource,
			Level:       cslog.LogLevel(),
		},
	)

	return &BufJSONHandler{
		bufHandler: *newBufHandler(t, buf, innerHandler),
	}
}

func (h *bufHandler) Buf(t *testing.T) *bytes.Buffer {
	t.Helper()
	return h.buf
}

func (h *bufHandler) ResetBuf(t *testing.T) {
	t.Helper()
	h.buf.Reset()
}

func (h *BufTextHandler) Check(t *testing.T, wantRegexp string) {
	t.Helper()

	// clean prepares log output for comparison.
	clean := func(s string) string {
		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}
		return strings.ReplaceAll(s, "\n", "~")
	}

	checkLogOutput := func(t *testing.T, got, wantRegexp string) {
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

	checkLogOutput(t, h.Buf(t).String(), wantRegexp)
	h.ResetBuf(t)
}

func (h *BufJSONHandler) Object(t *testing.T) map[string]any {
	j := h.Buf(t).Bytes()
	o := map[string]any{}
	if err := json.Unmarshal(j, &o); err != nil {
		t.Fatalf("invalid json. err: [%s], log: [%s]", err.Error(), j)
	}
	return o
}

func (h *BufJSONHandler) Objects(t *testing.T) []map[string]any {
	jsons := strings.Split(strings.Trim(h.Buf(t).String(), "\n"), "\n")
	objs := []map[string]any{}

	for _, j := range jsons {
		o := map[string]any{}
		if err := json.Unmarshal([]byte(j), &o); err != nil {
			t.Fatalf("invalid json. err: [%s], log: [%s]", err.Error(), j)
		}
		objs = append(objs, o)
	}
	return objs
}

func TypedJSONObject[T any](t *testing.T, obj any) *T {
	t.Helper()

	j, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}

	typed := new(T)
	err = json.Unmarshal(j, typed)
	if err != nil {
		t.Fatal(err)
	}

	return typed
}
