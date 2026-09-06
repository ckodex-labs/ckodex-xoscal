package graph

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func nodeID(ref map[string]string) string {
	kind := ref["kind"]
	if kind == "" {
		kind = "reference"
	}
	if ref["id"] != "" {
		return kind + ":" + ref["id"]
	}
	if ref["digest"] != "" {
		return kind + ":" + ref["digest"]
	}
	return ""
}

func nodeURN(ref map[string]string) string {
	if ref["purl"] != "" {
		return ref["purl"]
	}
	if ref["bom_ref"] != "" {
		return ref["bom_ref"]
	}
	if ref["id"] != "" {
		return "urn:xoscal:node:" + ref["kind"] + ":" + ref["id"]
	}
	return "urn:xoscal:node:" + ref["kind"] + ":" + ref["digest"]
}

func defaultProjectionEdgeID(claimID, fromNode, toNode, relation string) string {
	payload, _ := json.Marshal(struct {
		Version  string `json:"version"`
		ClaimID  string `json:"claim_id"`
		FromNode string `json:"from_node"`
		ToNode   string `json:"to_node"`
		Relation string `json:"relation"`
	}{
		Version:  "graph-edge-v1",
		ClaimID:  claimID,
		FromNode: fromNode,
		ToNode:   toNode,
		Relation: relation,
	})
	sum := sha256.Sum256(payload)
	return "edge_" + hex.EncodeToString(sum[:])
}

func mapStoreError(operation, id string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return status.Errorf(codes.NotFound, "%s %q not found", operation, id)
	}
	return status.Errorf(codes.Internal, "%s: %v", operation, err)
}

func mapGraphQueryError(operation string, err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "invalid minimum trust state") {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Errorf(codes.Internal, "%s: %v", operation, err)
}

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

func projectionEventToProto(event *ProjectionEvent) *servicesv1.GraphProjectionEvent {
	if event == nil {
		return nil
	}
	return &servicesv1.GraphProjectionEvent{
		Sequence:       event.Sequence,
		EventId:        event.EventID,
		EdgeId:         event.EdgeID,
		ClaimId:        event.ClaimID,
		FromNode:       event.FromNode,
		ToNode:         event.ToNode,
		Relation:       event.Relation,
		EvidenceDigest: event.EvidenceDigest,
		TrustState:     event.TrustState,
		PreviousHash:   event.PreviousHash,
		EventHash:      event.EventHash,
		ProjectedAt:    timestamppb.New(event.ProjectedAt),
	}
}

func parseProofState(jsonStr string) *servicesv1.ProofState {
	if jsonStr == "" {
		return defaultProofState()
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
	if err := json.Unmarshal([]byte(jsonStr), &ps); err != nil {
		return defaultProofState()
	}
	return &servicesv1.ProofState{
		Discovery: ps.Discovery, Graph: ps.Graph, Claim: ps.Claim,
		Source: ps.Source, Signature: ps.Signature, Transparency: ps.Transparency,
		WitnessJson: ps.WitnessJson, State: ps.State, Policy: ps.Policy,
	}
}

func defaultProofState() *servicesv1.ProofState {
	return &servicesv1.ProofState{
		Discovery: "candidate", Graph: "unresolved", Claim: "unbound",
		Source: "unbound", Signature: "unverified", Transparency: "unverified",
		WitnessJson: `{"status":"unverified"}`, State: "unverified", Policy: "unevaluated",
	}
}

const graphMaxInt32Value = 1<<31 - 1

const maxGraphListPageSize = 1000

func boundedInt32(value int) int32 {
	if value > graphMaxInt32Value {
		return graphMaxInt32Value
	}
	if value < -graphMaxInt32Value-1 {
		return -graphMaxInt32Value - 1
	}
	// #nosec G115 -- the bounds above prove that value fits in int32.
	return int32(value)
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
