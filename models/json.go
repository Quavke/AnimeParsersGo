package models

import (
	"io"
)

type JSONResponse interface {
	Decode(r io.Reader) error
}
