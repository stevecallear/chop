# Chop
[![Build Status](https://travis-ci.org/stevecallear/chop.svg?branch=master)](https://travis-ci.org/stevecallear/chop)
[![codecov](https://codecov.io/gh/stevecallear/chop/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/chop)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/chop)](https://goreportcard.com/report/github.com/stevecallear/chop)

Chop provides a wrapper to use Go HTTP handlers to handle AWS Lambda events.

## Getting started
```
go get github.com/stevecallear/chop
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
    e := chop.GetEvent(r)
    switch e.(type) {
    case events.APIGatewayProxyRequest:
        // handle the API gateway proxy integration event
    case events.ALBTargetGroupRequest:
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