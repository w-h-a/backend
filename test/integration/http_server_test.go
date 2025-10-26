package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/services/store"
)

func TestHTTPServerWithCSVRW(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Log("SKIPPING INTEGRATION TEST")
		return
	}

	schemas, rws, err := initReadWriters(t, "../testdata/rest")
	require.NoError(t, err)

	s := store.New(schemas, rws)
	err = s.Start()
	require.NoError(t, err)

	defer s.Stop()

	srv, err := initHttpServer(t, schemas, s)
	require.NoError(t, err)

	err = srv.Start()
	require.NoError(t, err)

	defer srv.Stop()

	tests := []struct {
		name     string
		method   string
		path     string
		body     any
		auth     [2]string // username, password
		status   int
		validate func(*testing.T, *http.Response)
	}{
		{
			name:   "List books unauthorized",
			method: "GET",
			path:   "/api/books",
			status: http.StatusOK,
			validate: func(t *testing.T, r *http.Response) {
				var books []v1alpha1.Resource
				json.NewDecoder(r.Body).Decode(&books)
				require.Equal(t, 2, len(books))
			},
		},
		{
			name:   "Create book unauthenticated",
			method: "POST",
			path:   "/api/books",
			body:   v1alpha1.Resource{"title": "New Book"},
			status: http.StatusUnauthorized,
		},
		{
			name:   "Create book invalid creds",
			method: "POST",
			path:   "/api/books",
			body:   v1alpha1.Resource{"title": "New Book"},
			auth:   [2]string{"user1", "wrongpass"},
			status: http.StatusUnauthorized,
		},
		{
			name:   "Create book valid creds",
			method: "POST",
			path:   "/api/books",
			body:   v1alpha1.Resource{"title": "Valid Book", "author": "Unknown Author", "year": 2023},
			auth:   [2]string{"user1", "user1pass"},
			status: http.StatusCreated,
			validate: func(t *testing.T, r *http.Response) {
				var id map[string]string
				json.NewDecoder(r.Body).Decode(&id)
				require.True(t, len(id["_id"]) > 0)
			},
		},
		{
			name:   "Update book unauthorized",
			method: "PUT",
			path:   "/api/books/book1",
			body:   v1alpha1.Resource{"title": "Updated Title"},
			auth:   [2]string{"user1", "user1pass"},
			status: http.StatusForbidden,
		},
		{
			name:   "Delete book as admin",
			method: "DELETE",
			path:   "/api/books/book2",
			auth:   [2]string{"admin", "admin123"},
			status: http.StatusNoContent,
		},
		{
			name:   "Create invalid book",
			method: "POST",
			path:   "/api/books",
			body:   v1alpha1.Resource{"title": "Book 123", "year": 3000},
			auth:   [2]string{"user1", "user1pass"},
			status: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body io.Reader
			if test.body != nil {
				bs, _ := json.Marshal(test.body)
				body = bytes.NewReader(bs)
			}

			req, _ := http.NewRequest(test.method, "http://localhost:4000"+test.path, body)

			if len(test.auth[0]) > 0 {
				req.SetBasicAuth(test.auth[0], test.auth[1])
			}

			rsp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer rsp.Body.Close()

			if test.validate != nil {
				test.validate(t, rsp)
			}

			require.Equal(t, test.status, rsp.StatusCode)
		})
	}
}
