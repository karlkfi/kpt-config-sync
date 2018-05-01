package configgen

import (
	"fmt"
	"os"

	"github.com/google/nomos/pkg/installer/config"
	"github.com/pkg/errors"
)

var _ Action = (*save)(nil)

// save is an Action that stores the current configuration into the supplied
// file.
type save struct {
	// out is the name of the output configuration file to write.
	out string
	// The configuration to write out. cfg is shared with the configgen proper.
	cfg *config.Config
	// The last error encountered.
	lasterr error
	// Set to true if a configuration was previously written through Run() with
	// success.
	written bool
}

// NewSave creates a new save Action, writing the supplied configuration to the
// supplied out filename.
func newSave(out string, cfg *config.Config) *save {
	return &save{out: out, cfg: cfg}
}

// Text implements Action.
func (s *save) Text() string {
	text := fmt.Sprintf("Save the current configuration [%v]", s.out)
	if s.written {
		if s.lasterr != nil {
			text = fmt.Sprintf("%v [%v]", text, s.lasterr)
		} else {
			text = fmt.Sprintf("%v [SUCCESS]", text)
		}
	}
	return text
}

// Name implements Action.
func (s *save) Name() string {
	return "Save"
}

// Run implements Action.
func (s *save) Run() (bool, error) {
	var err error
	defer func() {
		// Set on the first run only.
		s.written = true
		s.lasterr = err
	}()
	f, err := os.Create(s.out)
	if err != nil {
		return false, errors.Wrapf(err, "while creating file: %q", s.out)
	}
	defer func() {
		err = f.Close()
	}()
	err = s.cfg.WriteInto(f)
	if err != nil {
		return false, errors.Wrapf(err, "while creating config: %q", s.out)
	}
	return false, err
}
