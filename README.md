# Chop
[![Build Status](https://travis-ci.org/stevecallear/chop.svg?branch=master)](https://travis-ci.org/stevecallear/chop)
[![codecov](https://codecov.io/gh/stevecallear/chop/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/chop)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/chop)](https://goreportcard.com/report/github.com/stevecallear/chop)

Chop provides a wrapper to use Go HTTP handlers over AWS Lambda with API Gateway proxy integration. It has been built to work with [eawsy Lambda Go shim](https://github.com/eawsy/aws-lambda-go-shim), but the contract may be tweaked if and when native Go Lambdas become available.

## Getting started
```
go get github.com/stevecallear/chop
```

```
import (
    "fmt"
    "net/http"

    "github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
    "github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/apigatewayproxyevt"
    "github.com/stevecallear/chop"
)

func Handle(evt *apigatewayproxyevt.Event, ctx *runtime.Context) (interface{}, error) {
    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "%s %s", r.Method, r.URL.String())
    })
    return chop.Wrap(h).Handle(evt, ctx)
}
```

## Deploying
* Follow the [eawsy Lambda Go shim](https://github.com/eawsy/aws-lambda-go-shim) instructions to build a handler, create a `.zip` package and deploy it to AWS Lambda
* Create a new API Gateway API and add a 'proxy integration' resource
* Link the resource to the Lambda and test

## Request Context
The proxy integration event and runtime context are stored in the request context. They can be accessed by `chop.GetEvent` and `chop.GetContext` respectively.

```
func Handle(evt *apigatewayproxyevt.Event, ctx *runtime.Context) (interface{}, error) {
    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        c := chop.GetContext(r)
        fmt.Fprintf(w, "%s %s", c.FunctionName, c.FunctionVersion)
    })
    return chop.Wrap(h).Handle(evt, ctx)
}
```