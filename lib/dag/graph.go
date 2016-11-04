package dag

import "github.com/pkg/errors"

// NOTE: started from github.com/twmb/algoimpl/go/graph

// Graph is an adjacency slice representation of a directed graph.
type Graph struct {
	nodes []*node
}

type node struct {
	edges         []edge
	reversedEdges []edge
	index         int
	state         int   // used for metadata
	parent        *node // also used for metadata
	container     Node  // who holds me
}

// Node connects to a backing node on the graph. It can safely be used in maps.
type Node struct {
	// In an effort to prevent access to the actual graph
	// and so that the Node type can be used in a map while
	// the graph changes metadata, the Node type encapsulates
	// a pointer to the actual node data.
	node *node

	// Value can be used to store information on the caller side.
	// Its use is optional. See the Topological Sort example for
	// a reason on why to use this pointer.
	// The reason it is a pointer is so that graph function calls
	// can test for equality on Nodes. The pointer wont change,
	// the value it points to will. If the pointer is explicitly changed,
	// graph functions that use Nodes will cease to work.
	Value *interface{}
}

type edge struct {
	weight int
	end    *node
}

// MakeNode creates a node, adds it to the graph and returns the new node.
func (g *Graph) MakeNode(value interface{}) Node {
	newNode := &node{index: len(g.nodes)}
	newNode.container = Node{node: newNode, Value: &value}
	g.nodes = append(g.nodes, newNode)
	return newNode.container
}

// MakeEdge calls MakeEdgeWeight with a weight of 0 and returns an error if either of the nodes do not
// belong in the graph. Calling MakeEdge multiple times on the same nodes will not create multiple edges.
func (g *Graph) MakeEdge(from, to Node) error {
	return g.MakeEdgeWeight(from, to, 0)
}

// MakeEdgeWeight creates  an edge in the graph with a corresponding weight.
// It returns an error if either of the nodes do not belong in the graph.
//
// Calling MakeEdgeWeight multiple times on the same nodes will not create multiple edges;
// this function will update the weight on the node to the new value.
func (g *Graph) MakeEdgeWeight(from, to Node, weight int) error {
	if from.node == nil || from.node.index >= len(g.nodes) || g.nodes[from.node.index] != from.node {
		return errors.New("First node in MakeEdge call does not belong to this graph")
	}
	if to.node == nil || to.node.index >= len(g.nodes) || g.nodes[to.node.index] != to.node {
		return errors.New("Second node in MakeEdge call does not belong to this graph")
	}

	for i := range from.node.edges { // check if edge already exists
		if from.node.edges[i].end == to.node {
			from.node.edges[i].weight = weight
			return nil
		}
	}
	newEdge := edge{weight: weight, end: to.node}
	from.node.edges = append(from.node.edges, newEdge)
	reversedEdge := edge{weight: weight, end: from.node} // weight for undirected graph only
	to.node.reversedEdges = append(to.node.reversedEdges, reversedEdge)
	return nil
}
