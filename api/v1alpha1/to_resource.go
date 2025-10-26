package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"
)

func ToResource(s []FieldSchema, rec Record) (Resource, error) {
	res := Resource{}

	for i, fs := range s {
		if i >= len(rec) {
			return nil, fmt.Errorf("record length %d is less than schema length %d", len(rec), len(s))
		}

		strValue := rec[i]

		v, err := FormatResourceField(fs, strValue)
		if err != nil {
			return nil, err
		}

		res[fs.Field] = v
	}

	return res, nil
}

func FormatResourceField(fs FieldSchema, strValue string) (any, error) {
	switch fs.Type {
	case "number":
		n, _ := strconv.ParseFloat(strValue, 64)
		return n, nil
	case "text":
		return strValue, nil
	case "list":
		if len(strValue) > 0 {
			return strings.Split(strValue, ","), nil
		} else {
			return []string{}, nil
		}
	default:
		return nil, fmt.Errorf("unknown schema type %s during resource formatting", fs.Type)
	}
}
