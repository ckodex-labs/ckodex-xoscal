package transparency

import (
	"net/http"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

// ExchangeServer implements TransparencyExchangeService.
type ExchangeServer struct {
	servicesv1.UnimplementedTransparencyExchangeServiceServer
	store           Store
	fetchPolicy     *FetchPolicy
	fetchClientFn   func(*FetchPolicy) *http.Client
	keyRegistry     *KeyRegistry
	transparencyCfg *TransparencyConfig
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

// WithKeyRegistry registers the deployment's issuer key set. Rotation keeps
// prior keys registered as retired; claims signed with a retired key still
// verify but surface the rotation in diagnostics.
func (s *ExchangeServer) WithKeyRegistry(registry *KeyRegistry) *ExchangeServer {
	s.keyRegistry = registry
	return s
}

// WithTransparencyConfig enables the transparency inclusion provider against
// the configured log. A nil config keeps the check explicitly unavailable.
func (s *ExchangeServer) WithTransparencyConfig(cfg *TransparencyConfig) *ExchangeServer {
	s.transparencyCfg = cfg
	return s
}
