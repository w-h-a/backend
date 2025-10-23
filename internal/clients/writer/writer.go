package writer

import (
	"context"

	"github.com/w-h-a/backend/api/v1alpha1"
)

type Writer interface {
	Create(ctx context.Context, r v1alpha1.Record, opts ...WriteOption) error
	Update(ctx context.Context, r v1alpha1.Record, opts ...UpdateOption) error
	Delete(ctx context.Context, id string, opts ...DeleteOption) error
	Close(ctx context.Context) error
}
