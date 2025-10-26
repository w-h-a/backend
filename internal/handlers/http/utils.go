package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func reqToCtx(r *http.Request) context.Context {
	ctx := r.Context()

	for k, v := range r.Header {
		ctx = context.WithValue(ctx, strings.ToLower(k), v[0])
	}

	return ctx
}

func wrtJSON(w http.ResponseWriter, statusCode int, data any) {
	bs, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(bs)
}
