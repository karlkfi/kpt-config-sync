package nomostest

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/nomos/e2e/nomostest/testing"
	"k8s.io/apimachinery/pkg/util/validation"
)

// re splits strings at word boundaries. Test names always begin with "Test",
// so we know we aren't missing any of the text.
var re = regexp.MustCompile(`[A-Z][^A-Z]*`)

func testClusterName(t testing.NTB) string {
	t.Helper()

	// Kind seems to allow a max cluster name length of 49.  If we exceed that, hash the
	// name, truncate to 40 chars then append 8 hash digits (32 bits).
	const nameLimit = 49
	const hashChars = 8
	n := testDirName(t)
	// handle legacy testcase filenames
	n = strings.ReplaceAll(n, "_", "-")

	if nameLimit < len(n) {
		hashBytes := sha1.Sum([]byte(n))
		hashStr := hex.EncodeToString(hashBytes[:])
		n = fmt.Sprintf("%s-%s", n[:nameLimit-1-hashChars], hashStr[:hashChars])
	}

	if errs := validation.IsDNS1123Subdomain(n); len(errs) > 0 {
		t.Fatalf("transformed test name %q into %q, which is not a valid Kind cluster name: %+v",
			t.Name(), n, errs)
	}
	return n
}

func testDirName(t testing.NTB) string {
	t.Helper()

	n := t.Name()
	// Capital letters are forbidden in Kind cluster names, so convert to
	// kebab-case.
	words := re.FindAllString(n, -1)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}

	n = strings.Join(words, "-")
	n = strings.ReplaceAll(n, "/", "--")
	return n
}
