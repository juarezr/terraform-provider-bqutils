package sqlparse

import (
	"strings"
	"testing"
)

func TestParsePython_optionsBeforeLanguage_maxBatchingRows(t *testing.T) {
	// OPTIONS before LANGUAGE: max_batching_rows must not create remote_function_options.
	sql := `
CREATE FUNCTION myproject.mydataset.batch_fn(x FLOAT64)
RETURNS FLOAT64
OPTIONS (
  max_batching_rows = 10,
  entry_point = 'batch_fn',
  runtime_version = 'python-3.11'
)
LANGUAGE python
AS r'''
def batch_fn(x):
  return x
''';`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.RemoteFunctionOptions != nil {
		t.Fatalf("expected no remote_function_options, got %+v", res.RemoteFunctionOptions)
	}
	if res.SparkOptions != nil {
		t.Fatalf("expected no spark_options, got %+v", res.SparkOptions)
	}
	if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.MaxBatchingRows != "10" {
		t.Fatalf("external_runtime=%+v", res.ExternalRuntimeOptions)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "batch_fn" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
}

func TestParsePython_withConnectionBeforeLanguage_noSpark(t *testing.T) {
	sql := `
CREATE FUNCTION myproject.mydataset.translate(x STRING)
RETURNS STRING
WITH CONNECTION myproject.us.my_conn
LANGUAGE python
OPTIONS (
  entry_point='do_translate',
  runtime_version='python-3.11'
) AS r"""
def do_translate(x):
  return x
""";`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.SparkOptions != nil {
		t.Fatalf("spark_options should be nil for Python UDF, got %+v", res.SparkOptions)
	}
	if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.RuntimeConnection != "myproject.us.my_conn" {
		t.Fatalf("external_runtime=%+v", res.ExternalRuntimeOptions)
	}
}

func TestParseProcedure_beginBodyWithEndInString(t *testing.T) {
	sql := `
CREATE OR REPLACE PROCEDURE mydataset.demo(name STRING)
BEGIN
  DECLARE msg STRING;
  SET msg = 'END';
  SELECT msg, name;
END`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.DefinitionBody, "SET msg = 'END'") {
		t.Fatalf("body truncated: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "SELECT msg, name") {
		t.Fatalf("body missing SELECT: %q", res.DefinitionBody)
	}
}

func TestParseProcedure_beginBodyWithEndInBlockComment(t *testing.T) {
	sql := `
CREATE OR REPLACE PROCEDURE mydataset.demo()
BEGIN
  /* call END only as comment */
  SELECT 1;
END`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.DefinitionBody, "SELECT 1") {
		t.Fatalf("body truncated: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "/* call END only as comment */") {
		t.Fatalf("expected comment preserved in body: %q", res.DefinitionBody)
	}
}

func TestApplyOptionsMap_maxBatchingIndependentOfMapOrder(t *testing.T) {
	// Simulate unordered map keys: routing must use entry_point from the same OPTIONS block.
	for i := 0; i < 50; i++ {
		res := &ParseResult{Kind: KindScalarFunction}
		m := map[string]string{
			"max_batching_rows": "5",
			"entry_point":       "handler",
			"runtime_version":   "python-3.11",
		}
		applyOptionsMap(res, m)
		if res.RemoteFunctionOptions != nil {
			t.Fatalf("iter %d: remote set: %+v", i, res.RemoteFunctionOptions)
		}
		if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.MaxBatchingRows != "5" {
			t.Fatalf("iter %d: external=%+v", i, res.ExternalRuntimeOptions)
		}
	}
}
