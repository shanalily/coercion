package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"
)

var _ storage.ActionUpdater = actionUpdater{}

// actionUpdater implements the storage.actionUpdater interface.
type actionUpdater struct {
	mu *sync.Mutex
	cc ContainerClient
	pk azcosmos.PartitionKey

	private.Storage
}

// UpdateAction implements storage.ActionUpdater.UpdateAction().
func (a actionUpdater) UpdateAction(ctx context.Context, action *workflow.Action) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	patch := azcosmos.PatchOperations{}
	patch.AppendReplace("/stateStatus", action.State.Status)
	patch.AppendReplace("/stateStart", action.State.Start)
	patch.AppendReplace("/stateEnd", action.State.End)

	var ifMatchEtag *azcore.ETag = nil
	if action.State.ETag != "" {
		ifMatchEtag = (*azcore.ETag)(&action.State.ETag)
	}
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		IfMatchEtag:                  ifMatchEtag,
	}

	// save the item into Cosmos DB
	res, err := a.cc.PatchItem(ctx, a.pk, action.ID.String(), patch, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to patch item through Cosmos DB API: %w", err)
	}
	fmt.Println(res.ETag)

	return nil
}
