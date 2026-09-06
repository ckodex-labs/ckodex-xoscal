package transparency

import (
	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

// ExchangeServer implements TransparencyExchangeService.
type ExchangeServer struct {
	servicesv1.UnimplementedTransparencyExchangeServiceServer
	store Store
}

const (
	maxImportRecords = 10
	maxListPageSize  = 1000
)
