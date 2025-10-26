package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/handlers"
	"github.com/w-h-a/backend/internal/services/store"
)

type handler struct {
	schemas map[string][]v1alpha1.FieldSchema
	store   *store.Store
}

func (h *handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	ctx := reqToCtx(r)

	vars := mux.Vars(r)
	resourceName := vars["resource"]
	sortBy := r.URL.Query().Get("sort_by")

	user, _ := handlers.GetUserFromCtx(ctx)
	if err := h.store.Authorize(ctx, resourceName, "", "read", user); err != nil {
		if errors.Is(err, store.ErrAuthn) {
			http.Error(w, fmt.Sprintf("Unauthenticated: %v", err), http.StatusUnauthorized)
			return
		} else if errors.Is(err, store.ErrAuthz) {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusForbidden)
			return
		} else if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read resources: %v", err), http.StatusInternalServerError)
		return
	}

	resources, err := h.store.List(ctx, resourceName, sortBy)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to list resources: %v", err), http.StatusInternalServerError)
		return
	}

	if resources == nil {
		resources = []v1alpha1.Resource{}
	}

	wrtJSON(w, http.StatusOK, resources)
}

func (h *handler) GetRecord(w http.ResponseWriter, r *http.Request) {
	ctx := reqToCtx(r)

	vars := mux.Vars(r)
	resourceName := vars["resource"]
	recordId := vars["id"]

	user, _ := handlers.GetUserFromCtx(ctx)
	if err := h.store.Authorize(ctx, resourceName, recordId, "read", user); err != nil {
		if errors.Is(err, store.ErrAuthn) {
			http.Error(w, fmt.Sprintf("Unauthenticated: %v", err), http.StatusUnauthorized)
			return
		} else if errors.Is(err, store.ErrAuthz) {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusForbidden)
			return
		} else if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read resource: %v", err), http.StatusInternalServerError)
		return
	}

	resource, err := h.store.ReadOne(ctx, resourceName, recordId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read resource: %v", err), http.StatusInternalServerError)
		return
	}

	wrtJSON(w, http.StatusOK, resource)
}

func (h *handler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	ctx := reqToCtx(r)

	vars := mux.Vars(r)
	resourceName := vars["resource"]

	user, _ := handlers.GetUserFromCtx(ctx)
	if err := h.store.Authorize(ctx, resourceName, "", "create", user); err != nil {
		if errors.Is(err, store.ErrAuthn) {
			http.Error(w, fmt.Sprintf("Unauthenticated: %v", err), http.StatusUnauthorized)
			return
		} else if errors.Is(err, store.ErrAuthz) {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusForbidden)
			return
		} else if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to create resource: %v", err), http.StatusInternalServerError)
		return
	}

	var rawInput v1alpha1.Resource
	if err := json.NewDecoder(r.Body).Decode(&rawInput); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	if rawInput == nil {
		rawInput = v1alpha1.Resource{}
	}

	resourceSchema, ok := h.schemas[resourceName]
	if !ok {
		http.Error(w, fmt.Sprintf("No schema found for resource %s", resourceName), http.StatusBadRequest)
		return
	}

	newRes, err := v1alpha1.ParseResource(resourceSchema, rawInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	delete(newRes, "_id")
	delete(newRes, "_v")

	newId, err := h.store.Create(ctx, resourceName, newRes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create resource: %v", err), http.StatusInternalServerError)
		return
	}

	wrtJSON(w, http.StatusCreated, map[string]string{"_id": newId})
}

func (h *handler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	ctx := reqToCtx(r)

	vars := mux.Vars(r)
	resourceName := vars["resource"]
	recordId := vars["id"]

	user, _ := handlers.GetUserFromCtx(ctx)
	if err := h.store.Authorize(ctx, resourceName, recordId, "update", user); err != nil {
		if errors.Is(err, store.ErrAuthn) {
			http.Error(w, fmt.Sprintf("Unauthenticated: %v", err), http.StatusUnauthorized)
			return
		} else if errors.Is(err, store.ErrAuthz) {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusForbidden)
			return
		} else if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update resource: %v", err), http.StatusInternalServerError)
		return
	}

	var rawInput v1alpha1.Resource
	if err := json.NewDecoder(r.Body).Decode(&rawInput); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	if rawInput == nil {
		rawInput = v1alpha1.Resource{}
	}

	resourceSchema, ok := h.schemas[resourceName]
	if !ok {
		http.Error(w, fmt.Sprintf("No schema found for resource %s", resourceName), http.StatusBadRequest)
		return
	}

	updatedRes, err := v1alpha1.ParseResource(resourceSchema, rawInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	delete(updatedRes, "_v")
	updatedRes["_id"] = recordId

	if err := h.store.Update(ctx, resourceName, updatedRes); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update resource: %v", err), http.StatusInternalServerError)
		return
	}

	wrtJSON(w, 200, updatedRes)
}

func (h *handler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	ctx := reqToCtx(r)

	vars := mux.Vars(r)
	resourceName := vars["resource"]
	recordId := vars["id"]

	user, _ := handlers.GetUserFromCtx(ctx)
	if err := h.store.Authorize(ctx, resourceName, recordId, "delete", user); err != nil {
		if errors.Is(err, store.ErrAuthn) {
			http.Error(w, fmt.Sprintf("Unauthenticated: %v", err), http.StatusUnauthorized)
			return
		} else if errors.Is(err, store.ErrAuthz) {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusForbidden)
			return
		} else if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete resource: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.store.Delete(ctx, resourceName, recordId); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Resource: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func NewHandler(schemas map[string][]v1alpha1.FieldSchema, store *store.Store) *handler {
	return &handler{
		schemas: schemas,
		store:   store,
	}
}
