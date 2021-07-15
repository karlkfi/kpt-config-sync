package ntopts

// TestType represents the test type.
type TestType struct {
	// LoadTest specifies the test is a load test.
	LoadTest bool

	// StressTest specifies the test is a stress test.
	StressTest bool
}

// LoadTest specifies the test is a load test.
func LoadTest(opt *New) {
	opt.LoadTest = true
}

// StressTest specifies the test is a stress test.
func StressTest(opt *New) {
	opt.StressTest = true
}
