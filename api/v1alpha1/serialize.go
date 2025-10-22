package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"
)

func SerializeResource(s []FieldSchema, rec Record) (Resource, error) {
	res := Resource{}

	for i, fs := range s {
		if i >= len(rec) {
			return nil, fmt.Errorf("record length %d is less than schema length %d", len(rec), len(s))
		}

		strValue := rec[i]

		switch fs.Type {
		case "number":
			n, _ := strconv.ParseFloat(strValue, 64)
			res[fs.Field] = n
		case "text":
			res[fs.Field] = strValue
		case "list":
			if len(strValue) > 0 {
				res[fs.Field] = strings.Split(strValue, ",")
			} else {
				res[fs.Field] = []string{}
			}
		default:
			return nil, fmt.Errorf("unknown field type %s during resource serialization", fs.Type)
		}
	}

	return res, nil
}
