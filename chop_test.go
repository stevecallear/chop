package chop_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/rpc"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/aws/aws-lambda-go/lambdacontext"

	"github.com/stevecallear/chop/v2"
)

func TestStart(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := lambdacontext.FromContext(r.Context())
		e := chop.GetEvent(r)

		w.Write([]byte(fmt.Sprintf("%T|%T", c, e)))
	})

	tests := []struct {
		name    string
		payload string
		act     interface{}
		exp     interface{}
	}{
		{
			name:    "should handle api gateway proxy events",
			payload: apiGatewayProxyEventPayload,
			act:     new(events.APIGatewayProxyResponse),
			exp: &events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type": {"text/plain; charset=utf-8"},
				},
				Body: "*lambdacontext.LambdaContext|*events.APIGatewayProxyRequest",
			},
		},
		{
			name:    "should handle api gateway http v2 events",
			payload: apiGatewayV2HTTPEventPayload,
			act:     new(events.APIGatewayV2HTTPResponse),
			exp: &events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type": {"text/plain; charset=utf-8"},
				},
				Body:    "*lambdacontext.LambdaContext|*events.APIGatewayV2HTTPRequest",
				Cookies: []string{},
			},
		},
		{
			name:    "should handle alb target group events",
			payload: albTargetGroupSingleValueEventPayload,
			act:     new(events.ALBTargetGroupResponse),
			exp: &events.ALBTargetGroupResponse{
				StatusCode:        http.StatusOK,
				StatusDescription: toStatusDescription(http.StatusOK),
				Headers: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type": {"text/plain; charset=utf-8"},
				},
				Body: "*lambdacontext.LambdaContext|*events.ALBTargetGroupRequest",
			},
		},
	}

	go func() {
		os.Setenv("_LAMBDA_SERVER_PORT", "8081")
		chop.Start(handler)
	}()

	time.Sleep(100 * time.Millisecond) // allow the handler to start

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := invokeLocal("8081", []byte(tt.payload))
			if err != nil {
				t.Errorf("got %v, expected nil", err)
				t.FailNow()
			}

			if err = json.Unmarshal(b, tt.act); err != nil {
				t.Errorf("got %v, expected nil", err)
			}

			if !reflect.DeepEqual(tt.act, tt.exp) {
				t.Errorf("got %v, expected %v", tt.act, tt.exp)
			}
		})
	}
}

func TestHandler_Invoke(t *testing.T) {
	tests := []struct {
		name      string
		handlerFn func(*testing.T) http.HandlerFunc
		payload   string
		err       bool
		act       interface{}
		exp       interface{}
	}{
		{
			name:    "should return an error if the event is invalid",
			payload: `{}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should return an error if the api gateway proxy event cannot be unmarshalled",
			payload: `{"requestContext":{"apiId":"id"},"resource":"a}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should return an error if the api gateway proxy event path is invalid",
			payload: `{"httpMethod":"GET","path":"/resource###%","requestContext":{"apiId":"id"}}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should handle api gateway proxy events",
			payload: apiGatewayProxyEventPayload,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					exp := request{
						method: "GET",
						url:    "/resource/?q1=v1&q2=v2&q2=v3",
						body:   "body",
						header: http.Header{
							"X-Custom-Header1": {"v1"},
							"X-Custom-Header2": {"v2", "v3"},
						},
					}

					act := toRequest(r)

					if !reflect.DeepEqual(act, exp) {
						t.Errorf("got %v, expected %v", act, exp)
					}

					w.Header().Add("X-Custom-Header", "v1")
					w.Header().Add("X-Custom-Header", "v2")
					w.Write([]byte("body"))
				}
			},
			act: &events.APIGatewayProxyResponse{},
			exp: &events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type":    "text/plain; charset=utf-8",
					"X-Custom-Header": "v1",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type":    {"text/plain; charset=utf-8"},
					"X-Custom-Header": {"v1", "v2"},
				},
				Body: "body",
			},
		},
		{
			name:    "should return an error if the api gateway http v2 event cannot be unmarshalled",
			payload: `{"version":"2.0","requestContext":{"apiId":"id"},"resource":"a}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should return an error if the api gateway http v2 event path is invalid",
			payload: `{"version":"2.0","httpMethod":"GET","rawPath":"/resource###%","requestContext":{"apiId":"id"}}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should handle api gateway http v2 events",
			payload: apiGatewayV2HTTPEventPayload,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					exp := request{
						method: "GET",
						url:    "/resource/?q1=v1&q2=v2&q2=v3",
						body:   "body",
						header: http.Header{
							"X-Custom-Header1": {"v1"},
							"X-Custom-Header2": {"v2"},
						},
					}

					act := toRequest(r)

					if !reflect.DeepEqual(act, exp) {
						t.Errorf("got %v, expected %v", act, exp)
					}

					w.Header().Add("X-Custom-Header", "v1")
					w.Header().Add("X-Custom-Header", "v2")
					w.Write([]byte("body"))
				}
			},
			act: &events.APIGatewayV2HTTPResponse{},
			exp: &events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type":    "text/plain; charset=utf-8",
					"X-Custom-Header": "v1",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type":    {"text/plain; charset=utf-8"},
					"X-Custom-Header": {"v1", "v2"},
				},
				Body:    "body",
				Cookies: []string{},
			},
		},
		{
			name:    "should return an error if the alb target group event cannot be unmarshalled",
			payload: `{"requestContext":{"elb":{}},"resource":"a}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should return an error if the alb target group event path is invalid",
			payload: `{"httpMethod":"GET","path":"/resource###%","requestContext":{"elb":{}}}`,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
			err: true,
		},
		{
			name:    "should handle alb target group single value events",
			payload: albTargetGroupSingleValueEventPayload,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					exp := request{
						method: "GET",
						url:    "/resource/?q1=v1&q2=v2",
						body:   "body",
						header: http.Header{
							"X-Custom-Header1": {"v1"},
							"X-Custom-Header2": {"v2"},
						},
					}

					act := toRequest(r)

					if !reflect.DeepEqual(act, exp) {
						t.Errorf("got %v, expected %v", act, exp)
					}

					w.Header().Add("X-Custom-Header", "v1")
					w.Header().Add("X-Custom-Header", "v2")
					w.Write([]byte("body"))
				}
			},
			act: &events.ALBTargetGroupResponse{},
			exp: &events.ALBTargetGroupResponse{
				StatusCode:        http.StatusOK,
				StatusDescription: toStatusDescription(http.StatusOK),
				Headers: map[string]string{
					"Content-Type":    "text/plain; charset=utf-8",
					"X-Custom-Header": "v1",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type":    {"text/plain; charset=utf-8"},
					"X-Custom-Header": {"v1", "v2"},
				},
				Body: "body",
			},
		},
		{
			name:    "should handle alb target group multi value events",
			payload: albTargetGroupMultiValueEventPayload,
			handlerFn: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					exp := request{
						method: "GET",
						url:    "/resource/?q1=v1&q2=v2&q2=v3",
						body:   "body",
						header: http.Header{
							"X-Custom-Header1": {"v1"},
							"X-Custom-Header2": {"v2", "v3"},
						},
					}

					act := toRequest(r)

					if !reflect.DeepEqual(act, exp) {
						t.Errorf("got %v, expected %v", act, exp)
					}

					w.Header().Add("X-Custom-Header", "v1")
					w.Header().Add("X-Custom-Header", "v2")
					w.Write([]byte("body"))
				}
			},
			act: &events.ALBTargetGroupResponse{},
			exp: &events.ALBTargetGroupResponse{
				StatusCode:        http.StatusOK,
				StatusDescription: toStatusDescription(http.StatusOK),
				Headers: map[string]string{
					"Content-Type":    "text/plain; charset=utf-8",
					"X-Custom-Header": "v1",
				},
				MultiValueHeaders: map[string][]string{
					"Content-Type":    {"text/plain; charset=utf-8"},
					"X-Custom-Header": {"v1", "v2"},
				},
				Body: "body",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.handlerFn(t)
			b, err := chop.Wrap(h).Invoke(context.Background(), []byte(tt.payload))

			if err != nil && !tt.err {
				t.Errorf("got %v, expected nil", err)
			} else if err == nil && tt.err {
				t.Error("got nil, expected an error")
			}
			if err != nil {
				return
			}

			if err = json.Unmarshal(b, tt.act); err != nil {
				t.Errorf("got %v, expected nil", err)
			}

			if !reflect.DeepEqual(tt.act, tt.exp) {
				t.Errorf("got %v, expected %v", tt.act, tt.exp)
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		data   []string
		header http.Header
		exp    response
	}{
		{
			name: "should set default status code if not called",
			exp: response{
				status:     toStatusDescription(http.StatusOK),
				statusCode: http.StatusOK,
				header:     http.Header{},
			},
		},
		{
			name: "should set default status code and headers",
			data: []string{"body"},
			exp: response{
				status:     toStatusDescription(http.StatusOK),
				statusCode: http.StatusOK,
				body:       "body",
				header: http.Header{
					"Content-Type": {"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "should not overwrite existing status code",
			code: http.StatusCreated,
			data: []string{"body"},
			exp: response{
				status:     toStatusDescription(http.StatusCreated),
				statusCode: http.StatusCreated,
				body:       "body",
				header:     http.Header{},
			},
		},
		{
			name: "should not overwrite existing content type header",
			data: []string{"body"},
			header: http.Header{
				"Content-Type": {"application/json"},
			},
			exp: response{
				status:     toStatusDescription(http.StatusOK),
				statusCode: http.StatusOK,
				body:       "body",
				header: http.Header{
					"Content-Type": {"application/json"},
				},
			},
		},
		{
			name: "should not write content type header if transfer encoding is set",
			data: []string{"body"},
			header: http.Header{
				"Transfer-Encoding": {"gzip"},
			},
			exp: response{
				status:     toStatusDescription(http.StatusOK),
				statusCode: http.StatusOK,
				body:       "body",
				header: http.Header{
					"Transfer-Encoding": {"gzip"},
				},
			},
		},
		{
			name:   "should permit multiple writes",
			data:   []string{"a", "b", "c"},
			header: http.Header{},
			exp: response{
				status:     toStatusDescription(http.StatusOK),
				statusCode: http.StatusOK,
				body:       "abc",
				header: http.Header{
					"Content-Type": {"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "should support multi value headers",
			header: http.Header{
				"X-Custom-Header": {"value1", "value2"},
			},
			exp: response{
				status:     toStatusDescription(http.StatusOK),
				statusCode: http.StatusOK,
				header: http.Header{
					"X-Custom-Header": {"value1", "value2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := chop.NewResponseWriter()

			if tt.code != 0 {
				w.WriteHeader(tt.code)
			}

			if tt.header != nil {
				for k, vs := range tt.header {
					for _, v := range vs {
						w.Header().Add(k, v)
					}
				}
			}

			if tt.data != nil {
				for _, d := range tt.data {
					w.Write([]byte(d))
				}
			}

			act := toResponse(w)

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
		exp   response
	}{
		{
			name:  "should use the specified value",
			codes: []int{http.StatusBadRequest},
			exp: response{
				status:     toStatusDescription(http.StatusBadRequest),
				statusCode: http.StatusBadRequest,
				header:     http.Header{},
			},
		},
		{
			name:  "should use the first value",
			codes: []int{http.StatusBadRequest, http.StatusOK},
			exp: response{
				status:     toStatusDescription(http.StatusBadRequest),
				statusCode: http.StatusBadRequest,
				header:     http.Header{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := chop.NewResponseWriter()

			for _, c := range tt.codes {
				w.WriteHeader(c)
			}

			act := toResponse(w)

			if !reflect.DeepEqual(act, tt.exp) {
				t.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}

type (
	request struct {
		method string
		url    string
		body   string
		header http.Header
	}

	response struct {
		status     string
		statusCode int
		body       string
		header     http.Header
	}
)

func toRequest(r *http.Request) request {
	if r == nil {
		return request{}
	}

	b := bytes.NewBuffer(nil)
	b.ReadFrom(r.Body)

	return request{
		method: r.Method,
		url:    r.URL.String(),
		body:   b.String(),
		header: r.Header,
	}
}

func toResponse(w *chop.ResponseWriter) response {
	if w == nil {
		return response{}
	}

	return response{
		status:     w.Status(),
		statusCode: w.StatusCode(),
		body:       w.Body(),
		header:     w.Header(),
	}
}

func toStatusDescription(code int) string {
	return fmt.Sprintf("%d %s", code, http.StatusText(code))
}

func invokeLocal(port string, payload []byte) ([]byte, error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%s", port))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	req := &messages.InvokeRequest{Payload: payload}
	res := new(messages.InvokeResponse)

	err = client.Call("Function.Invoke", &req, &res)
	if err != nil {
		return nil, err
	}

	if res.Error != nil {
		return nil, errors.New(res.Error.Message)
	}

	return res.Payload, nil
}

const (
	apiGatewayProxyEventPayload = `{
	"resource": "/{proxy+}",
	"path": "/resource/",
	"httpMethod": "GET",
	"headers": {
		"X-Custom-Header1": "v1",
		"X-Custom-Header2": "v3"
	},
	"multiValueHeaders": {
		"X-Custom-Header1": [
			"v1"
		],
		"X-Custom-Header2": [
			"v2",
			"v3"
		]
	},
	"queryStringParameters": {
		"q1": "v1",
		"q2": "v3"
	},
	"multiValueQueryStringParameters": {
		"q1": [
			"v1"
		],
		"q2": [
			"v2",
			"v3"
		]
	},
	"pathParameters": {
		"proxy": "resource"
	},
	"stageVariables": null,
	"requestContext": {
		"resourcePath": "/{proxy+}",
		"httpMethod": "GET",
		"path": "/dev/resource/",
		"protocol": "HTTP/1.1",
		"apiId": "apiid"
	},
	"body": "body",
	"isBase64Encoded": false
}`

	apiGatewayV2HTTPEventPayload = ` {
	"version": "2.0",
	"routeKey": "$default",
	"rawPath": "/resource/",
	"rawQueryString": "q1=v1&q2=v2&q2=v3",
	"headers": {
		"x-custom-header1": "v1",
		"x-custom-header2": "v2"
	},
	"queryStringParameters": {
		"q1": "v1",
		"q2": "v2,v3"
	},
	"requestContext": {
		"apiId": "apiid",
		"http": {
			"method": "GET",
			"path": "/resource",
			"protocol": "HTTP/1.1"
		}
	},
	"body": "body",
	"isBase64Encoded": false
}`

	albTargetGroupSingleValueEventPayload = `{
	"requestContext": {
		"elb": {
			"targetGroupArn": "arn"
		}
	},
	"httpMethod": "GET",
	"path": "/resource/",
	"queryStringParameters": {
		"q1": "v1",
		"q2": "v2"
	},
	"headers": {
		"x-custom-header1": "v1",
		"x-custom-header2": "v2"
	},
	"body": "body",
	"isBase64Encoded": false
}`

	albTargetGroupMultiValueEventPayload = `{
	"requestContext": {
		"elb": {
			"targetGroupArn": "arn"
		}
	},
	"httpMethod": "GET",
	"path": "/resource/",
	"multiValueQueryStringParameters": {
		"q1": [
			"v1"
		],
		"q2": [
			"v2",
			"v3"
		]
	},
	"multiValueHeaders": {
		"x-custom-header1": [
			"v1"
		],
		"x-custom-header2": [
			"v2",
			"v3"
		]
	},
	"body": "body",
	"isBase64Encoded": false
}`
)
