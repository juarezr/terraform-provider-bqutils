package sqlparse

import (
	"strings"
	"testing"
)

func TestParsePythonFunction(t *testing.T) {
	sql := "CREATE FUNCTION `myproject`.`mydataset`.multiplyInputs (x FLOAT64, y FLOAT64) \nRETURNS FLOAT64 \nLANGUAGE python OPTIONS (\n    runtime_version = \"python-3.11\",\n    entry_point = \"multiply\"\n) AS r '''\n\ndef multiply(x, y):\n    return x * y\n\n''';\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindScalarFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.Project != "myproject" || res.DatasetID != "mydataset" || res.ObjectID != "multiplyInputs" {
		t.Fatalf("name=%s.%s.%s", res.Project, res.DatasetID, res.ObjectID)
	}
	if res.Language != "PYTHON" {
		t.Fatalf("lang=%s", res.Language)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "multiply" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
	if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.RuntimeVersion != "python-3.11" {
		t.Fatalf("external_runtime=%+v", res.ExternalRuntimeOptions)
	}
	if !strings.Contains(res.DefinitionBody, "def multiply") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParsePythonFunction2(t *testing.T) {
	sql := "CREATE FUNCTION `myproject.mydataset`.area(radius FLOAT64)\nRETURNS FLOAT64 \nLANGUAGE python\nOPTIONS (\n    entry_point='area_handler', \n    runtime_version='python-3.11', \n    packages=['scipy==1.15.3']\n) AS r\"\"\"\nimport scipy\n\ndef area_handler(radius):\n  return scipy.constants.pi*radius*radius\n\"\"\";\n\n-- SELECT `myproject.mydataset`.area(4.5);\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "area" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "area_handler" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
	if len(res.PythonOptions.Packages) != 1 || res.PythonOptions.Packages[0] != "scipy==1.15.3" {
		t.Fatalf("packages=%v", res.PythonOptions.Packages)
	}
	if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.RuntimeVersion != "python-3.11" {
		t.Fatalf("external_runtime=%+v", res.ExternalRuntimeOptions)
	}
	if !strings.Contains(res.DefinitionBody, "scipy.constants.pi") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParsePythonFunction3(t *testing.T) {
	sql := "CREATE FUNCTION `myproject.mydataset`.myFunc(a FLOAT64, b STRING)\nRETURNS STRING \nLANGUAGE python\nOPTIONS (\nentry_point='compute', runtime_version='python-3.11',\nlibrary=['gs://mybucket/mypath/lib1.py'])\nAS r\"\"\"\nimport path.to.lib1 as lib1\n\ndef compute(a, b):\n  # doInterestingStuff is a function defined in\n  # gs://mybucket/mypath/lib1.py\n  return lib1.doInterestingStuff(a, b);\n\n\"\"\";\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "myFunc" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "compute" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
	if len(res.ImportedLibraries) != 1 || res.ImportedLibraries[0] != "gs://mybucket/mypath/lib1.py" {
		t.Fatalf("libs=%v", res.ImportedLibraries)
	}
	if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.RuntimeVersion != "python-3.11" {
		t.Fatalf("external_runtime=%+v", res.ExternalRuntimeOptions)
	}
}

func TestParsePythonFunction4(t *testing.T) {
	sql := "CREATE FUNCTION `myproject.mydataset`.square_area (length FLOAT64) \nRETURNS FLOAT64 \nLANGUAGE python \nOPTIONS (\n    entry_point = 'square_area',\n    runtime_version = 'python-3.11',\n    container_memory = '2Gi',\n    container_cpu = 1,\n    container_request_concurrency = 4\n) AS r \"\"\"\ndef square_area(length):\n  return length*length\n\"\"\";\n\nSELECT\n    `PROJECT_ID.DATASET_ID`.square_area (4.5);"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "square_area" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "square_area" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
	ero := res.ExternalRuntimeOptions
	if ero == nil {
		t.Fatal("expected external_runtime_options")
	}
	if ero.RuntimeVersion != "python-3.11" || ero.ContainerMemory != "2Gi" || ero.ContainerCPU != "1" || ero.ContainerRequestConcurrency != "4" {
		t.Fatalf("external_runtime=%+v", ero)
	}
	if !strings.Contains(res.DefinitionBody, "def square_area") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParsePythonFunction5(t *testing.T) {
	sql := "CREATE FUNCTION `myproject.mydataset`.multiplyVectorizedArrow(x FLOAT64, y FLOAT64)\nRETURNS FLOAT64\nLANGUAGE python\nOPTIONS(\n  runtime_version=\"python-3.11\",\n  entry_point=\"vectorized_multiply_arrow\"\n)\nAS r'''\nimport pyarrow as pa\nimport pyarrow.compute as pc\n\ndef vectorized_multiply_arrow(batch: pa.RecordBatch):\n    # Access columns directly from the Arrow RecordBatch\n    x = batch.column('x')\n    y = batch.column('y')\n\n    # Use pyarrow.compute for vectorized operations\n    return pc.multiply(x, y)\n''';\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "multiplyVectorizedArrow" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "vectorized_multiply_arrow" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
	if res.ExternalRuntimeOptions == nil || res.ExternalRuntimeOptions.RuntimeVersion != "python-3.11" {
		t.Fatalf("external_runtime=%+v", res.ExternalRuntimeOptions)
	}
	if !strings.Contains(res.DefinitionBody, "pyarrow.compute") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParsePythonFunctionConnection(t *testing.T) {
	sql := "CREATE FUNCTION `myproject.mydataset`.translate_to_es(x STRING)\nRETURNS STRING \nLANGUAGE python\nWITH CONNECTION `myproject.us-east4.translation_service`\nOPTIONS (\n    entry_point='do_translate',\n    runtime_version='python-3.11',\n    packages=['google-cloud-translate>=3.11', 'google-api-core']\n) AS r\"\"\"\n\nfrom google.api_core.retry import Retry\nfrom google.cloud import translate\n\nproject = \"my_translate_project\"\ntranslate_client = translate.TranslationServiceClient()\n\ndef do_translate(x : str) -> str:\n\n    response = translate_client.translate_text(\n        request={\n            \"parent\": f\"projects/myproject/locations/us-central1\",\n            \"contents\": [x],\n            \"target_language_code\": \"es\",\n            \"mime_type\": \"text/plain\",\n        },\n        retry=Retry(),\n    )\n    return response.translations[0].translated_text\n\n\"\"\";\n\n-- -- Call the UDF.\n-- WITH text_table AS\n--   (SELECT \"Hello\" AS text\n--   UNION ALL\n--   SELECT \"Good morning\" AS text\n--   UNION ALL\n--   SELECT \"Goodbye\" AS text)\n-- SELECT text,\n-- `myproject.mydataset`.translate_to_es(text) AS translated_text\n-- FROM text_table;\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "translate_to_es" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if res.PythonOptions == nil || res.PythonOptions.EntryPoint != "do_translate" {
		t.Fatalf("python_options=%+v", res.PythonOptions)
	}
	if len(res.PythonOptions.Packages) != 2 {
		t.Fatalf("packages=%v", res.PythonOptions.Packages)
	}
	ero := res.ExternalRuntimeOptions
	if ero == nil || ero.RuntimeConnection != "myproject.us-east4.translation_service" {
		t.Fatalf("external_runtime=%+v", ero)
	}
	if ero.RuntimeVersion != "python-3.11" {
		t.Fatalf("runtime_version=%q", ero.RuntimeVersion)
	}
	if !strings.Contains(res.DefinitionBody, "TranslationServiceClient") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}
