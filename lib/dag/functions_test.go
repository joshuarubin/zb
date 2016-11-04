package dag

import (
	"testing"
)

// RandMinimumCut has been tested in practice (Coursera Algo course 1). If any bugs crop up, email me.

func BenchmarkTopologicalSort(b *testing.B) {
	b.StopTimer()
	graph, _ := setupTopologicalSort()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		graph.TopologicalSort()
	}
}

func TestTopologicalSort(t *testing.T) {
	graph, wantOrder := setupTopologicalSort()
	result := graph.TopologicalSort()
	firstLen := len(result)
	result = graph.TopologicalSort()
	if len(result) != firstLen {
		t.Errorf("topologicalSort 2 times fails")
	}
	for i := range result {
		if result[i] != wantOrder[i] {
			t.Errorf("index %v in result != wanted, value: %v, want value: %v", i, result[i], wantOrder[i])
		}
	}
}

func setupTopologicalSort() (*Graph, []Node) {
	graph := &Graph{}
	var nodes []Node
	// create graph on page 613 of CLRS ed. 3
	nodes = append(nodes, graph.MakeNode()) // shirt
	nodes = append(nodes, graph.MakeNode()) // tie
	nodes = append(nodes, graph.MakeNode()) // jacket
	nodes = append(nodes, graph.MakeNode()) // belt
	nodes = append(nodes, graph.MakeNode()) // watch
	nodes = append(nodes, graph.MakeNode()) // undershorts
	nodes = append(nodes, graph.MakeNode()) // pants
	nodes = append(nodes, graph.MakeNode()) // shoes
	nodes = append(nodes, graph.MakeNode()) // socks
	graph.MakeEdge(nodes[0], nodes[1])
	graph.MakeEdge(nodes[1], nodes[2])
	graph.MakeEdge(nodes[0], nodes[3])
	graph.MakeEdge(nodes[3], nodes[2])
	graph.MakeEdge(nodes[5], nodes[6])
	graph.MakeEdge(nodes[5], nodes[7])
	graph.MakeEdge(nodes[6], nodes[3])
	graph.MakeEdge(nodes[6], nodes[7])
	graph.MakeEdge(nodes[8], nodes[7])
	wantOrder := make([]Node, len(graph.nodes))
	wantOrder[0] = nodes[8] // socks
	wantOrder[1] = nodes[5] // undershorts
	wantOrder[2] = nodes[6] // pants
	wantOrder[3] = nodes[7] // shoes
	wantOrder[4] = nodes[4] // watch
	wantOrder[5] = nodes[0] // shirt
	wantOrder[6] = nodes[3] // belt
	wantOrder[7] = nodes[1] // tie
	wantOrder[8] = nodes[2] // jacket
	return graph, wantOrder
}
