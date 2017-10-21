package chop

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/apigatewayproxyevt"
)

type (
	// Result represents a proxy integration result
	Result struct {
		Code    int               `json:"statusCode"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}

	// Handler represents a lambda event handler
	Handler struct {
		http.Handler
	}

	// ResponseWriter represents a proxy integration response writer
	ResponseWriter struct {
		code        int
		header      http.Header
		buffer      *bytes.Buffer
		wroteHeader bool
	}

	contextKey string
)

var (
	runtimeKey = contextKey("runtime")
	eventKey   = contextKey("event")
)

// Wrap wraps the specified HTTP handler with a lambda event handler
func Wrap(h http.Handler) *Handler {
	return &Handler{
		Handler: h,
	}
}

// Handle converts the lambda event into an HTTP request
func (h *Handler) Handle(e *apigatewayproxyevt.Event, c *runtime.Context) (*Result, error) {
	r, err := NewRequest(e)
	if err != nil {
		return nil, err
	}
	w := NewResponseWriter()
	h.ServeHTTP(w, withContext(withEvent(r, e), c))
	return w.Result(), nil
}

// NewRequest parses the integration event and returns a new HTTP request
func NewRequest(e *apigatewayproxyevt.Event) (*http.Request, error) {
	r, err := http.NewRequest(strings.ToUpper(e.HTTPMethod), e.Path, bytes.NewBuffer([]byte(e.Body)))
	if err != nil {
		return nil, err
	}
	q := r.URL.Query()
	for k, v := range e.QueryStringParameters {
		q.Add(k, v)
	}
	r.URL.RawQuery = q.Encode()
	for k, v := range e.Headers {
		r.Header.Add(k, v)
	}
	return r, nil
}

// NewResponseWriter returns a new ResponseWriter
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		header: make(http.Header),
		buffer: new(bytes.Buffer),
	}
}

// Header returns the HTTP headers
func (w *ResponseWriter) Header() http.Header {
	return w.header
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	w.writeHeader(b)
	w.buffer.Write(b)
	return len(b), nil
}

// WriteHeader writes the specified status if the header has not been written
func (w *ResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.code = code
	w.wroteHeader = true
}

// Result returns a proxy integration result for the response
func (w *ResponseWriter) Result() *Result {
	h := make(map[string]string, len(w.header))
	for k := range w.header {
		h[k] = w.header.Get(k)
	}
	r := Result{
		Code:    w.code,
		Headers: h,
		Body:    w.buffer.String(),
	}
	if r.Code == 0 {
		r.Code = http.StatusOK
	}
	return &r
}

func (w *ResponseWriter) writeHeader(b []byte) {
	if w.wroteHeader {
		return
	}
	m := w.Header()
	if _, ct := m["Content-Type"]; !ct && m.Get("Transfer-Encoding") == "" {
		m.Set("Content-Type", http.DetectContentType(b))
	}
	w.WriteHeader(http.StatusOK)
}

// GetEvent returns a pointer to the proxy integration event
func GetEvent(r *http.Request) *apigatewayproxyevt.Event {
	return r.Context().Value(eventKey).(*apigatewayproxyevt.Event)
}

// GetContext returns the lambda runtime context
func GetContext(r *http.Request) *runtime.Context {
	return r.Context().Value(runtimeKey).(*runtime.Context)
}

func withEvent(r *http.Request, e *apigatewayproxyevt.Event) *http.Request {
	ctx := context.WithValue(r.Context(), eventKey, e)
	return r.WithContext(ctx)
}

func withContext(r *http.Request, c *runtime.Context) *http.Request {
	ctx := context.WithValue(r.Context(), runtimeKey, c)
	return r.WithContext(ctx)
}
