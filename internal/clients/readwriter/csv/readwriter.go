package csv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/reader"
	"github.com/w-h-a/backend/internal/clients/readwriter"
	"github.com/w-h-a/backend/internal/clients/writer"
)

type csvReadWriter struct {
	options readwriter.Options
	f       *os.File
	w       *csv.Writer
	index   map[string]int64
	version map[string]int64
	mtx     sync.RWMutex
}

func (rw *csvReadWriter) ReadOne(ctx context.Context, id string, opts ...reader.ReadOneOption) (v1alpha1.Record, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()

	if rw.version[id] < 1 {
		return nil, errors.New("record not found")
	}

	offset, ok := rw.index[id]
	if !ok {
		return nil, nil
	}

	if _, err := rw.f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	r := csv.NewReader(rw.f)

	rec, err := r.Read()
	if err != nil {
		return nil, err
	}

	if len(rec) > 0 && rec[0] != id {
		slog.ErrorContext(ctx, "corrupted index", "record", rec)
		return nil, errors.New("corrupted index")
	}

	return rec, nil
}

func (rw *csvReadWriter) List(ctx context.Context, opts ...reader.ListOption) ([]v1alpha1.Record, error) {
	options := reader.NewListOptions(opts...)

	rs := []v1alpha1.Record{}
	var listErr error

	generator := rw.iter(ctx)

	generator(func(r v1alpha1.Record, err error) bool {
		if err != nil {
			listErr = err
			return false
		}

		rs = append(rs, r)

		return true
	})

	if listErr != nil {
		return nil, listErr
	}

	if len(options.SortBy) == 0 {
		return rs, nil
	}

	sortDef, ok := rw.options.Schema[options.SortBy]
	if !ok {
		return nil, fmt.Errorf("field '%s' is not a defined schema field for sorting", options.SortBy)
	}

	sortIndex := sortDef.Index
	sortType := sortDef.Type

	sort.Slice(rs, func(i, j int) bool {
		if sortIndex >= len(rs[i]) || sortIndex >= len(rs[j]) {
			return false
		}

		a := rs[i][sortIndex]
		b := rs[j][sortIndex]

		if a == "" && b != "" {
			return false
		}
		if a != "" && b == "" {
			return true
		}

		switch sortType {
		case "number":
			aFloat, _ := strconv.ParseFloat(a, 64)
			bFloat, _ := strconv.ParseFloat(b, 64)

			return aFloat < bFloat
		case "text":
			return a < b
		default:
			return false
		}
	})

	return rs, nil
}

func (rw *csvReadWriter) Create(ctx context.Context, r v1alpha1.Record, opts ...writer.WriteOption) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	if len(r) == 0 || len(r[0]) == 0 {
		return errors.New("invalid record")
	}

	r[1] = "1"

	return rw.append(ctx, r)
}

func (rw *csvReadWriter) Update(ctx context.Context, r v1alpha1.Record, opts ...writer.UpdateOption) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	if len(r) == 0 {
		return errors.New("empty record")
	}

	r[1] = strconv.FormatInt(rw.version[r[0]]+1, 10)

	return rw.append(ctx, r)
}

func (rw *csvReadWriter) Delete(ctx context.Context, id string, opts ...writer.DeleteOption) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	if rw.version[id] < 1 {
		return errors.New("record not found")
	}

	return rw.append(ctx, v1alpha1.Record{id, "0"})
}

func (rw *csvReadWriter) Close(ctx context.Context) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	rw.w.Flush()

	return rw.f.Close()
}

func (rw *csvReadWriter) iter(_ context.Context) func(yield func(v1alpha1.Record, error) bool) {
	return func(yield func(v1alpha1.Record, error) bool) {
		rw.mtx.RLock()
		defer rw.mtx.RUnlock()

		if _, err := rw.f.Seek(0, io.SeekStart); err != nil {
			yield(nil, err)
			return
		}

		r := csv.NewReader(rw.f)

		r.FieldsPerRecord = -1

		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				yield(nil, err)
				return
			}
			if len(rec) < 2 {
				continue
			}
			id, version := rec[0], rec[1]
			if version == "0" || version != strconv.FormatInt(rw.version[id], 10) {
				continue // deleted or outdated
			}
			if !yield(rec, nil) {
				return
			}
		}
	}
}

func (rw *csvReadWriter) append(_ context.Context, r v1alpha1.Record) error {
	var err error

	pos, _ := rw.f.Seek(0, io.SeekEnd)

	err = rw.w.Write(r)
	if err != nil {
		return err
	}

	rw.w.Flush()

	rw.index[r[0]] = pos
	rw.version[r[0]], err = strconv.ParseInt(r[1], 10, 64)
	if err != nil {
		return err
	}

	return nil
}

func NewReadWriter(opts ...readwriter.Option) readwriter.ReadWriter {
	options := readwriter.NewOptions(opts...)

	rw := &csvReadWriter{
		options: options,
		index:   map[string]int64{},
		version: map[string]int64{},
	}

	f, err := os.OpenFile(options.Location, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}

	rw.f = f
	rw.w = csv.NewWriter(f)

	r := csv.NewReader(f)

	for {
		pos := r.InputOffset()
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}
		if len(rec) > 1 {
			rw.index[rec[0]] = pos
			rw.version[rec[0]], _ = strconv.ParseInt(rec[1], 10, 64)
		}
	}

	return rw
}
