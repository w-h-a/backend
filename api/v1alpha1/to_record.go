package v1alpha1

import (
	"fmt"
	"strings"
)

func ToRecord(s []FieldSchema, res Resource) (Record, error) {
	recordLen := len(s)
	rec := make(Record, recordLen)

	for i, fs := range s {
		v := res[fs.Field]

		formattedValue, err := FormatField(fs, v)
		if err != nil {
			return nil, err
		}

		rec[i] = formattedValue
	}

	return rec, nil
}

func FormatField(fs FieldSchema, v any) (string, error) {
	switch fs.Type {
	case "number":
		if v == nil {
			v = 0.0
		}
		if n, ok := v.(float64); ok {
			return fmt.Sprintf("%g", n), nil
		}
		return "", fmt.Errorf("internal error: expected float64 for field '%s', got %T", fs.Field, v)
	case "text":
		if v == nil {
			v = ""
		}
		if t, ok := v.(string); ok {
			return t, nil
		}
		return "", fmt.Errorf("internal error: expected string for field '%s', got %T", fs.Field, v)
	case "list":
		if v == nil {
			v = []string{}
		}
		if l, ok := v.([]string); ok {
			return strings.Join(l, ","), nil
		}
		return "", fmt.Errorf("internal error: expected []string for field '%s', got %T", fs.Field, v)
	default:
		return "", fmt.Errorf("unknown schema type '%s' during formatting", fs.Type)
	}
}
