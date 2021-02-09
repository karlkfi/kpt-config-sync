package applier

import (
	"flag"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
)

func TestMain(m *testing.M) {
	// This TestMain function is required in every e2e test case file.
	flag.Parse()

	if !*e2e.E2E || !*e2e.MultiRepo {
		return
	}
	rand.Seed(time.Now().UnixNano())

	os.Exit(m.Run())
}
