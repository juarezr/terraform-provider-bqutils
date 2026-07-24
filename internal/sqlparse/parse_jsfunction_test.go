package sqlparse

import (
	"strings"
	"testing"
)

func TestParseJSFunction1(t *testing.T) {
	sql := `CREATE FUNCTION mydataset.SumFieldsNamedFoo(json_row STRING)
RETURNS FLOAT64
LANGUAGE js
AS r"""
  function SumFoo(obj) {
    var sum = 0;
    for (var field in obj) {
      if (obj.hasOwnProperty(field) && obj[field] != null) {
        if (typeof obj[field] == "object") {
          sum += SumFoo(obj[field]);
        } else if (field == "foo") {
          sum += obj[field];
        }
      }
    }
    return sum;
  }
  var row = JSON.parse(json_row);
  return SumFoo(row);
""";

`
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindScalarFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.DatasetID != "mydataset" || res.ObjectID != "SumFieldsNamedFoo" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	if res.Language != "JAVASCRIPT" {
		t.Fatalf("lang=%s", res.Language)
	}
	if !strings.Contains(res.DefinitionBody, "SumFoo") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseJSFunction2(t *testing.T) {
	sql := `CREATE FUNCTION mydataset.customGreeting(a STRING)
RETURNS STRING
LANGUAGE js
AS r"""
  var d = new Date();
  if (d.getHours() < 12) {
    return 'Good Morning, ' + a + '!';
  } else {
    return 'Good Evening, ' + a + '!';
  }
""";
`
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "customGreeting" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if res.Language != "JAVASCRIPT" {
		t.Fatalf("lang=%s", res.Language)
	}
	if !strings.Contains(res.DefinitionBody, "Good Morning") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseJSFunction3(t *testing.T) {
	sql := `CREATE FUNCTION mydataset.myFunc(a FLOAT64, b STRING)
RETURNS STRING
LANGUAGE js
  OPTIONS (
    library=['gs://my-bucket/path/to/lib1.js', 'gs://my-bucket/path/to/lib2.js'])
AS r"""
  // Assumes 'doInterestingStuff' is defined in one of the library files.
  return doInterestingStuff(a, b);
""";
`
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "myFunc" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if len(res.ImportedLibraries) != 2 {
		t.Fatalf("libs=%v", res.ImportedLibraries)
	}
	if res.ImportedLibraries[0] != "gs://my-bucket/path/to/lib1.js" {
		t.Fatalf("lib0=%q", res.ImportedLibraries[0])
	}
	if !strings.Contains(res.DefinitionBody, "doInterestingStuff") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseJSFunction4(t *testing.T) {
	sql := `CREATE OR REPLACE AGGREGATE FUNCTION myproject.mydataset.SumPositive(x FLOAT64)
RETURNS FLOAT64
LANGUAGE js
AS r'''

  export function initialState() {
    return {sum: 0}
  }
  export function aggregate(state, x) {
    if (x > 0) {
      state.sum += x;
    }
  }
  export function merge(state, partialState) {
    state.sum += partialState.sum;
  }
  export function finalize(state) {
    return state.sum;
  }

''';`
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindAggregateFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.Project != "myproject" || res.DatasetID != "mydataset" || res.ObjectID != "SumPositive" {
		t.Fatalf("name=%s.%s.%s", res.Project, res.DatasetID, res.ObjectID)
	}
	if res.Language != "JAVASCRIPT" {
		t.Fatalf("lang=%s", res.Language)
	}
	if !strings.Contains(res.DefinitionBody, "initialState") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseJSFunction5Temp(t *testing.T) {
	sql := `CREATE TEMP AGGREGATE FUNCTION SumPositive(x FLOAT64)
RETURNS FLOAT64
LANGUAGE js
AS r'''

  export function initialState() {
    return {sum: 0}
  }
  export function aggregate(state, x) {
    if (x > 0) {
      state.sum += x;
    }
  }
  export function merge(state, partialState) {
    state.sum += partialState.sum;
  }
  export function finalize(state) {
    return state.sum;
  }

''';
`
	_, err := ParseRoutine(sql, Options{TrimBody: true})
	if err == nil {
		t.Fatal("expected TEMP error")
	}
	if !strings.Contains(err.Error(), "TEMP") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseJSFunction6Temp(t *testing.T) {
	sql := `CREATE TEMP AGGREGATE FUNCTION mydataset.JsStringAgg
(
  s STRING,
  delimiter STRING NOT AGGREGATE
)
RETURNS STRING
LANGUAGE js
AS r'''

  export function initialState() {
    return {strings: []}
  }
  export function aggregate(state, s) {
    state.strings.push(s);
  }
  export function merge(state, partialState) {
    state.strings = state.strings.concat(partialState.strings);
  }
  export function finalize(state, delimiter) {
    return state.strings.join(delimiter);
  }

''';`
	_, err := ParseRoutine(sql, Options{TrimBody: true})
	if err == nil {
		t.Fatal("expected TEMP error")
	}
	if !strings.Contains(err.Error(), "TEMP") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseJSFunction7Temp(t *testing.T) {
	sql := `CREATE TEMP AGGREGATE FUNCTION SumOfPrimes(x INT64)
RETURNS INT64
LANGUAGE js
AS r'''

  var primes = new Set([2]);
  var maxTested = 2;

  function isPrime(n) {
    if (primes.has(n)) {
      return true;
    }
    if (n <= maxTested) {
      return false;
    }
    for (var k = 2; k < n; ++k) {
      if (!isPrime(k)) {
        continue;
      }
      if ((n % k) == 0) {
        maxTested = n;
        return false;
      }
    }
    maxTested = n;
    primes.add(n);
    return true;
  }

  export function initialState() {
    return {sum: 0};
  }

  export function aggregate(state, x) {
    x = Number(x);
    if (isPrime(x)) {
      state.sum += x;
    }
  }

  export function merge(state, partialState) {
    state.sum += partialState.sum;
  }

  export function finalize(state) {
    return state.sum;
  }

''';
`
	_, err := ParseRoutine(sql, Options{TrimBody: true})
	if err == nil {
		t.Fatal("expected TEMP error")
	}
	if !strings.Contains(err.Error(), "TEMP") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseJSFunction8(t *testing.T) {
	sql := `CREATE AGGREGATE FUNCTION mydataset.JsAggFn(x FLOAT64)
RETURNS FLOAT64
LANGUAGE js
OPTIONS (library = ['gs://foo/bar.js'])
AS r'''

  import doInterestingStuff from 'bar.js';

  export function initialState() {
    return ...
  }
  export function aggregate(state, x) {
    var result = doInterestingStuff(x);
    ...
  }
  export function merge(state, partial_state) {
    ...
  }
  export function finalize(state) {
    return ...;
  }

''';`
	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindAggregateFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.ObjectID != "JsAggFn" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if len(res.ImportedLibraries) != 1 || res.ImportedLibraries[0] != "gs://foo/bar.js" {
		t.Fatalf("libs=%v", res.ImportedLibraries)
	}
	if !strings.Contains(res.DefinitionBody, "doInterestingStuff") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}
