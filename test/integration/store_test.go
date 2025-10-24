package integration

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/backend/api/v1alpha1"
	"github.com/w-h-a/backend/internal/clients/readwriter"
	"github.com/w-h-a/backend/internal/clients/readwriter/csv"
	"github.com/w-h-a/backend/internal/services/store"
)

func TestStoreAuthorizationWithCSVRW(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Log("SKIPPING INTEGRATION TEST")
		return
	}

	tests := []struct {
		name     string
		resource string
		id       string
		action   string
		username string
		password string
		err      bool
	}{
		{
			name:     "Public read access",
			resource: "books",
			action:   "read",
			username: "",
			password: "",
			err:      false,
		},
		{
			name:     "Create with editor role",
			resource: "books",
			action:   "create",
			username: "alice",
			password: "alicepass",
			err:      false,
		},
		{
			name:     "Update own post via owner field",
			resource: "books",
			id:       "book123",
			action:   "update",
			username: "bob",
			password: "bobpass",
			err:      false,
		},
		{
			name:     "Admin delete access",
			resource: "books",
			action:   "delete",
			username: "admin",
			password: "admin123",
			err:      false,
		},
		{
			name:     "Delete without admin role",
			resource: "books",
			action:   "delete",
			username: "alice",
			password: "alicepass",
			err:      true,
		},
		{
			name:     "Full access via coowner list",
			resource: "books",
			action:   "update",
			id:       "book123",
			username: "alice",
			password: "alicepass",
			err:      false,
		},
		{
			name:     "Invalid creds",
			resource: "books",
			action:   "create",
			username: "alice",
			password: "wrongpass",
			err:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schemas := map[string][]v1alpha1.FieldSchema{}
			resourceData := map[string][]struct {
				FieldSchema v1alpha1.FieldSchema
				Index       int
			}{}

			dir := testData(t, "../testdata/authz")

			schemaRW := csv.NewReadWriter(
				readwriter.WithLocation(dir + "/_schemas.csv"),
			)

			recs, err := schemaRW.List(context.Background())
			require.NoError(t, err)

			for _, rec := range recs {
				require.Equal(t, 8, len(rec))

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

			s := store.New(schemas, rws)

			err = s.Authorize(context.Background(), test.resource, test.id, test.action, test.username, test.password)

			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStoreCRUDWithCSVRW(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Log("SKIPPING INTEGRATION TEST")
		return
	}

	original := store.GenerateId
	defer func() {
		store.GenerateId = original
	}()

	tests := []struct {
		name      string
		operation func(*store.Store) error
		err       bool
		postCheck func(*store.Store) error
	}{
		{
			name: "Create valid book",
			operation: func(s *store.Store) error {
				store.GenerateId = func() string { return "test-id-1" }
				_, err := s.Create(context.Background(), "books", v1alpha1.Resource{
					"title":            "Let's Go Further",
					"author":           "Alex",
					"publication_year": 2020.0,
					"genres":           []string{"Programming"},
					"isbn":             "123-0123456789",
				})
				return err
			},
			err: false,
			postCheck: func(s *store.Store) error {
				res, err := s.ReadOne(context.Background(), "books", "test-id-1")
				if err != nil {
					return err
				}
				if res["title"] != "Let's Go Further" || res["_v"].(float64) != 1.0 {
					return errors.New("created book mismatch")
				}
				return nil
			},
		},
		{
			name: "Create invalid book (missing title)",
			operation: func(s *store.Store) error {
				_, err := s.Create(context.Background(), "books", v1alpha1.Resource{
					"author":           "Anonymous",
					"publication_year": 2023.0,
					"genres":           []string{"Mystery"},
					"isbn":             "999-999999",
				})
				return err
			},
			err: true,
		},
		{
			name: "Update book",
			operation: func(s *store.Store) error {
				store.GenerateId = func() string { return "test-id-2" }
				_, err := s.Create(context.Background(), "books", v1alpha1.Resource{
					"title":            "Original Title",
					"author":           "Author",
					"publication_year": 2020.0,
					"genres":           []string{"Old"},
					"isbn":             "111-1111111111",
				})
				if err != nil {
					return err
				}
				return s.Update(context.Background(), "books", v1alpha1.Resource{
					"_id":              "test-id-2",
					"title":            "Updated Title",
					"author":           "Author",
					"publication_year": 2020.0,
					"genres":           []string{"New"},
					"isbn":             "111-1111111111",
				})
			},
			err: false,
			postCheck: func(s *store.Store) error {
				res, err := s.ReadOne(context.Background(), "books", "test-id-2")
				if err != nil {
					return err
				}
				if res["title"] != "Updated Title" || res["_v"].(float64) != 2.0 {
					return errors.New("update failed")
				}
				return nil
			},
		},
		{
			name: "Delete book",
			operation: func(s *store.Store) error {
				store.GenerateId = func() string { return "test-id-3" }
				_, err := s.Create(context.Background(), "books", v1alpha1.Resource{
					"title":            "Marked For Deletion",
					"author":           "Author",
					"publication_year": 2021.0,
					"genres":           []string{"Temp"},
					"isbn":             "333-3333333333",
				})
				if err != nil {
					return err
				}
				return s.Delete(context.Background(), "books", "test-id-3")
			},
			err: false,
			postCheck: func(s *store.Store) error {
				_, err := s.ReadOne(context.Background(), "books", "test-id-3")
				if err == nil {
					return errors.New("book not deleted")
				}
				return nil
			},
		},
		{
			name: "List sorted books",
			operation: func(s *store.Store) error {
				store.GenerateId = func() string { return "book1" }
				_, err := s.Create(context.Background(), "books", v1alpha1.Resource{
					"title":            "Book A",
					"author":           "Author A",
					"publication_year": 2000.0,
					"genres":           []string{"Genre A"},
					"isbn":             "111-0000000000",
				})
				if err != nil {
					return err
				}
				store.GenerateId = func() string { return "book2" }
				_, err = s.Create(context.Background(), "books", v1alpha1.Resource{
					"title":            "Book B",
					"author":           "Author B",
					"publication_year": 2020.0,
					"genres":           []string{"Genre B"},
					"isbn":             "222-0000000000",
				})
				if err != nil {
					return err
				}
				return nil
			},
			err: false,
			postCheck: func(s *store.Store) error {
				books, err := s.List(context.Background(), "books", "publication_year")
				if err != nil || len(books) != 2 {
					return errors.New("list failed")
				}
				if books[0]["publication_year"].(float64) != 2000.0 || books[1]["publication_year"].(float64) != 2020.0 {
					return errors.New("incorrect sort")
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schemas := map[string][]v1alpha1.FieldSchema{}
			resourceData := map[string][]struct {
				FieldSchema v1alpha1.FieldSchema
				Index       int
			}{}

			dir := testData(t, "../testdata/basic")

			schemaRW := csv.NewReadWriter(
				readwriter.WithLocation(dir + "/_schemas.csv"),
			)

			recs, err := schemaRW.List(context.Background())
			require.NoError(t, err)

			for _, rec := range recs {
				require.Equal(t, 8, len(rec))

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

			s := store.New(schemas, rws)

			err = test.operation(s)

			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if test.postCheck != nil {
				err := test.postCheck(s)
				require.NoError(t, err)
			}
		})
	}
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
