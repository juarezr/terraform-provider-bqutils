package sqlparse

import (
	"testing"
)

func TestParseRemoteFunction1(t *testing.T) {
	sql := "CREATE FUNCTION `myproject.mydataset`.encrypt(x BYTES)\nRETURNS BYTES\nREMOTE WITH CONNECTION `myproject.us-east4.my_remote_connection`\nOPTIONS (\n  endpoint = 'https://us-east4-my_project.cloudfunctions.net/encript',\n  user_defined_context = [(\"mode\", \"encryption\")]\n)\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindScalarFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.ObjectID != "encrypt" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	rfo := res.RemoteFunctionOptions
	if rfo == nil {
		t.Fatal("expected remote_function_options")
	}
	if rfo.Connection != "myproject.us-east4.my_remote_connection" {
		t.Fatalf("connection=%q", rfo.Connection)
	}
	if rfo.Endpoint != "https://us-east4-my_project.cloudfunctions.net/encript" {
		t.Fatalf("endpoint=%q", rfo.Endpoint)
	}
	if rfo.UserDefinedContext == nil || rfo.UserDefinedContext["mode"] != "encryption" {
		t.Fatalf("context=%v", rfo.UserDefinedContext)
	}
}

func TestParseRemoteFunction2(t *testing.T) {
	sql := "CREATE OR REPLACE FUNCTION my_dataset.get_coordinates(address STRING) \nRETURNS STRING\nREMOTE WITH CONNECTION `my_project.us.my_geocode_function_connection`\nOPTIONS (\n  endpoint = 'https://us-east4-my_project.cloudfunctions.net/get_coordinates'\n);\n"
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.DatasetID != "my_dataset" || res.ObjectID != "get_coordinates" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	rfo := res.RemoteFunctionOptions
	if rfo == nil {
		t.Fatal("expected remote_function_options")
	}
	if rfo.Connection != "my_project.us.my_geocode_function_connection" {
		t.Fatalf("connection=%q", rfo.Connection)
	}
	if rfo.Endpoint != "https://us-east4-my_project.cloudfunctions.net/get_coordinates" {
		t.Fatalf("endpoint=%q", rfo.Endpoint)
	}
}
