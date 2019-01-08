/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package veterrors

import (
	"fmt"

	"github.com/pkg/errors"
)

var codes = map[string]bool{
	// Obsolete error codes. Do not reuse.
	"1016": true,
	"1023": true,

	// The unknown error code.
	UndefinedErrorCode: true,
}

// Examples is the canonical example errors for each code.
var Examples = make(map[string][]Error)

// Explanations is the error documentation for each code.
var Explanations = make(map[string]string)

func register(code string, examples []Error, explanation string) {
	if codes[code] {
		panic(errors.Errorf("duplicate error code: %q", code))
	}
	codes[code] = true

	for _, example := range examples {
		if example.Code() != code {
			panic(errors.Errorf("supplied error code %q does not match code of example %q", code, example.Code()))
		}
	}

	Examples[code] = examples
	Explanations[code] = explanation
}

// UndefinedErrorCode is the code representing an unregistered error. These should be eliminated.
const UndefinedErrorCode = "????"

// Error Defines a Kubernetes Nomos Vet error
// These are GKE Policy Management directory errors which are shown to the user and documented.
type Error interface {
	Error() string
	Code() string
}

// withPrefix formats the start of error messages consistently.
func format(err Error, format string, a ...interface{}) string {
	return fmt.Sprintf("KNV%s: ", err.Code()) + fmt.Sprintf(format, a...)
}
