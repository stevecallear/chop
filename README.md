# Chop
[![Build Status](https://github.com/stevecallear/chop/actions/workflows/build.yml/badge.svg)](https://github.com/stevecallear/chop/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/stevecallear/chop/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/chop)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/chop)](https://goreportcard.com/report/github.com/stevecallear/chop)

Chop provides a wrapper to use Go HTTP handlers to handle AWS Lambda events.

This repository started life before native Go Lambda support. Since then AWS have built their own wrapper for Lambda events, available [here](https://github.com/awslabs/aws-lambda-go-api-proxy). Any code going to production should use the officially supported proxy. The primary purpose of this repository is to satisfy my personal interest in API Gateway developments.

## Getting started
```
go get github.com/stevecallear/chop/v2
```
```
import (
    "fmt"
    "net/http"

    "github.com/stevecallear/chop"
)

func main() {
    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "%s %s", r.Method, r.URL.String())
    })

    chop.Start(h)
}
```

## Request Context
Both the Lambda request event and Lambda context are available on the request.

### Request Event
Chop will resolve the request event at runtime so the type cannot be guaranteed. The following example demonstrates how to retrieve the event from the request.

```
h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    switch e := chop.GetEvent(r).(type) {
    case *events.APIGatewayProxyRequest:
        // handle the API gateway proxy integration event
    case *events.APIGatewayV2HTTPRequest:
        // handle the API gateway http v2 event
    case *events.ALBTargetGroupRequest:
        // handle the ALB target group event
    default:
        panic("invalid event")
    }
})
```

> Note: the panic in the example above is unreachable code. Chop will return `ErrUnsupportedEventType` if the event type cannot be successfully parsed.

### Lambda Context
The following example demonstrates how to retrieve the Lambda context from the request.

```
h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ctx, ok := lambacontext.FromContext(r.Context())
    // handle the context
})
```
