package nomos

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	workingDirectory = "."
)

// WorkingDirectoryPath is the path to the working directory
var WorkingDirectoryPath = FilePath{workingDirectory}

// FilePath represents a (possibly invalid) path to a file or directory in a Nomos directory.
type FilePath struct {
	path string
}

var _ pflag.Value = &FilePath{}

// String implements Value
func (p FilePath) String() string {
	return p.path
}

// Set implements Value
func (p *FilePath) Set(str string) error {
	_, err := os.Stat(str)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Wrapf(err, "supplied path does not exist %q", str)
		} else if os.IsPermission(err) {
			return errors.Wrapf(err, "insufficient permissions to read %q", str)
		}
		return errors.Wrapf(err, "unable to read %q", str)
	}

	p.path = str
	return nil
}

// Type implements Value
func (p FilePath) Type() string {
	return "FilePath"
}

// Path returns the contained path
func (p FilePath) Path() string {
	return p.path
}
