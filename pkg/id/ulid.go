package id

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/oklog/ulid/v2"
)

// New gera um novo ULID e o retorna como string.
func New() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	return id.String()
}
