package bugreport

import (
	"fmt"
)

var _ error = &brNSError{}

type errName string

const (
	missingNamespace = errName("MissingNamespace")
	notManagedByACM  = errName("NotManagedByACM")
)

func errorIs(err error, name errName) bool {
	w, ok := err.(*wrappedError)
	for ok {
		err = w.child
		w, ok = err.(*wrappedError)
	}
	e, ok := err.(*brNSError)
	if !ok {
		return false
	}
	return e.Name() == name
}

type brNSError struct {
	msg       string
	name      errName
	namespace string
}

func (e brNSError) Error() string {
	return e.msg
}

func (e *brNSError) Name() errName { return e.name }

func (e *brNSError) Namespace() string { return e.namespace }

func newMissingNamespaceError(ns string) *brNSError {
	return &brNSError{msg: fmt.Sprintf("no namespace found named %s", ns), name: missingNamespace, namespace: ns}
}

func newNotManagedNamespaceError(ns string) *brNSError {
	return &brNSError{msg: fmt.Sprintf("namespace %s not managed by Anthos Config Management", ns), name: notManagedByACM, namespace: ns}
}

var _ error = &wrappedError{}

type wrappedError struct {
	child error
	msg   string
}

func (w *wrappedError) Error() string {
	return fmt.Sprintf("%s: %s", w.msg, w.child.Error())
}

func wrap(e error, msg string, args ...interface{}) *wrappedError {
	return &wrappedError{child: e, msg: fmt.Sprintf(msg, args...)}
}
