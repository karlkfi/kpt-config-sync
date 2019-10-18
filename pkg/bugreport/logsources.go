package bugreport

import (
	"fmt"
	"io"
)

// convertibleLogSourceIdentifiers is an interface for mocking the logSource type in unit tests
type convertibleLogSourceIdentifiers interface {
	fetchRcForLogSource(coreClient) (io.ReadCloser, error)
	pathName() string
}

type logSources []convertibleLogSourceIdentifiers

func (ls logSources) convertLogSourcesToReadables(cs coreClient) ([]Readable, []error) {
	var rs []Readable
	var errorList []error

	for _, l := range ls {
		rc, err := l.fetchRcForLogSource(cs)
		if err != nil {
			e := fmt.Errorf("failed to create reader for logSource: %v", err)
			errorList = append(errorList, e)
			continue
		}

		rs = append(rs, Readable{
			ReadCloser: rc,
			Name:       l.pathName(),
		})
	}

	return rs, errorList
}
