package integration

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/readwriter"
	"github.com/w-h-a/backend/internal/clients/readwriter/csv"
	httphandlers "github.com/w-h-a/backend/internal/handlers/http"
	"github.com/w-h-a/backend/internal/servers"
	httpserver "github.com/w-h-a/backend/internal/servers/http"
	"github.com/w-h-a/backend/internal/services/store"
)

func initHttpServer(t *testing.T, schemas map[string][]v1alpha1.FieldSchema, s *store.Store) (servers.Server, error) {
	t.Helper()

	srv := httpserver.NewServer(
		servers.WithAddress(":4000"),
		httpserver.WithMiddleware(
			httphandlers.NewAuthMiddleware(s),
		),
	)

	router := mux.NewRouter()

	handler := httphandlers.NewHandler(schemas, s)

	router.HandleFunc("/api/{resource}", handler.ListRecords).Methods(http.MethodGet)
	router.HandleFunc("/api/{resource}/{id}", handler.GetRecord).Methods(http.MethodGet)
	router.HandleFunc("/api/{resource}", handler.CreateRecord).Methods(http.MethodPost)
	router.HandleFunc("/api/{resource}/{id}", handler.UpdateRecord).Methods(http.MethodPut)
	router.HandleFunc("/api/{resource}/{id}", handler.DeleteRecord).Methods(http.MethodDelete)

	if err := srv.Handle(router); err != nil {
		return nil, err
	}

	return srv, nil
}

func initReadWriters(t *testing.T, dir string) (map[string][]v1alpha1.FieldSchema, map[string]readwriter.ReadWriter, error) {
	t.Helper()

	schemas := map[string][]v1alpha1.FieldSchema{}
	resourceData := map[string][]struct {
		FieldSchema v1alpha1.FieldSchema
		Index       int
	}{}

	dir = testData(t, dir)

	schemaRW := csv.NewReadWriter(
		readwriter.WithLocation(dir + "/_schemas.csv"),
	)

	recs, err := schemaRW.List(context.Background())
	if err != nil {
		return nil, nil, err
	}

	for _, rec := range recs {
		schema := v1alpha1.FieldSchema{
			Resource: rec[2],
			Field:    rec[3],
			Type:     rec[4],
			Regex:    rec[7],
		}

		schema.Min, _ = strconv.ParseFloat(rec[5], 64)
		schema.Max, _ = strconv.ParseFloat(rec[6], 64)

		schemas[schema.Resource] = append(schemas[schema.Resource], schema)

		index := len(resourceData[schema.Resource])

		resourceData[schema.Resource] = append(resourceData[schema.Resource], struct {
			FieldSchema v1alpha1.FieldSchema
			Index       int
		}{
			FieldSchema: schema,
			Index:       index,
		})
	}

	rws := map[string]readwriter.ReadWriter{}

	for name, dataList := range resourceData {
		schema := map[string]struct {
			Index int
			Type  string
		}{}

		for _, data := range dataList {
			schema[data.FieldSchema.Field] = struct {
				Index int
				Type  string
			}{
				Index: data.Index,
				Type:  data.FieldSchema.Type,
			}
		}

		if _, ok := rws[name]; !ok {
			rw := csv.NewReadWriter(
				readwriter.WithLocation(dir+"/"+name+".csv"),
				readwriter.WithSchema(schema),
			)
			rws[name] = rw
		}
	}

	return schemas, rws, nil
}

func testData(t *testing.T, src string) string {
	t.Helper()

	dst := t.TempDir()

	if err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(src, path)

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatalf("failed to copy testdata: %v", err)
	}

	return dst
}
