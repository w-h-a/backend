package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/reader"
	"github.com/w-h-a/backend/internal/clients/readwriter"
	"github.com/w-h-a/backend/internal/clients/writer"
)

type Store struct {
	schemas   map[string][]v1alpha1.FieldSchema
	rws       map[string]readwriter.ReadWriter
	isRunning bool
	mtx       sync.RWMutex
}

func (s *Store) Run(stop chan struct{}) error {
	s.mtx.RLock()
	if s.isRunning {
		s.mtx.RUnlock()
		return errors.New("store already running")
	}
	s.mtx.RUnlock()

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start store: %w", err)
	}

	<-stop

	return s.Stop()
}

func (s *Store) Start() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.isRunning {
		return errors.New("store already started")
	}

	s.isRunning = true

	return nil
}

func (s *Store) Stop() error {
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	return s.stop(stopCtx)
}

func (s *Store) stop(ctx context.Context) error {
	s.mtx.Lock()

	if !s.isRunning {
		s.mtx.Unlock()
		return errors.New("store not running")
	}

	s.isRunning = false

	s.mtx.Unlock()

	gracefulStopDone := make(chan struct{})
	go func() {
		for _, rw := range s.rws {
			if err := rw.Close(context.Background()); err != nil {
				// log error
			}
		}
		close(gracefulStopDone)
	}()

	var stopErr error

	select {
	case <-gracefulStopDone:
	case <-ctx.Done():
		stopErr = ctx.Err()
	}

	return stopErr
}

func (s *Store) Authenticate(ctx context.Context, username string, password string) (v1alpha1.Resource, error) {
	// ctx, span := s.tracer.Start(ctx, "store.Authenticate", trace.WithAttributes(attribute.String("user.id", username)))
	// defer span.End()

	u, err := s.readOne(ctx, "_users", username)
	if err != nil {
		// span.RecordError(err)
		// slog.WarnContext(ctx, "Authentication failed: user not found", "user.id", username, "error", err)
		return nil, ErrAuthn
	}

	salt, ok := u["salt"].(string)
	if !ok {
		// err := fmt.Errorf("user %q has invalid salt data", username)
		// span.RecordError(err)
		// slog.ErrorContext(ctx, "Authentication failed: invalid password data", "user.id", username)
		return nil, ErrAuthn
	}

	expectedPassword, ok := u["password"].(string)
	if !ok {
		// err = fmt.Errorf("user %q has invalid password data", username)
		// span.RecordError(err)
		// slog.ErrorContext(ctx, "Authentication failed: invalid password data", "user.id", username)
		return nil, ErrAuthn
	}

	if expectedPassword != HashPassword(password, salt) {
		// err := errors.New("password mismatch")
		// span.RecordError(err)
		// slog.WarnContext(ctx, "Authentication failed: password mismatch", "user.id", username)
		return nil, ErrAuthn
	}

	return u, nil
}

func (s *Store) Authorize(ctx context.Context, resource string, id string, action string, u v1alpha1.Resource) error {
	username := ""
	if u != nil {
		username = u["_id"].(string)
	}

	roles := []string{}
	if u != nil {
		if rs, rolesOk := u["roles"].([]string); rolesOk {
			roles = rs
		}
	}

	// ctx, span := s.tracer.Start(ctx, "store.Authorize", trace.WithAttributes(
	// 	attribute.String("resource.name", resource),
	// 	attribute.String("record.id", id),
	// 	attribute.String("action", action),
	// 	attribute.String("user.id", username), // Include username if available
	// ))
	// defer span.End()

	rs, err := s.list(ctx, "_permissions", "")
	if err != nil {
		// span.Record(err)
		// slog.ErrorContext(ctx, "Authorization failed: could not load permissions", "error", err)
		return ErrAuthz
	}

	for _, p := range rs {
		if p["resource"] != resource || (p["action"] != "*" && p["action"] != action) {
			continue // find what we're looking for
		}

		if p["field"] == "" && p["role"] == "" {
			return nil // public
		}

		if u == nil {
			return ErrAuthn
		}

		if role, roleOK := p["role"].(string); roleOK {
			if role == "*" || slices.Contains(roles, role) {
				return nil // rbac
			}
		}

		if len(id) > 0 {
			res, err := s.readOne(ctx, resource, id)
			if err != nil {
				return err
			}
			if field, ok := p["field"].(string); ok {
				if user, ok := res[field]; ok && user == username {
					return nil // user name matches requested resource field
				} else if users, ok := res[field].([]string); ok && slices.Contains(users, username) {
					return nil // user name is in the requested resource field
				}
			}
		}
	}

	return ErrAuthz
}

// TODO: traces
func (s *Store) List(ctx context.Context, resource string, sortBy string) ([]v1alpha1.Resource, error) {
	return s.list(ctx, resource, sortBy)
}

// TODO: traces
func (s *Store) list(ctx context.Context, resource string, sortBy string) ([]v1alpha1.Resource, error) {
	schemas, ok := s.schemas[resource]
	if !ok {
		return nil, ErrNotFound
	}

	rw, ok := s.rws[resource]
	if !ok {
		return nil, ErrNotFound
	}

	rs := []v1alpha1.Resource{}

	recs, err := rw.List(ctx, reader.WithSortBy(sortBy))
	if err != nil {
		return nil, err
	}

	for _, rec := range recs {
		r, err := v1alpha1.ToResource(schemas, rec)
		if err != nil {
			return rs, err
		}
		rs = append(rs, r)
	}

	return rs, nil
}

func (s *Store) ReadOne(ctx context.Context, resource string, id string) (v1alpha1.Resource, error) {
	// ctx, span := s.tracer.Start(ctx, "store.ReadOne", trace.WithAttributes(
	// 	attribute.String("resource.name", resource),
	// 	attribute.String("record.id", id),
	// ))
	// defer span.End()

	return s.readOne(ctx, resource, id)
}

func (s *Store) readOne(ctx context.Context, resource string, id string) (v1alpha1.Resource, error) {
	// ctx, span := s.tracer.Start(ctx, "store.ReadOne", trace.WithAttributes(
	// 	attribute.String("resource.name", resource),
	// 	attribute.String("record.id", id),
	// ))
	// defer span.End()

	schemas, ok := s.schemas[resource]
	if !ok {
		// err := errors.New("schema not found")
		// span.RecordError(err)
		return nil, ErrNotFound
	}

	rw, ok := s.rws[resource]
	if !ok {
		// ditto
		return nil, ErrNotFound
	}

	rec, err := rw.ReadOne(ctx, id)
	if err != nil {
		if errors.Is(err, reader.ErrNotFound) {
			return nil, ErrNotFound
		}
		// err := fmt.Errorf("persistence layer error: %w", err)
		// span.RecordError(err)
		return nil, err
	}

	rs, err := v1alpha1.ToResource(schemas, rec)
	if err != nil {
		// span.RecordError(err)
		return nil, err
	}

	return rs, nil
}

// TODO: traces
func (s *Store) Create(ctx context.Context, resource string, newRes v1alpha1.Resource) (string, error) {
	schemas := s.schemas[resource]
	rw := s.rws[resource]

	newId := GenerateId()

	newRes["_id"] = newId
	newRes["_v"] = 1.0

	rec, err := v1alpha1.ToRecord(schemas, newRes)
	if err != nil {
		return "", err
	}

	return newId, rw.Create(ctx, rec)
}

// TODO: traces
func (s *Store) Update(ctx context.Context, resource string, updatedRes v1alpha1.Resource) error {
	schemas := s.schemas[resource]
	rw := s.rws[resource]

	id := updatedRes["_id"].(string)

	oldRes, err := s.readOne(ctx, resource, id)
	if err != nil {
		return err
	}

	for _, fs := range schemas {
		if _, ok := updatedRes[fs.Field]; !ok {
			updatedRes[fs.Field] = oldRes[fs.Field]
		}
	}

	v, ok := oldRes["_v"].(float64)
	if !ok {
		return fmt.Errorf("'_v' field is missing or not a number in old resource with id %s", id)
	}

	updatedRes["_v"] = v + 1

	updatedRec, err := v1alpha1.ToRecord(schemas, updatedRes)
	if err != nil {
		return err
	}

	return rw.Update(ctx, updatedRec)
}

// TODO: traces
func (s *Store) Delete(ctx context.Context, resource string, id string) error {
	rw, ok := s.rws[resource]
	if !ok {
		return ErrNotFound
	}

	if err := rw.Delete(ctx, id); err != nil {
		if errors.Is(err, writer.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (s *Store) CheckHealth(ctx context.Context) error {
	// TODO
	return nil
}

func New(
	schemas map[string][]v1alpha1.FieldSchema,
	rws map[string]readwriter.ReadWriter,
) *Store {
	return &Store{
		schemas: schemas,
		rws:     rws,
		mtx:     sync.RWMutex{},
	}
}
