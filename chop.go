package chop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/tidwall/gjson"
)

type (
	// Handler represents a lambda event handler
	Handler struct {
		http.Handler
	}

	// ResponseWriter represents a lambda event response writer
	ResponseWriter struct {
		code        int
		buffer      *bytes.Buffer
		header      http.Header
		wroteHeader bool
	}

	eventProcessor struct {
		canProcess       func([]byte) bool
		unmarshalRequest func(context.Context, []byte) (*http.Request, error)
		marshalResponse  func(*ResponseWriter) ([]byte, error)
	}

	eventContextKey struct{}
)

var (
	// ErrUnsupportedEventType indicates that the received lambda event is not supported
	ErrUnsupportedEventType = errors.New("unsupported lambda event type")

	apiGatewayProxyEventProcessor = &eventProcessor{
		canProcess: func(payload []byte) bool {
			pv := gjson.GetManyBytes(payload, "version", "requestContext.apiId")
			return !pv[0].Exists() && pv[1].Exists()
		},
		unmarshalRequest: func(ctx context.Context, payload []byte) (*http.Request, error) {
			e := new(events.APIGatewayProxyRequest)
			if err := json.Unmarshal(payload, e); err != nil {
				return nil, err
			}

			r, err := http.NewRequest(e.HTTPMethod, e.Path, bytes.NewBufferString(e.Body))
			if err != nil {
				return nil, err
			}

			q := r.URL.Query()
			addMapValues(e.QueryStringParameters, e.MultiValueQueryStringParameters, q.Add)
			r.URL.RawQuery = q.Encode()

			addMapValues(e.Headers, e.MultiValueHeaders, r.Header.Add)

			return WithEvent(r.WithContext(ctx), e), nil
		},
		marshalResponse: func(w *ResponseWriter) ([]byte, error) {
			return json.Marshal(&events.APIGatewayProxyResponse{
				StatusCode:        w.StatusCode(),
				Headers:           reduceHeaders(w.Header()),
				MultiValueHeaders: w.Header(),
				Body:              w.Body(),
			})
		},
	}

	albTargetGroupEventProcessor = &eventProcessor{
		canProcess: func(payload []byte) bool {
			return gjson.GetBytes(payload, "requestContext.elb").Exists()
		},
		unmarshalRequest: func(ctx context.Context, payload []byte) (*http.Request, error) {
			e := new(events.ALBTargetGroupRequest)
			if err := json.Unmarshal(payload, e); err != nil {
				return nil, err
			}

			r, err := http.NewRequest(e.HTTPMethod, e.Path, bytes.NewBufferString(e.Body))
			if err != nil {
				return nil, err
			}

			q := r.URL.Query()
			addMapValues(e.QueryStringParameters, e.MultiValueQueryStringParameters, q.Add)
			r.URL.RawQuery = q.Encode()

			addMapValues(e.Headers, e.MultiValueHeaders, r.Header.Add)

			return WithEvent(r.WithContext(ctx), e), nil
		},
		marshalResponse: func(w *ResponseWriter) ([]byte, error) {
			return json.Marshal(&events.ALBTargetGroupResponse{
				StatusCode:        w.StatusCode(),
				StatusDescription: w.Status(),
				Headers:           reduceHeaders(w.Header()),
				MultiValueHeaders: w.Header(),
				Body:              w.Body(),
			})
		},
	}
)

// Start wraps and starts the specified HTTP handler as a lambda function handler
func Start(h http.Handler) {
	lambda.StartHandler(Wrap(h))
}

// Wrap wraps the specified HTTP handler as a lambda function handler
func Wrap(h http.Handler) *Handler {
	return &Handler{
		Handler: h,
	}
}

// Invoke invokes the lambda function handler
func (h *Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	p, err := getEventProcessor(payload)
	if err != nil {
		return nil, err
	}

	r, err := p.unmarshalRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	w := NewResponseWriter()
	h.ServeHTTP(w, r)

	return p.marshalResponse(w)
}

// NewResponseWriter returns a new ResponseWriter
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		code:   200,
		buffer: new(bytes.Buffer),
		header: http.Header{},
	}
}

// Status returns the HTTP status as a string
func (w *ResponseWriter) Status() string {
	return fmt.Sprintf("%d %s", w.code, http.StatusText(w.code))
}

// StatusCode returns the HTTP status code
func (w *ResponseWriter) StatusCode() int {
	return w.code
}

// Body returns the response body as a string
func (w *ResponseWriter) Body() string {
	return w.buffer.String()
}

// Header returns the response headers
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

// WithEvent returns a copy of the request with the specified event stored in the request context
func WithEvent(r *http.Request, event interface{}) *http.Request {
	ctx := context.WithValue(r.Context(), eventContextKey{}, event)

	return r.WithContext(ctx)
}

// GetEvent returns the lambda event stored within the specified request context if it exists
func GetEvent(r *http.Request) interface{} {
	return r.Context().Value(eventContextKey{})
}

func getEventProcessor(payload []byte) (*eventProcessor, error) {
	for _, p := range []*eventProcessor{
		apiGatewayProxyEventProcessor,
		albTargetGroupEventProcessor,
	} {
		if p.canProcess(payload) {
			return p, nil
		}
	}

	return nil, ErrUnsupportedEventType
}

func addMapValues(values map[string]string, multiValues map[string][]string, addFn func(string, string)) {
	if len(multiValues) > 1 {
		for k, mv := range multiValues {
			for _, v := range mv {
				addFn(k, v)
			}
		}

		return
	}

	for k, v := range values {
		addFn(k, v)
	}
}

func reduceHeaders(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k := range h {
		m[k] = h.Get(k)
	}

	return m
}
