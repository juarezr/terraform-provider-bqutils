package deps

import (
	"reflect"
	"strings"
	"testing"
)

func TestComputeLayers_simpleChain(t *testing.T) {
	got, err := ComputeLayers([]SourceNode{
		{
			DatasetID: "d", ObjectID: "b", ObjectType: "SCALAR_FUNCTION",
			References: []EdgeRef{{DatasetID: "d", ObjectID: "a", ObjectType: "SCALAR_FUNCTION"}},
		},
		{
			DatasetID: "d", ObjectID: "a", ObjectType: "SCALAR_FUNCTION",
			References: nil,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := LayerResult{
		MaxLayers: 2,
		Layered: []LayeredNode{
			{Layer: 1, DatasetID: "d", ObjectID: "a", ObjectType: "SCALAR_FUNCTION", ResourceType: "ROUTINE"},
			{Layer: 2, DatasetID: "d", ObjectID: "b", ObjectType: "SCALAR_FUNCTION", ResourceType: "ROUTINE"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
}

func TestComputeLayers_dropUnknownAndInfoSchema(t *testing.T) {
	got, err := ComputeLayers([]SourceNode{
		{
			DatasetID: "d", ObjectID: "v1", ObjectType: "VIEW",
			References: []EdgeRef{
				{DatasetID: "d", ObjectID: "ext_table", ObjectType: "VIEW"},
				{DatasetID: "d", ObjectID: "INFORMATION_SCHEMA.TABLES", ObjectType: "TABLE"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxLayers != 1 || len(got.Layered) != 1 {
		t.Fatalf("got=%#v", got)
	}
	if got.Layered[0].ResourceType != "VIEW" {
		t.Fatalf("resource_type=%s", got.Layered[0].ResourceType)
	}
}

func TestComputeLayers_cycle(t *testing.T) {
	_, err := ComputeLayers([]SourceNode{
		{DatasetID: "d", ObjectID: "a", ObjectType: "SCALAR_FUNCTION", References: []EdgeRef{{DatasetID: "d", ObjectID: "b"}}},
		{DatasetID: "d", ObjectID: "b", ObjectType: "SCALAR_FUNCTION", References: []EdgeRef{{DatasetID: "d", ObjectID: "a"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("err=%v", err)
	}
}

func TestComputeLayers_duplicate(t *testing.T) {
	_, err := ComputeLayers([]SourceNode{
		{DatasetID: "d", ObjectID: "a", ObjectType: "VIEW"},
		{DatasetID: "d", ObjectID: "a", ObjectType: "VIEW"},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err=%v", err)
	}
}

func TestComputeLayers_emptyIDs(t *testing.T) {
	_, err := ComputeLayers([]SourceNode{{DatasetID: "", ObjectID: "a", ObjectType: "VIEW"}})
	if err == nil || !strings.Contains(err.Error(), "dataset_id") {
		t.Fatalf("err=%v", err)
	}
	_, err = ComputeLayers([]SourceNode{{DatasetID: "d", ObjectID: "", ObjectType: "VIEW"}})
	if err == nil || !strings.Contains(err.Error(), "object_id") {
		t.Fatalf("err=%v", err)
	}
}

func TestComputeLayers_empty(t *testing.T) {
	got, err := ComputeLayers(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxLayers != 0 || len(got.Layered) != 0 {
		t.Fatalf("got=%#v", got)
	}
}

func TestComputeLayers_stableSort(t *testing.T) {
	got, err := ComputeLayers([]SourceNode{
		{DatasetID: "d", ObjectID: "z", ObjectType: "SCALAR_FUNCTION"},
		{DatasetID: "d", ObjectID: "a", ObjectType: "SCALAR_FUNCTION"},
		{DatasetID: "c", ObjectID: "m", ObjectType: "VIEW"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{"c.m", "d.a", "d.z"}
	for i, n := range got.Layered {
		id := n.DatasetID + "." + n.ObjectID
		if id != wantIDs[i] || n.Layer != 1 {
			t.Fatalf("i=%d got=%s layer=%d", i, id, n.Layer)
		}
	}
}
