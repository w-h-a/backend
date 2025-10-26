package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/handlers"
	httpserver "github.com/w-h-a/backend/internal/servers/http"
	"github.com/w-h-a/backend/internal/services/store"
)

type authMiddleware struct {
	handler http.Handler
	store   *store.Store
}

func (m *authMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := reqToCtx(r)

	var authenticatedUser v1alpha1.Resource
	var authErr error

	username, password, ok := r.BasicAuth()
	if ok {
		user, err := m.store.Authenticate(ctx, username, password)
		if err == nil {
			authenticatedUser = user
		} else {
			authErr = err
		}
	}

	if authErr != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, fmt.Sprintf("Unauthorized: %v", authErr), http.StatusUnauthorized)
		return
	}

	ctxWithUser := context.WithValue(ctx, handlers.UserKey{}, authenticatedUser)
	rWithUser := r.WithContext(ctxWithUser)

	m.handler.ServeHTTP(w, rWithUser)
}

func NewAuthMiddleware(store *store.Store) httpserver.Middleware {
	return func(handler http.Handler) http.Handler {
		return &authMiddleware{
			handler: handler,
			store:   store,
		}
	}
}
