# Chop
[![Build Status](https://travis-ci.org/stevecallear/chop.svg?branch=master)](https://travis-ci.org/stevecallear/chop)
[![codecov](https://codecov.io/gh/stevecallear/chop/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/chop)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/chop)](https://goreportcard.com/report/github.com/stevecallear/chop)

Chop provides a wrapper to use Go HTTP handlers to handle AWS Lambda API Gateway proxy integration events.

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

## Deploying
Follow the 'Build and deploy' steps in [this](https://aws.amazon.com/blogs/compute/announcing-go-support-for-aws-lambda/) AWS blog post. 

## Request Context
Both the proxy integration event and lambda context are stored in the request context. They can be accessed using the `chop.GetEvent` and `chop.GetContext` functions respectively.

```
func main() {
    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        e, _ := chop.GetEvent(r)
        fmt.Fprintf(w, "Stage: %s", e.RequestContext.Stage)
        
        c, _ := chop.GetContext(r)
        fmt.Fprintf(w, "AwsRequestID: %s", c.AwsRequestID)
    })
    chop.Start(h)
}
```
