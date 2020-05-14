package nomostest

import (
	"regexp"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

// re splits strings at word boundaries. Test names always begin with "Test",
// so we know we aren't missing any of the text.
var re = regexp.MustCompile(`[A-Z][^A-Z]*`)

func testName(t *testing.T) string {
	t.Helper()

	n := t.Name()
	// Capital letters are forbidden in Kind cluster names, so convert to
	// kebab-case.
	words := re.FindAllString(n, -1)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}

	n = strings.Join(words, "-")
	if errs := validation.IsDNS1123Subdomain(n); len(errs) > 0 {
		t.Fatalf("transformed test name %q into %q, which is not a valid Kind cluster name: %+v",
			t.Name(), n, errs)
	}
	return n
}
