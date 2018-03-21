package action

import (
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
)

var exceptions = map[string]string{}
var pluralizer = namer.NewPublicPluralNamer(exceptions)

// AddPluralException adds an exception for pluralization.
func AddPluralException(name, value string) {
	exceptions[name] = value
	pluralizer = namer.NewPublicPluralNamer(exceptions)
}

// Plural returns the plural form of the object with name in UpperCamelCase. We need this for
// properly using reflection to get the correct member since the client stubs use the plural form
// instead of singular.
func Plural(name string) string {
	return pluralizer.Name(&types.Type{Name: types.Name{Name: name}})
}
