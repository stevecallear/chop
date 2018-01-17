package chop_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/stevecallear/chop"
)

func TestStart(t *testing.T) {
	t.Run("should start the lambda", func(t *testing.T) {
		exp := events.APIGatewayProxyResponse{
			StatusCode: http.StatusCreated,
			Body:       "expected",
			Headers:    map[string]string{},
		}
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			os.Setenv("_LAMBDA_SERVER_PORT", "8081")
			chop.Start(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(exp.StatusCode)
				fmt.Fprintf(w, exp.Body)
			}))
		}()
		go func() {
			req := events.APIGatewayProxyRequest{}
			act, err := invokeLocal("8081", req)
			if err != nil {
				t.Errorf("got %v, expected nil", err)
			}
			if !reflect.DeepEqual(act, exp) {
				t.Errorf("got %v, expected %v", act, exp)
			}
			wg.Done()
		}()
		wg.Wait()
	})
}

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name  string
		event events.APIGatewayProxyRequest
		path  string
		code  int
		err   bool
		exp   events.APIGatewayProxyResponse
	}{
		{
			name: "should return an error if the path is invalid",
			event: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/resource###%",
			},
			err: true,
		},
		{
			name: "should handle the event",
			event: events.APIGatewayProxyRequest{
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
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusCreated,
				Body:       "body",
				Headers: map[string]string{
					"X-Custom-Header": "header",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			act, err := chop.Wrap(fn).Handle(tt.event)
			if err != nil && !tt.err {
				t.Errorf("got %v, expected nil", err)
			}
			if err == nil && tt.err {
				t.Errorf("got nil, expected an error")
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(act, tt.exp) {
				t.Errorf("got %v, expected %v", act, tt.exp)
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
		exp     events.APIGatewayProxyResponse
	}{
		{
			name:    "should set default status code if not called",
			data:    [][]byte{},
			headers: make(map[string]string),
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    make(map[string]string),
			},
		},
		{
			name:    "should set default status code and headers",
			data:    [][]byte{[]byte("body")},
			headers: make(map[string]string),
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
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
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusCreated,
				Headers:    make(map[string]string),
				Body:       "body",
			},
		},
		{
			name: "should not overwrite existing content type header",
			data: [][]byte{[]byte("body")},
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
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
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
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
			exp: events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
				Body: "abc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			act := w.Result()
			if !reflect.DeepEqual(act, tt.exp) {
				t.Errorf("got %v, expected %v", act, tt.exp)
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
		t.Run(tt.name, func(t *testing.T) {
			w := chop.NewResponseWriter()
			for _, c := range tt.codes {
				w.WriteHeader(c)
			}
			act := w.Result().StatusCode
			if act != tt.exp {
				t.Errorf("got %d, expected %d", act, tt.exp)
			}
		})
	}
}

func TestGetEvent(t *testing.T) {
	t.Run("should return the integration event", func(t *testing.T) {
		exp := events.APIGatewayProxyRequest{
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
		}
		fn := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			act := chop.GetEvent(r)
			if !reflect.DeepEqual(act, exp) {
				t.Errorf("got %v, expected %v", act, exp)
			}
		})
		_, err := chop.Wrap(fn).Handle(exp)
		if err != nil {
			t.Errorf("got %v, expected nil", err)
		}
	})
}

func invokeLocal(port string, e events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%s", port))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	defer client.Close()
	payload, err := json.Marshal(&e)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	invReq := messages.InvokeRequest{Payload: payload}
	invRes := messages.InvokeResponse{}
	err = client.Call("Function.Invoke", &invReq, &invRes)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	if invRes.Error != nil {
		return events.APIGatewayProxyResponse{}, errors.New(invRes.Error.Message)
	}
	res := events.APIGatewayProxyResponse{}
	err = json.Unmarshal(invRes.Payload, &res)
	return res, err
}
