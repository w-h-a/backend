package handlers

import (
	"context"

	"github.com/w-h-a/backend/api/v1alpha1"
)

type UserKey struct{}

func GetUserFromCtx(ctx context.Context) (v1alpha1.Resource, bool) {
	user, ok := ctx.Value(UserKey{}).(v1alpha1.Resource)
	return user, ok
}
