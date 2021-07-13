package hydrate

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// Filepath annotates the given Raw objects with the path from the Git repo
// policy directory to the files which declare them.
func Filepath(objs *objects.Raw) status.MultiError {
	for _, obj := range objs.Objects {
		path := objs.PolicyDir.Join(obj.Relative).SlashPath()
		core.SetAnnotation(obj, metadata.SourcePathAnnotationKey, path)
	}
	return nil
}
