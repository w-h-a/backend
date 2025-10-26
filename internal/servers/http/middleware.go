package http

import "net/http"

type Middleware func(h http.Handler) http.Handler
