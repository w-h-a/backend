package reader

import (
	"context"

	"github.com/w-h-a/backend/api/v1alpha1"
)

type Reader interface {
	ReadOne(ctx context.Context, id string, opts ...ReadOneOption) (v1alpha1.Record, error)
	List(ctx context.Context, opts ...ListOption) ([]v1alpha1.Record, error)
}
