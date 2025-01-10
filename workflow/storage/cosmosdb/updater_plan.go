package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	// "github.com/go-json-experiment/json"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"
)

var _ storage.PlanUpdater = planUpdater{}

// planUpdater implements the storage.PlanUpdater interface.
type planUpdater struct {
	mu *sync.Mutex
	cc ContainerClient
	pk azcosmos.PartitionKey

	private.Storage
}

// UpdatePlan implements storage.PlanUpdater.UpdatePlan().
func (u planUpdater) UpdatePlan(ctx context.Context, p *workflow.Plan) error {
	// not necessary right now?
	u.mu.Lock()
	defer u.mu.Unlock()

	patch := azcosmos.PatchOperations{}
	patch.AppendReplace("/reason", p.Reason)
	patch.AppendReplace("/stateStatus", p.State.Status)
	patch.AppendReplace("/stateStart", p.State.Start)
	patch.AppendReplace("/stateEnd", p.State.End)

	var ifMatchEtag *azcore.ETag = nil
	if p.State.ETag != "" {
		ifMatchEtag = (*azcore.ETag)(&p.State.ETag)
	}
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		IfMatchEtag:                  ifMatchEtag,
	}

	// save the item into Cosmos DB
	res, err := u.cc.PatchItem(ctx, u.pk, p.ID.String(), patch, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to patch item through Cosmos DB API: %w", err)
	}
	fmt.Println(res.ETag)

	return nil
}
