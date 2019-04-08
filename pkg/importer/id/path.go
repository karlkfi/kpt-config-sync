package id

// Path represents an object associated with a path in a Nomos repository.
type Path interface {
	// SlashPath returns the slash-delimited path.
	SlashPath() string

	// OSPath returns the OS-specific path.
	OSPath() string
}
