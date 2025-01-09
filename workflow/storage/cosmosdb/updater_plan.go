package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	// "github.com/go-json-experiment/json"
	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

var _ storage.PlanUpdater = planUpdater{}

// planUpdater implements the storage.PlanUpdater interface.
type planUpdater struct {
	mu *sync.Mutex
	cc ContainerClient
	pk string

	private.Storage
}

// UpdatePlan implements storage.PlanUpdater.UpdatePlan().
func (u planUpdater) UpdatePlan(ctx context.Context, p *workflow.Plan) error {
	// not necessary right now?
	u.mu.Lock()
	defer u.mu.Unlock()

	// plan, err := planToEntry(ctx, u.pk, p)
	// if err != nil {
	// 	return err
	// }

	// should I be making sure only these fields are updated?
	// patch item instead?
	patch := azcosmos.PatchOperations{}
	patch.AppendReplace("/stateStatus", int64(p.State.Status))
	patch.AppendReplace("/stateStart", p.State.Start.UnixNano())
	patch.AppendReplace("/stateEnd", p.State.End.UnixNano())
	// stmt.SetText("$id", plan.ID.String())
	// stmt.SetInt64("$reason", int64(plan.Reason))
	// stmt.SetInt64("$state_status", int64(plan.State.Status))
	// stmt.SetInt64("$state_start", plan.State.Start.UnixNano())
	// stmt.SetInt64("$state_end", plan.State.End.UnixNano())

	// var ifMatchEtag *azcore.ETag = nil
	// if etag, ok := h.GetEtag(item); ok {
	// 	ifMatchEtag = (*azcore.ETag)(&etag)
	// }
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		// IfMatchEtag:                  ifMatchEtag,
	}

	// // save the item into Cosmos DB
	// itemJson, err := json.Marshal(plan)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal JSON while create Item in Cosmos DB: %w", err)
	// }
	// res, err := u.cc.ReplaceItem(ctx, u.cc.GetPK(), plan.ID.String(), itemJson, itemOpt)
	// res, err := u.cc.ReplaceItem(ctx, partitionKey(u.pk), plan.ID.String(), itemJson, itemOpt)
	res, err := u.cc.PatchItem(ctx, partitionKey(u.pk), p.ID.String(), patch, itemOpt)
	if err != nil {
		// return fmt.Errorf("failed to update item through Cosmos DB API: %w", cosmosErr(err))
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res.ETag)

	return nil
}
