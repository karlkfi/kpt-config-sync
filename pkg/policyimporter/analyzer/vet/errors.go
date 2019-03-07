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

package vet

import (
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

var codes = map[string]bool{
	// Obsolete error codes. Do not reuse.
	"1023": true,
}

// Examples is the canonical example errors for each code.
var Examples = make(map[string][]status.Error)

// Explanations is the error documentation for each code.
var Explanations = make(map[string]string)

func register(code string, examples []status.Error, explanation string) {
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
