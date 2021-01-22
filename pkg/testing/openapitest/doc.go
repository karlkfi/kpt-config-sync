package openapitest

import (
	"io/ioutil"
	// The openapi Document type does not satisfy the proto.Message interface in
	// the new non-deprecated proto library.
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
)

// Doc returns an openapi Document intialized from a static file of openapi
// schemas.
func Doc(path string) (*openapi_v2.Document, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := &openapi_v2.Document{}
	if err := proto.Unmarshal(data, doc); err != nil {
		return nil, err
	}

	return doc, err
}
