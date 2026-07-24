package sqlparse

import (
	"strings"
	"testing"
)

func TestParseSparkProc1(t *testing.T) {
	sql := "CREATE PROCEDURE my_bq_project.my_dataset.spark_proc1 ()\nWITH\nCONNECTION `my-project-id.us.my-connection` \nOPTIONS (\n    engine = \"SPARK\",\n    runtime_version = \"2.2\",\n    main_file_uri = \"gs://my-bucket/my-pyspark-main.py\"\n) \nLANGUAGE PYTHON"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindProcedure {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.Project != "my_bq_project" || res.DatasetID != "my_dataset" || res.ObjectID != "spark_proc1" {
		t.Fatalf("name=%s.%s.%s", res.Project, res.DatasetID, res.ObjectID)
	}
	if res.Language != "PYTHON" {
		t.Fatalf("lang=%s", res.Language)
	}
	so := res.SparkOptions
	if so == nil {
		t.Fatal("expected spark_options")
	}
	if so.Connection != "my-project-id.us.my-connection" {
		t.Fatalf("connection=%q", so.Connection)
	}
	if so.RuntimeVersion != "2.2" {
		t.Fatalf("runtime_version=%q", so.RuntimeVersion)
	}
	if so.MainFileURI != "gs://my-bucket/my-pyspark-main.py" {
		t.Fatalf("main_file_uri=%q", so.MainFileURI)
	}
}

func TestParseSparkProc2(t *testing.T) {
	sql := "CREATE OR REPLACE PROCEDURE my_bq_project.my_dataset.spark_proc()\nWITH CONNECTION `my-project-id.us.my-connection`\nOPTIONS(engine=\"SPARK\", runtime_version=\"2.2\")\nLANGUAGE PYTHON AS R\"\"\"\nfrom pyspark.sql import SparkSession\n\nspark = SparkSession.builder.appName(\"spark-bigquery-demo\").getOrCreate()\n\n# Load data from BigQuery.\nwords = spark.read.format(\"bigquery\") \\\n  .option(\"table\", \"bigquery-public-data:samples.shakespeare\") \\\n  .load()\nwords.createOrReplaceTempView(\"words\")\n\n# Perform word count.\nword_count = words.select('word', 'word_count').groupBy('word').sum('word_count').withColumnRenamed(\"sum(word_count)\", \"sum_word_count\")\nword_count.show()\nword_count.printSchema()\n\n# Saving the data to BigQuery\nword_count.write.format(\"bigquery\") \\\n  .option(\"writeMethod\", \"direct\") \\\n  .save(\"wordcount_dataset.wordcount_output\")\n\"\"\"\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "spark_proc" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	so := res.SparkOptions
	if so == nil || so.Connection != "my-project-id.us.my-connection" || so.RuntimeVersion != "2.2" {
		t.Fatalf("spark_options=%+v", so)
	}
	if !strings.Contains(res.DefinitionBody, "SparkSession") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseSparkProc3(t *testing.T) {
	sql := "CREATE OR REPLACE PROCEDURE my_bq_project.my_dataset.spark_proc(num INT64)\nWITH CONNECTION `my-project-id.us.my-connection`\nOPTIONS(engine=\"SPARK\", runtime_version=\"2.2\")\nLANGUAGE PYTHON AS R\"\"\"\nfrom pyspark.sql import SparkSession\nimport os\nimport json\n\nspark = SparkSession.builder.appName(\"spark-bigquery-demo\").getOrCreate()\nsc = spark.sparkContext\n\n# Get the input parameter num in JSON string and convert to a Python variable\nnum = int(json.loads(os.environ[\"BIGQUERY_PROC_PARAM.num\"]))\n\n\"\"\"\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Arguments) != 1 || res.Arguments[0].Name != "num" {
		t.Fatalf("args=%+v", res.Arguments)
	}
	so := res.SparkOptions
	if so == nil || so.RuntimeVersion != "2.2" {
		t.Fatalf("spark_options=%+v", so)
	}
	if !strings.Contains(res.DefinitionBody, "BIGQUERY_PROC_PARAM.num") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseSparkProc4(t *testing.T) {
	sql := "CREATE OR REPLACE PROCEDURE my_bq_project.my_dataset.spark_proc(num INT64, info ARRAY<STRUCT<a INT64, b STRING>>)\nWITH CONNECTION `my-project-id.us.my-connection`\nOPTIONS(engine=\"SPARK\", runtime_version=\"2.2\")\nLANGUAGE PYTHON AS R\"\"\"\nfrom pyspark.sql import SparkSession\nfrom bigquery.spark.procedure import SparkProcParamContext\n\ndef check_in_param(x, num):\n  return x['a'] + num\n\ndef main():\n  spark = SparkSession.builder.appName(\"spark-bigquery-demo\").getOrCreate()\n  sc=spark.sparkContext\n  spark_proc_param_context = SparkProcParamContext.getOrCreate(spark)\n\n  # Get the input parameter num of type INT64\n  num = spark_proc_param_context.num\n\n  # Get the input parameter info of type ARRAY<STRUCT<a INT64, b STRING>>\n  info = spark_proc_param_context.info\n\n  # Pass the parameter to executors\n  df = sc.parallelize(info)\n  value = df.map(lambda x : check_in_param(x, num)).sum()\n\nmain()\n\"\"\"\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Arguments) != 2 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[1].Name != "info" || !strings.Contains(res.Arguments[1].DataTypeJSON, "ARRAY") {
		t.Fatalf("args=%+v", res.Arguments)
	}
	so := res.SparkOptions
	if so == nil || so.Connection == "" {
		t.Fatalf("spark_options=%+v", so)
	}
	if !strings.Contains(res.DefinitionBody, "SparkProcParamContext") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseSparkProc5(t *testing.T) {
	sql := "CREATE OR REPLACE PROCEDURE my_bq_project.my_dataset.pyspark_proc\n(\n    IN int INT64,\n    INOUT datetime DATETIME,\n    OUT b BOOL,\n    OUT info ARRAY<STRUCT<a INT64, b STRING>>,\n    OUT time TIME,\n    OUT f FLOAT64,\n    OUT bs BYTES,\n    OUT date DATE,\n    OUT ts TIMESTAMP,\n    OUT js JSON\n)\nWITH CONNECTION `my_bq_project.my_dataset.my_connection`\nOPTIONS(engine=\"SPARK\", runtime_version=\"2.2\") \nLANGUAGE PYTHON AS\nR\"\"\"\nfrom pyspark.sql.session import SparkSession\nimport datetime\nfrom bigquery.spark.procedure import SparkProcParamContext\n\nspark = SparkSession.builder.appName(\"bigquery-pyspark-demo\").getOrCreate()\nspark_proc_param_context = SparkProcParamContext.getOrCreate(spark)\n\n# Reading the IN and INOUT parameter values.\nint = spark_proc_param_context.int\ndt = spark_proc_param_context.datetime\nprint(\"IN parameter value: \", int, \", INOUT parameter value: \", dt)\n\n# Returning the value of the OUT and INOUT parameters.\nspark_proc_param_context.datetime = datetime.datetime(1970, 1, 1, 0, 20, 0, 2, tzinfo=datetime.timezone.utc)\nspark_proc_param_context.b = True\nspark_proc_param_context.info = [{\"a\":2, \"b\":\"dd\"}, {\"a\":2, \"b\":\"dd\"}]\nspark_proc_param_context.time = datetime.time(23, 20, 50, 520000)\nspark_proc_param_context.f = 20.23\nspark_proc_param_context.bs = b\"hello\"\nspark_proc_param_context.date = datetime.date(1985, 4, 12)\nspark_proc_param_context.ts = datetime.datetime(1970, 1, 1, 0, 20, 0, 2, tzinfo=datetime.timezone.utc)\nspark_proc_param_context.js = {\"name\": \"Alice\", \"age\": 30}\n\"\"\";"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "pyspark_proc" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if len(res.Arguments) != 10 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	modes := map[string]string{}
	for _, a := range res.Arguments {
		modes[a.Name] = a.Mode
	}
	if modes["int"] != "IN" || modes["datetime"] != "INOUT" || modes["b"] != "OUT" {
		t.Fatalf("modes=%v", modes)
	}
	so := res.SparkOptions
	if so == nil || so.Connection != "my_bq_project.my_dataset.my_connection" || so.RuntimeVersion != "2.2" {
		t.Fatalf("spark_options=%+v", so)
	}
	if !strings.Contains(res.DefinitionBody, "spark_proc_param_context") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}
