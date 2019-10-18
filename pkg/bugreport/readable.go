package bugreport

import "io"

// Readable is a read handle and an accompanying name
type Readable struct {
	io.ReadCloser
	Name string
}
