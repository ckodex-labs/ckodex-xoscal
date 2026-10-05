package main

import (
	"fmt"
	services "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/protobuf/proto"
)

func main() {
	original := &services.ListCatalogsRequest{PageSize: 17}
	raw, err := proto.Marshal(original)
	if err != nil {
		panic(err)
	}
	copy := &services.ListCatalogsRequest{}
	if err := proto.Unmarshal(raw, copy); err != nil {
		panic(err)
	}
	if copy.PageSize != 17 {
		panic("protobuf roundtrip failed")
	}
	if services.OscalService_ServiceDesc.ServiceName != "oscal.services.v1.OscalService" {
		panic("OSCAL service missing")
	}
	if services.GovernanceService_ServiceDesc.ServiceName == "" || services.TransparencyExchangeService_ServiceDesc.ServiceName == "" || services.TransparencyGraphService_ServiceDesc.ServiceName == "" {
		panic("service missing")
	}
	fmt.Println("Go SDK message and four service descriptors passed")
}
