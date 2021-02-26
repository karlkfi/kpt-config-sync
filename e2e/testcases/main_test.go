package e2e

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest"
)

func TestMain(m *testing.M) {
	// This TestMain function is required in every e2e test case file.
	flag.Parse()

	if !*e2e.E2E {
		return
	}
	rand.Seed(time.Now().UnixNano())

	if *e2e.ShareTestEnv {
		if e2e.RunInParallel() {
			fmt.Println("The test cannot use a shared test environment if it is running in parallel")
			os.Exit(1)
		}
		sharedNT := nomostest.NewSharedNT()
		exitCode := m.Run()
		nomostest.Clean(sharedNT, false)
		os.Exit(exitCode)
	} else {
		os.Exit(m.Run())
	}
}
