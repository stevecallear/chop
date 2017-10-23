package main

import (
	"fmt"
	"net/http"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/apigatewayproxyevt"
	"github.com/stevecallear/chop"
)

// Handle is the lambda entrypoint
func Handle(evt *apigatewayproxyevt.Event, ctx *runtime.Context) (interface{}, error) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s %s", r.Method, r.URL.String())
	})
	return chop.Wrap(h).Handle(evt, ctx)
}

func main() {
}
