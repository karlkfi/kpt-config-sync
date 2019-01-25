package visitor_test

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

func TestBase(t *testing.T) {
	v := &testVisitor{
		fmt: "%s mutated value (pointers not equal)",
	}
	b := visitor.NewBase()
	b.SetImpl(v)
	v.Visitor = b

	input := vt.Helper.AcmeRoot()
	out := input.Accept(v)
	if out != input {
		t.Errorf("ouptut and input have different pointer value")
	}
	v.Check(t)
}
