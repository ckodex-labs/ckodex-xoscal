package service

import (
	"strconv"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func toKGEntity(p *servicesv1.Entity) *kg.Entity {
	version := 1
	if p.Version != "" {
		if v, err := strconv.Atoi(p.Version); err == nil {
			version = v
		}
	}
	return &kg.Entity{
		URN:     p.Urn,
		Type:    p.Type,
		Version: version,
		Status:  kg.EntityStatus(p.Status),
		Payload: []byte(p.Payload),
	}
}

func toProtoEntity(e *kg.Entity) *servicesv1.Entity {
	return &servicesv1.Entity{
		Urn:     e.URN,
		Type:    e.Type,
		Version: strconv.Itoa(e.Version),
		Status:  string(e.Status),
		Payload: string(e.Payload),
	}
}

const maxInt32Value = 1<<31 - 1

// boundedInt32 converts a count or identifier only after constraining it to
// the range representable by the protobuf int32 field.
func boundedInt32(value int) int32 {
	if value > maxInt32Value {
		return maxInt32Value
	}
	if value < -maxInt32Value-1 {
		return -maxInt32Value - 1
	}
	// #nosec G115 -- the bounds above prove that value fits in int32.
	return int32(value)
}
