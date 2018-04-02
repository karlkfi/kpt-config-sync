package exec

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/glog"
)

// SetTestBinary instructs this module to start a different binary instead,
// and substitute the environment variables as given below.  The process to be
// started will get the stdin, stderr and stdout as set in the exec module.
func setTestBinary(env []string, binary string, args []string) {
	glog.Infof("Substituting: binary: %q, args: %#v, env: %#v", binary, args, env)
	testBinary = binary
	testArgs = append([]string{binary}, args...)
	testEnv = env
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		// The text to supply at stdin.
		stdin string
		// The text to print on stdout and stderr.
		stdout, stderr string
		// The text to expect after the program completes.
		expectedStdout, expectedStderr string
	}{
		{
			name:           "First",
			stdin:          "input_text",
			stdout:         "output_text",
			stderr:         "error_text",
			expectedStdout: "stdout:input_text:output_text",
			expectedStderr: "stderr:input_text:error_text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run a specific test case instead of a binary so that the
			// exec interface can be tested.
			// Taken from: https://npf.io/2015/06/testing-exec-command/
			setTestBinary(
				[]string{
					fmt.Sprintf("GO_STDOUT_TEXT=%v", tt.stdout),
					fmt.Sprintf("GO_STDERR_TEXT=%v", tt.stderr),
					fmt.Sprintf("GO_EXITCODE=0"),
					"GO_WANT_HELPER_PROCESS=1",
				},
				os.Args[0],
				[]string{
					"-test.run=TestHelperProcess",
					"--",
				},
			)
			out := bytes.NewBuffer(nil)
			err := bytes.NewBuffer(nil)
			c := NewRedirected(strings.NewReader(tt.stdin), out, err)
			c.Run(context.Background(), "foo", "bar")
			if tt.expectedStdout != out.String() {
				t.Errorf("out.String(): %q, want: %q", out.String(), tt.stdout)
			}
			if tt.expectedStderr != err.String() {
				t.Errorf("err.String(): %q, want: %q", err.String(), tt.stderr)
			}
		})
	}
}

// TestHelperProcess is a subprocess that will be ran by TestRun to substitute
// the binary under test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	stdoutText := os.Getenv("GO_STDOUT_TEXT")
	stderrText := os.Getenv("GO_STDERR_TEXT")
	c, err := strconv.Atoi(os.Getenv("GO_EXITCODE"))
	if err != nil {
		t.Fatalf("exit code problem: %v", err)
	}
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		t.Fatalf("could not read stdin: %v", err)
	}
	fmt.Fprintf(os.Stdout, "stdout:%v:%v", string(b), stdoutText)
	fmt.Fprintf(os.Stderr, "stderr:%v:%v", string(b), stderrText)
	os.Stderr.Sync()
	os.Exit(c)
}

func TestRunWithEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{
			name: "Basic",
			env:  "SOME_KEY=SOME_VALUE",
		},
		{
			name: "Another test",
			env:  "SOME_KEY_2=SOME_VALUE_2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestBinary(
				[]string{
					"GO_WANT_HELPER_PROCESS=1",
				},
				os.Args[0],
				[]string{
					"-test.run=TestRunWithEnvHelper",
					"--",
				},
			)
			out := bytes.NewBuffer(nil)
			err := bytes.NewBuffer(nil)
			c := NewRedirected(strings.NewReader(""), out, err)
			c.SetEnv([]string{tt.env})
			c.Run(context.Background(), "dummy_command")
			if strings.Index(out.String(), tt.env) < 0 {
				t.Errorf("Unexpected: %v, want: %v", out.String(), tt.env)
			}
		})
	}
}

func TestRunWithEnvHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	for _, env := range os.Environ() {
		fmt.Printf("%v\n", env)
	}
	os.Stdout.Sync()
}
