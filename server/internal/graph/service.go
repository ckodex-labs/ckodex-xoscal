package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GraphServer implements TransparencyGraphService.
type GraphServer struct {
	servicesv1.UnimplementedTransparencyGraphServiceServer
	store Store
}

// NewGraphServer creates a new GraphServer.
func NewGraphServer(store Store) *GraphServer {
	return &GraphServer{store: store}
}

// ProjectEdge creates an edge from an existing claim.
func (s *GraphServer) ProjectEdge(ctx context.Context, req *servicesv1.ProjectEdgeRequest) (*servicesv1.ProjectEdgeResponse, error) {
	edgeID := req.GetEdgeId()
	if edgeID == "" {
		edgeID = fmt.Sprintf("edge_%d", time.Now().UnixNano())
	}

	// In a real implementation, we'd fetch the claim and verify it exists.
	// For now, create a placeholder edge with the claim_id.
	e := &Edge{
		ID:         edgeID,
		ClaimID:    req.GetClaimId(),
		TrustState: "candidate",
		Weight:     1.0,
		ValidFrom:  time.Now().UTC(),
	}
	if err := s.store.CreateEdge(ctx, e); err != nil {
		return nil, fmt.Errorf("create edge: %w", err)
	}

	return &servicesv1.ProjectEdgeResponse{Edge: edgeToProto(e)}, nil
}

// GetEdge retrieves a graph edge by ID.
func (s *GraphServer) GetEdge(ctx context.Context, req *servicesv1.GetEdgeRequest) (*servicesv1.GetEdgeResponse, error) {
	e, err := s.store.GetEdge(ctx, req.GetEdgeId())
	if err != nil {
		return nil, fmt.Errorf("get edge: %w", err)
	}
	return &servicesv1.GetEdgeResponse{Edge: edgeToProto(e)}, nil
}

// ListEdges returns edges matching optional filters.
func (s *GraphServer) ListEdges(ctx context.Context, req *servicesv1.ListEdgesRequest) (*servicesv1.ListEdgesResponse, error) {
	var validAfter time.Time
	if req.GetValidAfter() != nil {
		validAfter = req.GetValidAfter().AsTime()
	}
	edges, err := s.store.ListEdges(ctx, req.GetFromNode(), req.GetToNode(),
		req.GetRelation(), req.GetTrustState(), validAfter, int(req.GetPageSize()))
	if err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}
	var out []*servicesv1.GraphEdge
	for _, e := range edges {
		out = append(out, edgeToProto(e))
	}
	return &servicesv1.ListEdgesResponse{Edges: out}, nil
}

// DeleteEdge removes a graph edge.
func (s *GraphServer) DeleteEdge(ctx context.Context, req *servicesv1.DeleteEdgeRequest) (*servicesv1.DeleteEdgeResponse, error) {
	if err := s.store.DeleteEdge(ctx, req.GetEdgeId()); err != nil {
		return nil, fmt.Errorf("delete edge: %w", err)
	}
	return &servicesv1.DeleteEdgeResponse{Success: true}, nil
}

// GetNode retrieves a graph node by ID.
func (s *GraphServer) GetNode(ctx context.Context, req *servicesv1.GetNodeRequest) (*servicesv1.GetNodeResponse, error) {
	n, err := s.store.GetNode(ctx, req.GetNodeId())
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return &servicesv1.GetNodeResponse{Node: nodeToProto(n)}, nil
}

// ListNodes returns nodes matching optional filters.
func (s *GraphServer) ListNodes(ctx context.Context, req *servicesv1.ListNodesRequest) (*servicesv1.ListNodesResponse, error) {
	var createdAfter time.Time
	if req.GetCreatedAfter() != nil {
		createdAfter = req.GetCreatedAfter().AsTime()
	}
	nodes, err := s.store.ListNodes(ctx, req.GetKind(), req.GetLabelFilter(), createdAfter, int(req.GetPageSize()))
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	var out []*servicesv1.GraphNode
	for _, n := range nodes {
		out = append(out, nodeToProto(n))
	}
	return &servicesv1.ListNodesResponse{Nodes: out}, nil
}

// Traverse performs BFS or DFS from a start node.
func (s *GraphServer) Traverse(ctx context.Context, req *servicesv1.TraverseRequest) (*servicesv1.TraverseResponse, error) {
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
			return nil, fmt.Errorf("list edges: %w", err)
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
				Depth: int32(current.depth + 1),
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
			return nil, fmt.Errorf("list edges: %w", err)
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
			return nil, fmt.Errorf("list edges: %w", err)
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
	return &servicesv1.ImpactRadiusResponse{Nodes: nodes, Edges: edges}, nil
}

// ExplainClaim returns full claim details with evidence and trust state (placeholder).
func (s *GraphServer) ExplainClaim(ctx context.Context, req *servicesv1.ExplainClaimRequest) (*servicesv1.ExplainClaimResponse, error) {
	return &servicesv1.ExplainClaimResponse{
		TrustState: "candidate",
	}, nil
}

// ComputeTrustState re-computes trust state for a claim or edge.
func (s *GraphServer) ComputeTrustState(ctx context.Context, req *servicesv1.ComputeTrustStateRequest) (*servicesv1.ComputeTrustStateResponse, error) {
	if req.GetEdgeId() != "" {
		e, err := s.store.GetEdge(ctx, req.GetEdgeId())
		if err != nil {
			return nil, fmt.Errorf("get edge: %w", err)
		}
		ps := parseProofState(e.ProofStateJSON)
		return &servicesv1.ComputeTrustStateResponse{
			EdgeId:     e.ID,
			ProofState: ps,
			TrustState: e.TrustState,
		}, nil
	}
	return &servicesv1.ComputeTrustStateResponse{
		TrustState: "candidate",
	}, nil
}

// VerifyClosure verifies all claims reachable from a subject for a given purpose.
func (s *GraphServer) VerifyClosure(ctx context.Context, req *servicesv1.VerifyClosureRequest) (*servicesv1.VerifyClosureResponse, error) {
	// Simplified: just traverse and check trust states.
	resp, err := s.ImpactRadius(ctx, &servicesv1.ImpactRadiusRequest{
		Node:          req.GetSubjectNode(),
		MaxDepth:      5,
		MinTrustState: "verified",
	})
	if err != nil {
		return nil, err
	}

	return &servicesv1.VerifyClosureResponse{
		SubjectNode:  req.GetSubjectNode(),
		Purpose:      req.GetPurpose(),
		Verdict:      len(resp.Edges) > 0,
		TrustState:   "incomplete",
		Diagnostics:  []string{"verify-closure placeholder"},
		EdgesChecked: int32(len(resp.Edges)),
		EdgesFailed:  0,
	}, nil
}

// ---- helpers ----

func nodeToProto(n *Node) *servicesv1.GraphNode {
	return &servicesv1.GraphNode{
		Id:         n.ID,
		Kind:       n.Kind,
		Urn:        n.URN,
		LabelsJson: n.LabelsJSON,
		CreatedAt:  timestamppb.New(n.CreatedAt),
	}
}

func edgeToProto(e *Edge) *servicesv1.GraphEdge {
	edge := &servicesv1.GraphEdge{
		Id:             e.ID,
		FromNode:       e.FromNode,
		ToNode:         e.ToNode,
		Relation:       e.Relation,
		Qualifier:      e.Qualifier,
		ClaimId:        e.ClaimID,
		EvidenceDigest: e.EvidenceDigest,
		ProofState:     parseProofState(e.ProofStateJSON),
		TrustState:     e.TrustState,
		Weight:         e.Weight,
		ExtensionsJson: e.ExtensionsJSON,
	}
	if !e.ValidFrom.IsZero() {
		edge.ValidTime = &servicesv1.TimeWindow{FromTime: timestamppb.New(e.ValidFrom)}
		if e.ValidTo != nil {
			edge.ValidTime.ToTime = timestamppb.New(*e.ValidTo)
		}
	}
	return edge
}

func parseProofState(jsonStr string) *servicesv1.ProofState {
	if jsonStr == "" {
		return &servicesv1.ProofState{
			Discovery: "candidate", Graph: "unresolved", Claim: "unbound",
			Source: "unbound", Signature: "unverified", Transparency: "unverified",
			WitnessJson: `{"status":"unverified"}`, State: "unverified", Policy: "unevaluated",
		}
	}
	var ps struct {
		Discovery    string `json:"discovery"`
		Graph        string `json:"graph"`
		Claim        string `json:"claim"`
		Source       string `json:"source"`
		Signature    string `json:"signature"`
		Transparency string `json:"transparency"`
		WitnessJson  string `json:"witness_json"`
		State        string `json:"state"`
		Policy       string `json:"policy"`
	}
	json.Unmarshal([]byte(jsonStr), &ps)
	return &servicesv1.ProofState{
		Discovery: ps.Discovery, Graph: ps.Graph, Claim: ps.Claim,
		Source: ps.Source, Signature: ps.Signature, Transparency: ps.Transparency,
		WitnessJson: ps.WitnessJson, State: ps.State, Policy: ps.Policy,
	}
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
