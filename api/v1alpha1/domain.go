package v1alpha1

type Resource map[string]any

type Record []string

type FieldSchema struct {
	Field string
	Type  string
	Min   float64
	Max   float64
	Regex string
}
