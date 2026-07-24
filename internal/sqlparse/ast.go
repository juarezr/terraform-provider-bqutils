package sqlparse

import "fmt"

// ObjectKind classifies the CREATE statement.
type ObjectKind string

const (
	KindScalarFunction    ObjectKind = "SCALAR_FUNCTION"
	KindTableFunction     ObjectKind = "TABLE_VALUED_FUNCTION"
	KindProcedure         ObjectKind = "PROCEDURE"
	KindAggregateFunction ObjectKind = "AGGREGATE_FUNCTION"
	KindView              ObjectKind = "VIEW"
	KindMaterializedView  ObjectKind = "MATERIALIZED_VIEW"
)

// Argument is a routine argument.
type Argument struct {
	Name         string
	DataTypeJSON string
	ArgumentKind string // FIXED_TYPE, ANY_TYPE
	Mode         string // IN, OUT, INOUT
	// IsAggregate mirrors BigQuery Argument.is_aggregate for UDAF parameters.
	// nil = unspecified (non-UDAF); false = NOT AGGREGATE; true = aggregate parameter.
	IsAggregate *bool
}

// ColumnDef is a view column list entry.
type ColumnDef struct {
	Name        string
	Description string
}

// SparkOptions maps to google_bigquery_routine.spark_options.
type SparkOptions struct {
	Connection     string
	RuntimeVersion string
	ContainerImage string
	Properties     map[string]string
	MainFileURI    string
	PyFileURIs     []string
	JarURIs        []string
	FileURIs       []string
	ArchiveURIs    []string
	MainClass      string
}

// PythonOptions maps to google_bigquery_routine.python_options.
type PythonOptions struct {
	EntryPoint string
	Packages   []string
}

// ExternalRuntimeOptions maps to google_bigquery_routine.external_runtime_options.
type ExternalRuntimeOptions struct {
	ContainerMemory             string
	ContainerCPU                string
	RuntimeConnection           string
	MaxBatchingRows             string
	RuntimeVersion              string
	ContainerRequestConcurrency string
}

// RemoteFunctionOptions maps to google_bigquery_routine.remote_function_options.
type RemoteFunctionOptions struct {
	Endpoint           string
	Connection         string
	MaxBatchingRows    string
	UserDefinedContext map[string]string
}

// ParseResult holds parsed CREATE statement fields.
type ParseResult struct {
	Kind           ObjectKind
	Project        string
	DatasetID      string
	ObjectID       string
	IsTemporary    bool
	IsMaterialized bool

	Language            string
	DefinitionBody      string
	Query               string
	ReturnTypeJSON      string
	ReturnTableTypeJSON string
	Arguments           []Argument

	Description        string
	FriendlyName       string
	Labels             map[string]string
	ImportedLibraries  []string
	DeterminismLevel   string
	DataGovernanceType string

	EnableRefresh                 *bool
	AllowNonIncrementalDefinition *bool
	RefreshIntervalMs             *int64
	MaxStaleness                  string
	KmsKeyName                    string
	PartitioningType              string
	PartitioningField             string
	Clustering                    []string
	SchemaJSON                    string
	Columns                       []ColumnDef

	SparkOptions           *SparkOptions
	PythonOptions          *PythonOptions
	ExternalRuntimeOptions *ExternalRuntimeOptions
	RemoteFunctionOptions  *RemoteFunctionOptions

	// Legacy aliases filled for compatibility with older provider wiring.
	SparkOptionsJSON          string
	RemoteFunctionOptionsJSON string
	RemoteConnection          string
	RemoteEndpoint            string
}

// ParseError is a positioned parse failure.
type ParseError struct {
	Message string
	Line    int
	Column  int
	Offset  int
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d, col %d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

func (r *ParseResult) ensureSpark() *SparkOptions {
	if r.SparkOptions == nil {
		r.SparkOptions = &SparkOptions{}
	}
	return r.SparkOptions
}

func (r *ParseResult) ensurePython() *PythonOptions {
	if r.PythonOptions == nil {
		r.PythonOptions = &PythonOptions{}
	}
	return r.PythonOptions
}

func (r *ParseResult) ensureExternalRuntime() *ExternalRuntimeOptions {
	if r.ExternalRuntimeOptions == nil {
		r.ExternalRuntimeOptions = &ExternalRuntimeOptions{}
	}
	return r.ExternalRuntimeOptions
}

func (r *ParseResult) ensureRemote() *RemoteFunctionOptions {
	if r.RemoteFunctionOptions == nil {
		r.RemoteFunctionOptions = &RemoteFunctionOptions{}
	}
	return r.RemoteFunctionOptions
}
