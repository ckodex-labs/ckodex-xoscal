package transparency

import (
	"net/http"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

// ExchangeServer implements TransparencyExchangeService.
type ExchangeServer struct {
	servicesv1.UnimplementedTransparencyExchangeServiceServer
	store         Store
	fetchPolicy   *FetchPolicy
	fetchClientFn func(*FetchPolicy) *http.Client
}

const (
	maxImportRecords = 10
	maxListPageSize  = 1000
)

// WithFetchPolicy bounds the external evidence fetcher. A nil policy (the
// default) keeps fetching disabled: the RPC fails closed.
func (s *ExchangeServer) WithFetchPolicy(policy *FetchPolicy) *ExchangeServer {
	s.fetchPolicy = policy
	return s
}
