package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/urfave/cli/v2"
	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/readwriter"
	"github.com/w-h-a/backend/internal/clients/readwriter/csv"
	httphandlers "github.com/w-h-a/backend/internal/handlers/http"
	"github.com/w-h-a/backend/internal/servers"
	httpserver "github.com/w-h-a/backend/internal/servers/http"
	"github.com/w-h-a/backend/internal/services/store"
)

func Run(ctx *cli.Context) error {
	// config

	// resource

	// logs

	// traces

	// wait group & stop channels
	var wg sync.WaitGroup
	stopChannels := map[string]chan struct{}{}

	// setup
	schemas, rws, err := initReadWriters()
	if err != nil {
		return err
	}

	s := store.New(schemas, rws)
	stopChannels["store"] = make(chan struct{})

	httpSrv, err := initHttpServer(schemas, s)
	if err != nil {
		return err
	}
	stopChannels["httpserver"] = make(chan struct{})

	// error and sig chans
	errCh := make(chan error, len(stopChannels))
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// start
	wg.Add(1)
	go func() {
		defer wg.Done()
		// log
		errCh <- s.Run(stopChannels["store"])
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// log
		errCh <- httpSrv.Run(stopChannels["httpserver"])
	}()

	// block
	select {
	case err := <-errCh:
		if err != nil {
			// log that we failed
			return err
		}
	case <-sigChan:
		for _, stop := range stopChannels {
			close(stop)
		}
	}

	wg.Wait()

	close(errCh)

	for err := range errCh {
		if err != nil {
			// log
		}
	}

	return nil
}

func initReadWriters() (map[string][]v1alpha1.FieldSchema, map[string]readwriter.ReadWriter, error) {
	schemas := map[string][]v1alpha1.FieldSchema{}
	resourceData := map[string][]struct {
		FieldSchema v1alpha1.FieldSchema
		Index       int
	}{}

	dir := "examples/todo"

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

func initHttpServer(schemas map[string][]v1alpha1.FieldSchema, s *store.Store) (servers.Server, error) {
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
		return nil, fmt.Errorf("failed to attach root handler: %w", err)
	}

	return srv, nil
}
