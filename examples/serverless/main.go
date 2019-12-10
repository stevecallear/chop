package main

import (
	"fmt"
	"net/http"

	"github.com/stevecallear/chop/v2"
)

func main() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s %s", r.Method, r.URL.String())
	})

	chop.Start(h)
}
