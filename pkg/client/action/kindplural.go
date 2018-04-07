package action

import (
	"reflect"
	"strings"

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

// Plural returns the plural form of the object's name in UpperCamelCase. We need this for properly
// using reflection to get the correct member since the client stubs use the plural form instead of
// singular.
// TODO(briantkennedy): convert these to taking runtime.Object type since that's what the API is
// using.
func Plural(i interface{}) string {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()
	return pluralizer.Name(&types.Type{Name: types.Name{Name: name}})
}

// LowerPlural returns the lower case plural form of the object's name.
func LowerPlural(i interface{}) string {
	return strings.ToLower(Plural(i))
}
