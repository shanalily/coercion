package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"
)

var _ storage.BlockUpdater = blockUpdater{}

// blockUpdater implements the storage.blockUpdater interface.
type blockUpdater struct {
	mu *sync.Mutex
	cc ContainerClient
	pk azcosmos.PartitionKey

	private.Storage
}

// UpdateBlock implements storage.Blockupdater.UpdateBlock().
func (b blockUpdater) UpdateBlock(ctx context.Context, block *workflow.Block) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	patch := azcosmos.PatchOperations{}
	patch.AppendReplace("/stateStatus", block.State.Status)
	patch.AppendReplace("/stateStart", block.State.Start)
	patch.AppendReplace("/stateEnd", block.State.End)

	// var ifMatchEtag *azcore.ETag = nil
	// if etag, ok := h.GetEtag(item); ok {
	// 	ifMatchEtag = (*azcore.ETag)(&etag)
	// }
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		// IfMatchEtag:                  ifMatchEtag,
	}

	// save the item into Cosmos DB
	res, err := b.cc.PatchItem(ctx, b.pk, block.ID.String(), patch, itemOpt)
	if err != nil {
		// return fmt.Errorf("failed to update item through Cosmos DB API: %w", cosmosErr(err))
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res.ETag)

	return nil
}
