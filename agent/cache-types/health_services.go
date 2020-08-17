package cachetype

import (
	"fmt"

	"github.com/hashicorp/consul/agent/cache"
	"github.com/hashicorp/consul/agent/structs"
)

// Recommended name for registration.
const HealthServicesName = "health-services"

// HealthServices supports fetching discovering service instances via the
// catalog.
type HealthServices struct {
	RegisterOptionsBlockingRefresh
	RPC RPC
}

func (c *HealthServices) Fetch(opts cache.FetchOptions, req cache.Request) (cache.FetchResult, error) {
	var result cache.FetchResult

	// The request should be a DCSpecificRequest.
	reqReal, ok := req.(*structs.ServiceSpecificRequest)
	if !ok {
		return result, fmt.Errorf(
			"Internal cache failure: request wrong type: %T", req)
	}

	// Lightweight copy this object so that manipulating QueryOptions doesn't race.
	dup := *reqReal
	reqReal = &dup

	// Set the minimum query index to our current index so we block
	reqReal.QueryOptions.MinQueryIndex = opts.MinIndex
	reqReal.QueryOptions.MaxQueryTime = opts.Timeout

	// Always allow stale - there's no point in hitting leader if the request is
	// going to be served from cache and end up arbitrarily stale anyway. This
	// allows cached entries to automatically read scale across all
	// servers too.
	reqReal.QueryOptions.AllowStale = true

	if opts.LastResult != nil {
		reqReal.QueryOptions.AllowNotModifiedResponse = true
	}

	var reply structs.IndexedCheckServiceNodes
	if err := c.RPC.RPC("Health.ServiceNodes", reqReal, &reply); err != nil {
		return result, err
	}

	result.Value = &reply
	result.Index = reply.QueryMeta.Index
	result.NotModified = reply.QueryMeta.NotModified
	return result, nil
}
