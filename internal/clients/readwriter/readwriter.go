package readwriter

import (
	"github.com/w-h-a/backend/internal/clients/reader"
	"github.com/w-h-a/backend/internal/clients/writer"
)

type ReadWriter interface {
	reader.Reader
	writer.Writer
}
