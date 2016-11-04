package dag

import (
	"testing"
)

func (g *Graph) reversedEdgeBack(from, to *node) bool {
	for _, v := range to.reversedEdges {
		if v.end == from {
			return true
		}
	}
	return false
}

func (g *Graph) verify(t *testing.T) {
	// over all the nodes
	for i, node := range g.nodes {
		if node.index != i {
			t.Errorf("node's graph index %v != actual graph index %v", node.index, i)
		}
		// over each edge
		for _, edge := range node.edges {

			// check that the graph contains it in the correct position
			if edge.end.index >= len(g.nodes) {
				t.Errorf("adjacent node end graph index %v >= len(g.nodes)%v", edge.end.index, len(g.nodes))
			}
			if g.nodes[edge.end.index] != edge.end {
				t.Errorf("adjacent node %p does not belong to the graph on edge %v: should be %p", edge.end, edge, g.nodes[edge.end.index])
			}
			// if graph is undirected, check that the to node's reversed edges connects to the from edge
			if !g.reversedEdgeBack(node, edge.end) {
				t.Errorf("directed graph: node %v has edge to %v, reversedEdges start at end does not have edge back to node", node, edge.end)
			}
		}
	}
}

func TestMakeNode(t *testing.T) {
	graph := &Graph{}
	nodes := make(map[Node]int, 0)
	for i := 0; i < 10; i++ {
		nodes[graph.MakeNode()] = i
	}
	graph.verify(t)
}

func TestMakeEdge(t *testing.T) {
	graph := &Graph{}
	mapped := make(map[int]Node, 0)
	for i := 0; i < 10; i++ {
		mapped[i] = graph.MakeNode()
	}
	for j := 0; j < 5; j++ {
		for i := 0; i < 10; i++ {
			graph.MakeEdge(mapped[i], mapped[(i+1+j)%10])
		}
	}
	graph.verify(t)
	for i, node := range graph.nodes {
		if mapped[i].node != node {
			t.Errorf("Node at index %v = %v, != %v, wrong!", i, mapped[i], node)
		}
	}
}
