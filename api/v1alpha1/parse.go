package v1alpha1

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseRecord(s []FieldSchema, res Resource) (Record, error) {
	rec := Record{}

	for _, fs := range s {
		v := res[fs.Field]
		switch fs.Type {
		case "number":
			if v == nil {
				v = 0.0
			}
			n, err := ParseField[float64](fs, v)
			if err != nil {
				return nil, err
			}
			rec = append(rec, fmt.Sprintf("%g", n))
		case "text":
			if v == nil {
				v = ""
			}
			t, err := ParseField[string](fs, v)
			if err != nil {
				return nil, err
			}
			rec = append(rec, t)
		case "list":
			if v == nil {
				v = []string{}
			}
			l, err := ParseField[[]string](fs, v)
			if err != nil {
				return nil, err
			}
			rec = append(rec, strings.Join(l, ","))
		default:
			return nil, fmt.Errorf("unknown field type %s during record parsing", fs.Type)
		}
	}

	return rec, nil
}

type FieldType interface {
	float64 | string | []string
}

func ParseField[T FieldType](fs FieldSchema, v any) (T, error) {
	var result T

	switch any(result).(type) {
	case float64:
		n, ok := v.(float64)
		if !ok {
			return result, fmt.Errorf("failed to parse field \"%s\" as a number", fs.Field)
		}
		ok = (fs.Min == 0 && fs.Max == 0) ||
			(n >= fs.Min && (fs.Max < fs.Min || n <= fs.Max))
		if !ok {
			return result, fmt.Errorf("failed to parse field \"%s\" as a valid number", fs.Field)
		}
		return any(n).(T), nil
	case string:
		t, ok := v.(string)
		if !ok {
			return result, fmt.Errorf("failed to parse field \"%s\" as a string", fs.Field)
		}
		if len(fs.Regex) > 0 {
			matched, err := regexp.MatchString(fs.Regex, t)
			if err != nil {
				return result, fmt.Errorf("invalid regex for field \"%s\": %w", fs.Field, err)
			}
			if !matched {
				return result, fmt.Errorf("failed to parse field \"%s\" as a valid string", fs.Field)
			}
		}
		return any(t).(T), nil
	case []string:
		l, ok := v.([]string)
		if !ok {
			return result, fmt.Errorf("failed to parse field \"%s\" as a list", fs.Field)
		}
		return any(l).(T), nil
	default:
		return result, fmt.Errorf("unsupported generic type %T", result)
	}
}
