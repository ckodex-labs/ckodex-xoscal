package graph

import (
	"context"
	"sort"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GraphServer) GetEdge(ctx context.Context, req *servicesv1.GetEdgeRequest) (*servicesv1.GetEdgeResponse, error) {
	e, err := s.store.GetEdge(ctx, req.GetEdgeId())
	if err != nil {
		return nil, mapStoreError("get edge", req.GetEdgeId(), err)
	}
	return &servicesv1.GetEdgeResponse{Edge: edgeToProto(e)}, nil
}

// ListEdges returns edges matching optional filters.
func (s *GraphServer) ListEdges(ctx context.Context, req *servicesv1.ListEdgesRequest) (*servicesv1.ListEdgesResponse, error) {
	pageSize := int(req.GetPageSize())
	if pageSize < 0 || pageSize > maxGraphListPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be between 0 and %d", maxGraphListPageSize)
	}
	var validAfter time.Time
	if req.GetValidAfter() != nil {
		validAfter = req.GetValidAfter().AsTime()
	}
	edges, nextToken, err := s.store.ListEdges(ctx, req.GetFromNode(), req.GetToNode(),
		req.GetRelation(), req.GetTrustState(), validAfter, pageSize, req.GetPageToken())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid page token") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "list edges: %v", err)
	}
	var out []*servicesv1.GraphEdge
	for _, e := range edges {
		out = append(out, edgeToProto(e))
	}
	return &servicesv1.ListEdgesResponse{Edges: out, NextPageToken: nextToken}, nil
}

// DeleteEdge removes a graph edge.
func (s *GraphServer) DeleteEdge(ctx context.Context, req *servicesv1.DeleteEdgeRequest) (*servicesv1.DeleteEdgeResponse, error) {
	if err := s.store.DeleteEdge(ctx, req.GetEdgeId()); err != nil {
		return nil, mapStoreError("delete edge", req.GetEdgeId(), err)
	}
	return &servicesv1.DeleteEdgeResponse{Success: true}, nil
}

// GetNode retrieves a graph node by ID.
func (s *GraphServer) GetNode(ctx context.Context, req *servicesv1.GetNodeRequest) (*servicesv1.GetNodeResponse, error) {
	n, err := s.store.GetNode(ctx, req.GetNodeId())
	if err != nil {
		return nil, mapStoreError("get node", req.GetNodeId(), err)
	}
	return &servicesv1.GetNodeResponse{Node: nodeToProto(n)}, nil
}

// ListNodes returns nodes matching optional filters.
func (s *GraphServer) ListNodes(ctx context.Context, req *servicesv1.ListNodesRequest) (*servicesv1.ListNodesResponse, error) {
	pageSize := int(req.GetPageSize())
	if pageSize < 0 || pageSize > maxGraphListPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be between 0 and %d", maxGraphListPageSize)
	}
	var createdAfter time.Time
	if req.GetCreatedAfter() != nil {
		createdAfter = req.GetCreatedAfter().AsTime()
	}
	nodes, nextToken, err := s.store.ListNodes(ctx, req.GetKind(), req.GetLabelFilter(), createdAfter, pageSize, req.GetPageToken())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid page token") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "list nodes: %v", err)
	}
	var out []*servicesv1.GraphNode
	for _, n := range nodes {
		out = append(out, nodeToProto(n))
	}
	return &servicesv1.ListNodesResponse{Nodes: out, NextPageToken: nextToken}, nil
}

// Traverse performs BFS or DFS from a start node.
func (s *GraphServer) Traverse(ctx context.Context, req *servicesv1.TraverseRequest) (*servicesv1.TraverseResponse, error) {
	if req.GetStartNode() == "" {
		return nil, status.Error(codes.InvalidArgument, "start_node is required")
	}
	if _, err := s.store.GetNode(ctx, req.GetStartNode()); err != nil {
		return nil, mapStoreError("get start node", req.GetStartNode(), err)
	}
	mode := req.GetTraversalMode()
	if mode == "" {
		mode = "bfs"
	}
	maxDepth := int(req.GetMaxDepth())
	if maxDepth <= 0 {
		maxDepth = 10
	}

	relations := req.GetRelations()
	minTrust := req.GetMinTrustState()

	var result []*servicesv1.PathSegment
	visited := make(map[string]bool)
	queue := []struct {
		nodeID string
		depth  int
	}{
		{nodeID: req.GetStartNode(), depth: 0},
	}
	visited[req.GetStartNode()] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		edges, err := s.store.ListEdgesFrom(ctx, current.nodeID, relations, minTrust, 100)
		if err != nil {
			return nil, mapGraphQueryError("list edges", err)
		}

		for _, e := range edges {
			if visited[e.ToNode] {
				continue
			}
			visited[e.ToNode] = true

			toNode, err := s.store.GetNode(ctx, e.ToNode)
			if err != nil {
				continue
			}

			result = append(result, &servicesv1.PathSegment{
				Depth: boundedInt32(current.depth + 1),
				Edge:  edgeToProto(e),
				Node:  nodeToProto(toNode),
			})

			if mode == "bfs" {
				queue = append(queue, struct {
					nodeID string
					depth  int
				}{
					nodeID: e.ToNode, depth: current.depth + 1,
				})
			} else {
				queue = append([]struct {
					nodeID string
					depth  int
				}{
					{nodeID: e.ToNode, depth: current.depth + 1},
				}, queue...)
			}
		}
	}

	return &servicesv1.TraverseResponse{Path: result}, nil
}

// ShortestPath finds shortest path between two nodes using BFS with trust filtering.
func (s *GraphServer) ShortestPath(ctx context.Context, req *servicesv1.ShortestPathRequest) (*servicesv1.ShortestPathResponse, error) {
	start := req.GetFromNode()
	goal := req.GetToNode()
	if start == "" || goal == "" {
		return nil, status.Error(codes.InvalidArgument, "from_node and to_node are required")
	}

	if start == goal {
		return &servicesv1.ShortestPathResponse{Found: true, TotalWeight: 0}, nil
	}

	type nodeInfo struct {
		prev  string
		edge  *Edge
		depth int
	}

	visited := make(map[string]*nodeInfo)
	queue := []string{start}
	visited[start] = &nodeInfo{depth: 0}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		info := visited[current]

		edges, err := s.store.ListEdgesFrom(ctx, current, req.GetAllowedRelations(), req.GetMinTrustState(), 100)
		if err != nil {
			return nil, mapGraphQueryError("list edges", err)
		}

		for _, e := range edges {
			if visited[e.ToNode] != nil {
				continue
			}
			visited[e.ToNode] = &nodeInfo{prev: current, edge: e, depth: info.depth + 1}
			if e.ToNode == goal {
				var path []*servicesv1.GraphEdge
				var totalWeight float64
				node := goal
				for node != start {
					info := visited[node]
					path = append([]*servicesv1.GraphEdge{edgeToProto(info.edge)}, path...)
					totalWeight += info.edge.Weight
					node = info.prev
				}
				return &servicesv1.ShortestPathResponse{
					Edges:       path,
					TotalWeight: totalWeight,
					Found:       true,
				}, nil
			}
			queue = append(queue, e.ToNode)
		}
	}

	return &servicesv1.ShortestPathResponse{Found: false}, nil
}

// ImpactRadius finds all nodes reachable from a node within N hops.
func (s *GraphServer) ImpactRadius(ctx context.Context, req *servicesv1.ImpactRadiusRequest) (*servicesv1.ImpactRadiusResponse, error) {
	if req.GetNode() == "" {
		return nil, status.Error(codes.InvalidArgument, "node is required")
	}
	if _, err := s.store.GetNode(ctx, req.GetNode()); err != nil {
		return nil, mapStoreError("get node", req.GetNode(), err)
	}
	maxDepth := int(req.GetMaxDepth())
	if maxDepth <= 0 {
		maxDepth = 3
	}

	visitedNodes := make(map[string]*Node)
	visitedEdges := make(map[string]*Edge)
	queue := []struct {
		nodeID string
		depth  int
	}{{nodeID: req.GetNode(), depth: 0}}
	visitedNodes[req.GetNode()] = &Node{ID: req.GetNode()}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		edges, err := s.store.ListEdgesFrom(ctx, current.nodeID, req.GetRelations(), req.GetMinTrustState(), 100)
		if err != nil {
			return nil, mapGraphQueryError("list edges", err)
		}

		for _, e := range edges {
			if visitedEdges[e.ID] == nil {
				visitedEdges[e.ID] = e
			}
			if visitedNodes[e.ToNode] != nil {
				continue
			}
			toNode, err := s.store.GetNode(ctx, e.ToNode)
			if err != nil {
				continue
			}
			visitedNodes[e.ToNode] = toNode
			queue = append(queue, struct {
				nodeID string
				depth  int
			}{
				nodeID: e.ToNode, depth: current.depth + 1,
			})
		}
	}

	var nodes []*servicesv1.GraphNode
	for _, n := range visitedNodes {
		nodes = append(nodes, nodeToProto(n))
	}
	var edges []*servicesv1.GraphEdge
	for _, e := range visitedEdges {
		edges = append(edges, edgeToProto(e))
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].GetId() < nodes[j].GetId() })
	sort.Slice(edges, func(i, j int) bool { return edges[i].GetId() < edges[j].GetId() })
	return &servicesv1.ImpactRadiusResponse{Nodes: nodes, Edges: edges}, nil
}

// ExplainClaim returns the claim, its source evidence, proof state, and current
// trust state from the same exchange store used by verification.
