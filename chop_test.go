package chop_test

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/apigatewayproxyevt"
	"github.com/stevecallear/chop"
)

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name  string
		event apigatewayproxyevt.Event
		path  string
		code  int
		err   bool
		exp   chop.Result
	}{
		{
			name: "should return an error if the path is invalid",
			event: apigatewayproxyevt.Event{
				HTTPMethod: "GET",
				Path:       "/resource###%",
			},
			err: true,
		},
		{
			name: "should handle the event",
			event: apigatewayproxyevt.Event{
				HTTPMethod: "GET",
				Path:       "/resource",
				QueryStringParameters: map[string]string{
					"a": "1",
					"b": "2",
				},
				Body: "body",
				Headers: map[string]string{
					"X-Custom-Header": "header",
				},
			},
			path: "/resource?a=1&b=2",
			code: http.StatusCreated,
			exp: chop.Result{
				Code: http.StatusCreated,
				Body: "body",
				Headers: map[string]string{
					"X-Custom-Header": "header",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.String() != tt.path {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(tt.code)
				b, _ := ioutil.ReadAll(r.Body)
				w.Write(b)
				for k := range r.Header {
					w.Header().Add(k, r.Header.Get(k))
				}
			})
			r, err := chop.Wrap(fn).Handle(&tt.event, nil)
			if err != nil && !tt.err {
				st.Errorf("got %v, expected nil", err)
			}
			if err == nil && tt.err {
				st.Errorf("got nil, expected an error")
			}
			if err != nil {
				return
			}
			act := *r
			if !reflect.DeepEqual(act, tt.exp) {
				st.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		data    [][]byte
		headers map[string]string
		exp     chop.Result
	}{
		{
			name:    "should set default status code if not called",
			data:    [][]byte{},
			headers: make(map[string]string),
			exp: chop.Result{
				Code:    http.StatusOK,
				Headers: make(map[string]string),
			},
		},
		{
			name:    "should set default status code and headers",
			data:    [][]byte{[]byte("body")},
			headers: make(map[string]string),
			exp: chop.Result{
				Code: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
				Body: "body",
			},
		},
		{
			name:    "should not overwrite existing status code",
			code:    http.StatusCreated,
			data:    [][]byte{[]byte("body")},
			headers: make(map[string]string),
			exp: chop.Result{
				Code:    http.StatusCreated,
				Headers: make(map[string]string),
				Body:    "body",
			},
		},
		{
			name: "should not overwrite existing content type header",
			data: [][]byte{[]byte("body")},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			exp: chop.Result{
				Code: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: "body",
			},
		},
		{
			name: "should not write content type header if transfer encoding is set",
			data: [][]byte{[]byte("body")},
			headers: map[string]string{
				"Transfer-Encoding": "gzip",
			},
			exp: chop.Result{
				Code: http.StatusOK,
				Headers: map[string]string{
					"Transfer-Encoding": "gzip",
				},
				Body: "body",
			},
		},
		{
			name:    "should permit multiple writes",
			data:    [][]byte{[]byte("a"), []byte("b"), []byte("c")},
			headers: make(map[string]string),
			exp: chop.Result{
				Code: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
				Body: "abc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			w := chop.NewResponseWriter()
			if tt.code != 0 {
				w.WriteHeader(tt.code)
			}
			for k, v := range tt.headers {
				w.Header().Add(k, v)
			}
			for _, d := range tt.data {
				w.Write(d)
			}
			r := w.Result()
			act := *r
			if !reflect.DeepEqual(act, tt.exp) {
				st.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		name  string
		codes []int
		exp   int
	}{
		{
			name:  "should use the value",
			codes: []int{http.StatusBadRequest},
			exp:   http.StatusBadRequest,
		},
		{
			name:  "should use the first value",
			codes: []int{http.StatusBadRequest, http.StatusOK},
			exp:   http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			w := chop.NewResponseWriter()
			for _, c := range tt.codes {
				w.WriteHeader(c)
			}
			act := w.Result().Code
			if act != tt.exp {
				st.Errorf("got %d, expected %d", act, tt.exp)
			}
		})
	}
}

func TestGetEvent(t *testing.T) {
	t.Run("should return the integration event", func(st *testing.T) {
		exp := new(apigatewayproxyevt.Event)
		var act *apigatewayproxyevt.Event
		fn := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			act = chop.GetEvent(r)
		})
		_, err := chop.Wrap(fn).Handle(exp, nil)
		if err != nil {
			st.Errorf("got %v, expected nil", err)
		}
		if act != exp {
			st.Errorf("got %v, expected %v", act, exp)
		}
	})
}

func TestGetContext(t *testing.T) {
	t.Run("should return the runtime context", func(st *testing.T) {
		exp := new(runtime.Context)
		var act *runtime.Context
		fn := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			act = chop.GetContext(r)
		})
		evt := apigatewayproxyevt.Event{
			HTTPMethod: "GET",
			Path:       "/",
		}
		_, err := chop.Wrap(fn).Handle(&evt, exp)
		if err != nil {
			st.Errorf("got %v, expected nil", err)
		}
		if act != exp {
			st.Errorf("got %v, expected %v", act, exp)
		}
	})
}
