package deps

import (
	"fmt"
	"sort"
	"strings"
)

// SourceNode is one managed object fed into dependency layering.
type SourceNode struct {
	DatasetID  string
	ObjectID   string
	ObjectType string
	References []EdgeRef
}

// EdgeRef is a dependency edge from a source node's body references.
type EdgeRef struct {
	DatasetID  string
	ObjectID   string
	ObjectType string
}

// LayeredNode is one output row of dependency layering.
type LayeredNode struct {
	Layer        int
	DatasetID    string
	ObjectID     string
	ObjectType   string
	ResourceType string
}

// LayerResult is the outcome of ComputeLayers.
type LayerResult struct {
	Layered   []LayeredNode
	MaxLayers int
}

type nodeKey struct {
	dataset, object string
}

func (k nodeKey) String() string {
	return k.dataset + "." + k.object
}

// ComputeLayers assigns topo-sort stages (layers) to managed objects.
// Unknown and INFORMATION_SCHEMA edges are dropped. Cycles and duplicate/empty
// node ids return an error.
func ComputeLayers(sources []SourceNode) (LayerResult, error) {
	nodes := make(map[nodeKey]*SourceNode, len(sources))
	order := make([]nodeKey, 0, len(sources))

	for i, s := range sources {
		if strings.TrimSpace(s.ObjectID) == "" {
			return LayerResult{}, fmt.Errorf("source_references[%d]: object_id is empty", i)
		}
		if strings.TrimSpace(s.DatasetID) == "" {
			return LayerResult{}, fmt.Errorf("source_references[%d]: dataset_id is empty or could not be derived (object_id=%q)", i, s.ObjectID)
		}
		key := nodeKey{dataset: s.DatasetID, object: s.ObjectID}
		if _, exists := nodes[key]; exists {
			return LayerResult{}, fmt.Errorf("duplicate source_references entry for %s", key.String())
		}
		cp := s
		nodes[key] = &cp
		order = append(order, key)
	}

	// indegree[n] = number of managed deps that must be created before n
	indegree := make(map[nodeKey]int, len(nodes))
	// dependents[d] = nodes that depend on d
	dependents := make(map[nodeKey][]nodeKey, len(nodes))
	for _, k := range order {
		indegree[k] = 0
	}

	for _, k := range order {
		n := nodes[k]
		seenEdge := map[nodeKey]struct{}{}
		for _, e := range n.References {
			if isInfoSchemaEdge(e) {
				continue
			}
			ek := nodeKey{dataset: e.DatasetID, object: e.ObjectID}
			if _, ok := nodes[ek]; !ok {
				continue // unknown / external — drop
			}
			if ek == k {
				continue // self-edge
			}
			if _, dup := seenEdge[ek]; dup {
				continue
			}
			seenEdge[ek] = struct{}{}
			dependents[ek] = append(dependents[ek], k)
			indegree[k]++
		}
	}

	var layered []LayeredNode
	remaining := len(nodes)
	layer := 0
	for remaining > 0 {
		var stage []nodeKey
		for _, k := range order {
			if _, ok := nodes[k]; !ok {
				continue
			}
			if indegree[k] == 0 {
				stage = append(stage, k)
			}
		}
		if len(stage) == 0 {
			var cycle []string
			for _, k := range order {
				if _, ok := nodes[k]; ok {
					cycle = append(cycle, k.String())
				}
			}
			sort.Strings(cycle)
			return LayerResult{}, fmt.Errorf("cyclic dependency among: %s", strings.Join(cycle, ", "))
		}
		sort.Slice(stage, func(i, j int) bool {
			if stage[i].dataset != stage[j].dataset {
				return stage[i].dataset < stage[j].dataset
			}
			return stage[i].object < stage[j].object
		})
		layer++
		for _, k := range stage {
			n := nodes[k]
			layered = append(layered, LayeredNode{
				Layer:        layer,
				DatasetID:    n.DatasetID,
				ObjectID:     n.ObjectID,
				ObjectType:   n.ObjectType,
				ResourceType: deriveResourceType(n.ObjectType),
			})
			delete(nodes, k)
			remaining--
			for _, dep := range dependents[k] {
				indegree[dep]--
			}
		}
	}

	if layered == nil {
		layered = []LayeredNode{}
	}
	return LayerResult{Layered: layered, MaxLayers: layer}, nil
}

func isInfoSchemaEdge(e EdgeRef) bool {
	if strings.EqualFold(e.ObjectType, "TABLE") {
		return true
	}
	if strings.Contains(strings.ToUpper(e.ObjectID), "INFORMATION_SCHEMA") {
		return true
	}
	if strings.EqualFold(e.DatasetID, "INFORMATION_SCHEMA") {
		return true
	}
	return false
}

func deriveResourceType(objectType string) string {
	switch strings.ToUpper(objectType) {
	case "VIEW", "MATERIALIZED_VIEW", "TABLE":
		return "VIEW"
	default:
		return "ROUTINE"
	}
}
