package v1alpha1

import (
	"fmt"
	"regexp"
)

func ParseResource(s []FieldSchema, res Resource) (Resource, error) {
	parsed := Resource{}

	for _, fs := range s {
		if fs.Field == "_id" || fs.Field == "_v" {
			continue
		}

		v := res[fs.Field]

		var parsedValue any
		var err error

		switch fs.Type {
		case "number":
			if v == nil {
				v = 0.0
			}
			parsedValue, err = ParseField[float64](fs, v)
		case "text":
			if v == nil {
				v = ""
			}
			parsedValue, err = ParseField[string](fs, v)
		case "list":
			if v == nil {
				v = []string{}
			}
			parsedValue, err = ParseField[[]string](fs, v)
		default:
			err = fmt.Errorf("unknown field type %s during record parsing", fs.Type)
		}

		if err != nil {
			return nil, err
		}

		parsed[fs.Field] = parsedValue
	}

	return parsed, nil
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
