package cloner

import (
	"fmt"
	"os"
)

// NewNilOutput does nothing for Printfln requests.
func NewNilOutput() InfoOutput {
	return InfoOutput{}
}

// NewStdOutput writes to standard out.
func NewStdOutput() InfoOutput {
	return InfoOutput{out: os.Stdout}
}

// InfoOutput is an output stream to write messages to.
type InfoOutput struct {
	out *os.File
}

// Printfln prints formatted text to the output stream and begins a new line.
func (io InfoOutput) Printfln(format string, a ...interface{}) {
	if io.out == nil {
		return
	}
	_, printErr := io.out.WriteString(fmt.Sprintf(format+"\n", a...))
	if printErr != nil {
		panic(printErr)
	}
}
