package util

import "fmt"

// PolicyControllerError is the shared error schema for Gatekeeper constraints
// and constraint templates.
type PolicyControllerError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

// FormatErrors flattens the given errors into a string array.
func FormatErrors(id string, pces []PolicyControllerError) []string {
	var errs []string
	for _, pce := range pces {
		var prefix string
		if len(pce.Code) > 0 {
			prefix = fmt.Sprintf("[%s] %s:", id, pce.Code)
		} else {
			prefix = fmt.Sprintf("[%s]:", id)
		}

		msg := pce.Message
		if len(msg) == 0 {
			msg = "[missing PolicyController error]"
		}

		errs = append(errs, fmt.Sprintf("%s %s", prefix, msg))
	}
	return errs
}
