package logging

import (
	"io"
)

// MultiWriter creates a writer that duplicates its writes to all the provided writers
func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}
