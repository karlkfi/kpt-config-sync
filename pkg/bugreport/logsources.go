package bugreport

import (
	"context"
	"fmt"
	"io"
)

// convertibleLogSourceIdentifiers is an interface for mocking the logSource type in unit tests
type convertibleLogSourceIdentifiers interface {
	fetchRcForLogSource(context.Context, coreClient) (io.ReadCloser, error)
	pathName() string
}

type logSources []convertibleLogSourceIdentifiers

func (ls logSources) convertLogSourcesToReadables(ctx context.Context, cs coreClient) ([]Readable, []error) {
	var rs []Readable
	var errorList []error

	for _, l := range ls {
		rc, err := l.fetchRcForLogSource(ctx, cs)
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
