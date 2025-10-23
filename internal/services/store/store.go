package store

import (
	"context"
	"fmt"

	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/reader"
	"github.com/w-h-a/backend/internal/clients/readwriter"
)

type Store struct {
	schemas map[string][]v1alpha1.FieldSchema
	rws     map[string]readwriter.ReadWriter
}

func (s *Store) Start(stop chan struct{}) {
	// TODO
}

func (s *Store) ReadOne(ctx context.Context, resource string, id string) (v1alpha1.Resource, error) {
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

	oldRec, err := rw.ReadOne(ctx, id)
	if err != nil {
		return err
	}

	oldRes, err := v1alpha1.ToResource(schemas, oldRec)
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
