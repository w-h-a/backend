package store

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/reader"
	"github.com/w-h-a/backend/internal/clients/readwriter"
)

type Store struct {
	schemas map[string][]v1alpha1.FieldSchema
	rws     map[string]readwriter.ReadWriter
}

func (s *Store) Start(stop chan struct{}) error {
	// TODO

	<-stop

	for _, rw := range s.rws {
		if err := rw.Close(context.Background()); err != nil {
			// log error
		}
	}

	return nil
}

func (s *Store) Authenticate(ctx context.Context, username string, password string) (v1alpha1.Resource, error) {
	return s.authenticate(ctx, username, password)
}

func (s *Store) authenticate(ctx context.Context, username string, password string) (v1alpha1.Resource, error) {
	rs, err := s.list(ctx, "_users", "")
	if err != nil {
		return nil, errors.New("unautheticated")
	}

	// TODO: don't do this loop
	for _, u := range rs {
		if u["_id"] == username {
			salt, ok := u["salt"].(string)
			if !ok {
				return nil, fmt.Errorf("user %q has invalid salt data", username)
			}
			expectedPassword, ok := u["password"].(string)
			if !ok {
				return nil, fmt.Errorf("user %q has invalid password data", username)
			}
			if expectedPassword == HashPassword(password, salt) {
				return u, nil
			}
		}
	}

	return nil, errors.New("unauthenticated")
}

func (s *Store) Authorize(ctx context.Context, resource string, id string, action string, username string, password string) error {
	rs, err := s.list(ctx, "_permissions", "")
	if err != nil {
		return fmt.Errorf("permissions error: %w", err)
	}

	var u v1alpha1.Resource

	for _, p := range rs {
		if p["resource"] != resource || (p["action"] != "*" && p["action"] != action) {
			continue // find what we're looking for
		}

		if p["field"] == "" && p["role"] == "" {
			return nil // public
		}

		if u == nil {
			u, err = s.authenticate(ctx, username, password)
			if err != nil {
				return err
			}
		}

		if role, roleOK := p["role"].(string); roleOK {
			if roles, rolesOK := u["roles"].([]string); rolesOK {
				if role == "*" || slices.Contains(roles, role) {
					return nil // rbac
				}
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

	return errors.New("unauthorized")
}

func (s *Store) ReadOne(ctx context.Context, resource string, id string) (v1alpha1.Resource, error) {
	return s.readOne(ctx, resource, id)
}

func (s *Store) readOne(ctx context.Context, resource string, id string) (v1alpha1.Resource, error) {
	schemas, ok := s.schemas[resource]
	if !ok {
		return nil, fmt.Errorf("no schema found for resource %s", resource)
	}

	rw, ok := s.rws[resource]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resource)
	}

	rec, err := rw.ReadOne(ctx, id)
	if err != nil {
		return nil, err
	}

	return v1alpha1.ToResource(schemas, rec)
}

func (s *Store) List(ctx context.Context, resource string, sortBy string) ([]v1alpha1.Resource, error) {
	return s.list(ctx, resource, sortBy)
}

func (s *Store) list(ctx context.Context, resource string, sortBy string) ([]v1alpha1.Resource, error) {
	schemas, ok := s.schemas[resource]
	if !ok {
		return nil, fmt.Errorf("no schema found for resource %s", resource)
	}

	rw, ok := s.rws[resource]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resource)
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

func (s *Store) Create(ctx context.Context, resource string, newRes v1alpha1.Resource) (string, error) {
	schemas, ok := s.schemas[resource]
	if !ok {
		return "", fmt.Errorf("no schema found for resource %s", resource)
	}

	rw, ok := s.rws[resource]
	if !ok {
		return "", fmt.Errorf("resource %s not found", resource)
	}

	newId := GenerateId()

	newRes["_id"] = newId
	newRes["_v"] = 1.0

	rec, err := v1alpha1.ToRecord(schemas, newRes)
	if err != nil {
		return "", err
	}

	return newId, rw.Create(ctx, rec)
}

func (s *Store) Update(ctx context.Context, resource string, updatedRes v1alpha1.Resource) error {
	schemas, ok := s.schemas[resource]
	if !ok {
		return fmt.Errorf("no schema found for resource %s", resource)
	}

	rw, ok := s.rws[resource]
	if !ok {
		return fmt.Errorf("resource %s not found", resource)
	}

	id, ok := updatedRes["_id"].(string)
	if !ok {
		return fmt.Errorf("'_id' field is missing or not a string in update payload")
	}

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

func (s *Store) Delete(ctx context.Context, resource string, id string) error {
	rw, ok := s.rws[resource]
	if !ok {
		return fmt.Errorf("resource %s not found", resource)
	}

	return rw.Delete(ctx, id)
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
	}
}
