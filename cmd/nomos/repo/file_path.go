package repo

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	workingDirectory = "."
	// PathFlag is the key to use to set adn retreive the path value.
	PathFlag = "path"
)

// WorkingDirectoryPath is the path to the working directory
var WorkingDirectoryPath = FilePath{workingDirectory, true}

// FilePath represents a (possibly invalid) path to a file or directory in a Nomos directory.
type FilePath struct {
	string
	Exists bool
}

var _ pflag.Value = &FilePath{}

// String implements Value
func (p FilePath) String() string {
	return p.string
}

// Set implements Value
func (p *FilePath) Set(str string) error {
	_, err := os.Stat(str)
	if err != nil {
		if os.IsPermission(err) {
			return errors.Wrapf(err, "insufficient permissions to read %q", str)
		} else if os.IsNotExist(err) {
			// The method using the dir can decide what to do if the directory does not exist.
			p.Exists = false
		} else {
			return errors.Wrapf(err, "unable to read %q", str)
		}
	} else {
		p.Exists = true
	}

	p.string = str
	return nil
}

// Type implements Value
func (p FilePath) Type() string {
	return "FilePath"
}
