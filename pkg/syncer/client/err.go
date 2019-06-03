package client

// noUpdateNeededError is returned if no update is needed for the given resource.
type noUpdateNeededError struct {
}

// Error implements error
func (e *noUpdateNeededError) Error() string {
	return "noUpdateNeededError"
}

// NoUpdateNeeded returns an error code for update not required.
func NoUpdateNeeded() error {
	return &noUpdateNeededError{}
}

// IsNoUpdateNeeded checks for whether the returned error is noUpdateNeededError
func IsNoUpdateNeeded(err error) bool {
	_, ok := err.(*noUpdateNeededError)
	return ok
}
