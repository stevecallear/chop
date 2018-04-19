package chop

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

type (
	// Handler represents a proxy integration event handler
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

var eventContextKey = contextKey("eventContextKey")

// Start wraps and starts specified HTTP handler as a proxy integration event handler
func Start(h http.Handler) {
	lambda.Start(Wrap(h).Handle)
}

// Wrap wraps the specified HTTP handler with a proxy integration event handler
func Wrap(h http.Handler) *Handler {
	return &Handler{
		Handler: h,
	}
}

// Handle dispatches the integration event as an HTTP request to the wrapped handler
func (h *Handler) Handle(c context.Context, e events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	r, err := NewRequest(c, e)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	w := NewResponseWriter()
	h.ServeHTTP(w, WithEvent(r, e))
	return w.Result(), nil
}

// NewRequest parses the integration event and returns a new HTTP request
func NewRequest(c context.Context, e events.APIGatewayProxyRequest) (*http.Request, error) {
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
	return r.WithContext(c), nil
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
func (w *ResponseWriter) Result() events.APIGatewayProxyResponse {
	h := make(map[string]string, len(w.header))
	for k := range w.header {
		h[k] = w.header.Get(k)
	}
	r := events.APIGatewayProxyResponse{
		StatusCode: w.code,
		Headers:    h,
		Body:       w.buffer.String(),
	}
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	return r
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

// GetEvent returns a copy of the proxy integration event if it exists
func GetEvent(r *http.Request) (events.APIGatewayProxyRequest, bool) {
	e, ok := r.Context().Value(eventContextKey).(events.APIGatewayProxyRequest)
	return e, ok
}

// GetContext returns a copy of the lambda context if it exists
func GetContext(r *http.Request) (lambdacontext.LambdaContext, bool) {
	if c, ok := lambdacontext.FromContext(r.Context()); ok {
		return *c, true
	}
	return lambdacontext.LambdaContext{}, false
}

// WithEvent returns a copy of the request with the specified event stored in the request context
// The function is exported to simplify testing for apps that use GetEvent
func WithEvent(r *http.Request, e events.APIGatewayProxyRequest) *http.Request {
	ctx := context.WithValue(r.Context(), eventContextKey, e)
	return r.WithContext(ctx)
}
