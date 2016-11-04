package dag

const (
	// dequeued = ^(1<<31 - 1)
	unseen = 0
	seen   = 1
)

// TopologicalSort topoligically sorts a directed acyclic graph.
// If the graph is cyclic, the sort order will change
// based on which node the sort starts on.
//
// The StronglyConnectedComponents function can be used to determine if a graph has cycles.
func (g *Graph) TopologicalSort() []Node {
	// init states
	for i := range g.nodes {
		g.nodes[i].state = unseen
	}
	sorted := make([]Node, 0, len(g.nodes))
	// sort preorder (first jacket, then shirt)
	for _, node := range g.nodes {
		if node.state == unseen {
			g.dfs(node, &sorted)
		}
	}
	// now make post order for correct sort (jacket follows shirt). O(V)
	length := len(sorted)
	for i := 0; i < length/2; i++ {
		sorted[i], sorted[length-i-1] = sorted[length-i-1], sorted[i]
	}
	return sorted
}

// O(V + E). It does not matter to traverse back
// on a bidirectional edge, because any vertex dfs is
// recursing on is marked as visited and won't be visited
// again anyway.
func (g *Graph) dfs(node *node, finishList *[]Node) {
	node.state = seen
	for _, edge := range node.edges {
		if edge.end.state == unseen {
			edge.end.parent = node
			g.dfs(edge.end, finishList)
		}
	}
	*finishList = append(*finishList, node.container)
}
