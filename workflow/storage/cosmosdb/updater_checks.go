package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

var _ storage.ChecksUpdater = checksUpdater{}

// checksUpdater implements the storage.checksUpdater interface.
type checksUpdater struct {
	mu *sync.Mutex
	cc ContainerClient
	pk azcosmos.PartitionKey

	private.Storage
}

// UpdateChecks implements storage.ChecksUpdater.UpdateCheck().
func (c checksUpdater) UpdateChecks(ctx context.Context, check *workflow.Checks) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	patch := azcosmos.PatchOperations{}
	patch.AppendReplace("/stateStatus", int64(check.State.Status))
	patch.AppendReplace("/stateStart", check.State.Start.UnixNano())
	patch.AppendReplace("/stateEnd", check.State.End.UnixNano())

	// var ifMatchEtag *azcore.ETag = nil
	// if etag, ok := h.GetEtag(item); ok {
	// 	ifMatchEtag = (*azcore.ETag)(&etag)
	// }
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		// IfMatchEtag:                  ifMatchEtag,
	}

	// save the item into Cosmos DB
	res, err := c.cc.PatchItem(ctx, c.pk, check.ID.String(), patch, itemOpt)
	if err != nil {
		// return fmt.Errorf("failed to update item through Cosmos DB API: %w", cosmosErr(err))
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res.ETag)

	return nil

}
