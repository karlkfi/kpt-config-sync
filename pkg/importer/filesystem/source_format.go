package filesystem

// SourceFormat specifies how the Importer should parse the repository.
type SourceFormat string

// SourceFormatUnstructured says to parse all YAMLs in the config directory and
// ignore directory structure.
const SourceFormatUnstructured SourceFormat = "unstructured"

// SourceFormatHierarchy says to use hierarchical namespace inheritance based on
// directory structure and requires that manifests be declared in specific
// subdirectories.
const SourceFormatHierarchy SourceFormat = "hierarchy"

// SourceFormatKey is the OS env variable and ConfigMap key for the SOT
// repository format.
const SourceFormatKey = "SOURCE_FORMAT"
