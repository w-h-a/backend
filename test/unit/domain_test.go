package unit

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/backend/api/v1alpha1"
)

func TestParseNumberField(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	tests := []struct {
		name  string
		fs    v1alpha1.FieldSchema
		input any
		want  float64
		err   bool
	}{
		{
			name:  "valid number within range",
			fs:    v1alpha1.FieldSchema{Type: "number", Min: 5, Max: 10},
			input: 7.0,
			want:  7.0,
			err:   false,
		},
		{
			name:  "number at min boundary",
			fs:    v1alpha1.FieldSchema{Type: "number", Min: 5, Max: 10},
			input: 5.0,
			want:  5.0,
			err:   false,
		},
		{
			name:  "number at max boundary",
			fs:    v1alpha1.FieldSchema{Type: "number", Min: 5, Max: 10},
			input: 10.0,
			want:  10.0,
			err:   false,
		},
		{
			name:  "number below min",
			fs:    v1alpha1.FieldSchema{Type: "number", Min: 5, Max: 10},
			input: 4.9,
			want:  0,
			err:   true,
		},
		{
			name:  "number above max",
			fs:    v1alpha1.FieldSchema{Type: "number", Min: 5, Max: 10},
			input: 10.1,
			want:  0,
			err:   true,
		},
		{
			name:  "not a number",
			fs:    v1alpha1.FieldSchema{Type: "number"},
			input: "not a number",
			want:  0,
			err:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, err := v1alpha1.ParseField[float64](test.fs, test.input)
			if !test.err {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, test.want, v)
		})
	}
}

func TestParseTextField(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	tests := []struct {
		name  string
		fs    v1alpha1.FieldSchema
		input any
		want  string
		err   bool
	}{
		{
			name:  "text matches regex",
			fs:    v1alpha1.FieldSchema{Type: "text", Regex: "^[a-z]+$"},
			input: "lowercase",
			want:  "lowercase",
			err:   false,
		},
		{
			name:  "empty text with regex",
			fs:    v1alpha1.FieldSchema{Type: "text", Regex: "^.*$"},
			input: "",
			want:  "",
			err:   false,
		},
		{
			name:  "text doesn't match regex",
			fs:    v1alpha1.FieldSchema{Type: "text", Regex: "^[a-z]+$"},
			input: "Uppdercase",
			want:  "",
			err:   true,
		},
		{
			name:  "not text",
			fs:    v1alpha1.FieldSchema{Type: "text"},
			input: 1000,
			want:  "",
			err:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, err := v1alpha1.ParseField[string](test.fs, test.input)
			if !test.err {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, test.want, v)
		})
	}
}

func TestParseListField(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	tests := []struct {
		name  string
		fs    v1alpha1.FieldSchema
		input any
		want  []string
		err   bool
	}{
		{
			name:  "valid string list",
			fs:    v1alpha1.FieldSchema{Type: "list"},
			input: []string{"a", "b"},
			want:  []string{"a", "b"},
			err:   false,
		},
		{
			name:  "empty list",
			fs:    v1alpha1.FieldSchema{Type: "list"},
			input: []string{},
			want:  []string{},
			err:   false,
		},
		{
			name:  "not a list of strings",
			fs:    v1alpha1.FieldSchema{Type: "list"},
			input: []float64{1.9},
			want:  nil,
			err:   true,
		},
		{
			name:  "not a list",
			fs:    v1alpha1.FieldSchema{Type: "list"},
			input: "not a list",
			want:  nil,
			err:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, err := v1alpha1.ParseField[[]string](test.fs, test.input)
			if !test.err {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, test.want, v)
		})
	}
}

func TestParseResource(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	testSchema := []v1alpha1.FieldSchema{
		{Field: "_id", Type: "text", Regex: "^[A-Za-z0-9]+$"},
		{Field: "_v", Type: "number", Min: 1},
		{Field: "name", Type: "text", Regex: "^[A-Z][a-z]*$"},
		{Field: "age", Type: "number", Min: 0, Max: 150},
		{Field: "tags", Type: "list"},
	}

	tests := []struct {
		name     string
		resource v1alpha1.Resource
		expected v1alpha1.Resource
		err      bool
	}{
		{
			name: "complete resource",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "John",
				"age":  30.0,
				"tags": []string{"admin", "user"},
			},
			expected: v1alpha1.Resource{
				"name": "John",
				"age":  30.0,
				"tags": []string{"admin", "user"},
			},
			err: false,
		},
		{
			name: "missing optional fields",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "John",
			},
			expected: v1alpha1.Resource{
				"name": "John",
				"age":  0.0,
				"tags": []string{},
			},
			err: false,
		},
		{
			name: "invalid id",
			resource: v1alpha1.Resource{
				"_id":  "?",
				"_v":   1.0,
				"name": "John",
			},
			expected: v1alpha1.Resource{
				"name": "John",
				"age":  0.0,
				"tags": []string{},
			},
			err: false,
		},
		{
			name: "invalid version",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   0.0,
				"name": "John",
			},
			expected: v1alpha1.Resource{
				"name": "John",
				"age":  0.0,
				"tags": []string{},
			},
			err: false,
		},
		{
			name: "invalid name",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "john",
				"age":  30.0,
			},
			err: true,
		},
		{
			name: "invalid age",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "John",
				"age":  200.0,
			},
			err: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := v1alpha1.ParseResource(testSchema, test.resource)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(test.expected), len(parsed))
				require.Equal(t, test.expected, parsed)
			}
		})
	}
}

func TestToRecord(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	testSchema := []v1alpha1.FieldSchema{
		{Field: "_id", Type: "text", Regex: "^[A-Za-z0-9]+$"},
		{Field: "_v", Type: "number", Min: 1},
		{Field: "name", Type: "text", Regex: "^[A-Z][a-z]*$"},
		{Field: "age", Type: "number", Min: 0, Max: 150},
		{Field: "tags", Type: "list"},
	}

	tests := []struct {
		name     string
		resource v1alpha1.Resource
		expected v1alpha1.Record
		err      bool
	}{
		{
			name: "complete resource",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "John",
				"age":  30.0,
				"tags": []string{"admin", "user"},
			},
			expected: v1alpha1.Record{
				"test007",
				"1",
				"John",
				"30",
				"admin,user",
			},
			err: false,
		},
		{
			name: "missing optional fields",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "John",
			},
			expected: v1alpha1.Record{
				"test007",
				"1",
				"John",
				"0",
				"",
			},
			err: false,
		},
		{
			name: "invalid id",
			resource: v1alpha1.Resource{
				"_id": "?",
				"_v":  1.0,
			},
			expected: v1alpha1.Record{
				"?",
				"1",
				"",
				"0",
				"",
			},
			err: false,
		},
		{
			name: "invalid version",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   0.0,
				"name": "John",
			},
			expected: v1alpha1.Record{
				"test007",
				"0",
				"John",
				"0",
				"",
			},
			err: false,
		},
		{
			name: "invalid name",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "john",
				"age":  30.0,
			},
			expected: v1alpha1.Record{
				"test007",
				"1",
				"john",
				"30",
				"",
			},
			err: false,
		},
		{
			name: "invalid age",
			resource: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "John",
				"age":  200.0,
			},
			expected: v1alpha1.Record{
				"test007",
				"1",
				"John",
				"200",
				"",
			},
			err: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec, err := v1alpha1.ToRecord(testSchema, test.resource)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(test.expected), len(rec))
				require.Equal(t, test.expected, rec)
			}
		})
	}
}

func TestToResource(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	testSchema := []v1alpha1.FieldSchema{
		{Field: "_id", Type: "text", Regex: "^[A-Za-z0-9]+$"},
		{Field: "_v", Type: "number", Min: 1},
		{Field: "name", Type: "text", Regex: "^[A-Z][a-z]*$"},
		{Field: "age", Type: "number", Min: 0, Max: 150},
		{Field: "tags", Type: "list"},
	}

	tests := []struct {
		name     string
		record   v1alpha1.Record
		expected v1alpha1.Resource
		err      bool
	}{
		{
			name: "filled out record",
			record: v1alpha1.Record{
				"test007",
				"2",
				"Alice",
				"25",
				"staff,manager",
			},
			expected: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   2.0,
				"name": "Alice",
				"age":  25.0,
				"tags": []string{"staff", "manager"},
			},
			err: false,
		},
		{
			name: "empty fields",
			record: v1alpha1.Record{
				"test007",
				"1",
				"",
				"0",
				"",
			},
			expected: v1alpha1.Resource{
				"_id":  "test007",
				"_v":   1.0,
				"name": "",
				"age":  0.0,
				"tags": []string{},
			},
			err: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := v1alpha1.ToResource(testSchema, test.record)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(test.expected), len(res))
				require.Equal(t, test.expected, res)
			}
		})
	}
}

func TestEdgeCase(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) > 0 {
		t.Log("SKIPPING UNIT TEST")
		return
	}

	schema := []v1alpha1.FieldSchema{}

	rec, err := v1alpha1.ToRecord(schema, v1alpha1.Resource{})
	require.NoError(t, err)
	require.True(t, len(rec) == 0)

	res, err := v1alpha1.ToResource(schema, rec)
	require.NoError(t, err)
	require.True(t, len(res) == 0)
}
