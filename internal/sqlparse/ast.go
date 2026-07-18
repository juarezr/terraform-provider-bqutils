package sqlparse

import "fmt"

// ObjectKind classifies the CREATE statement.
type ObjectKind string

const (
	KindScalarFunction    ObjectKind = "SCALAR_FUNCTION"
	KindTableFunction     ObjectKind = "TABLE_FUNCTION"
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
}

// ColumnDef is a view column list entry.
type ColumnDef struct {
	Name        string
	Description string
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
