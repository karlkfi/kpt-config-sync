package openapitest

import (
	"io/ioutil"
	// The openapi Document type does not satisfy the proto.Message interface in
	// the new non-deprecated proto library.
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	openapiv2 "github.com/googleapis/gnostic/openapiv2"
)

// Doc returns an openapi Document intialized from a static file of openapi
// schemas.
func Doc(path string) (*openapiv2.Document, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := &openapiv2.Document{}
	if err := proto.Unmarshal(data, doc); err != nil {
		return nil, err
	}

	return doc, err
}
