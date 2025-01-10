package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/google/uuid"
)

type deleter struct {
	mu *sync.Mutex
	cc Client

	reader reader
}

// Delete deletes a plan with "id" from the storage.
func (d deleter) Delete(ctx context.Context, id uuid.UUID) error {
	plan, err := d.reader.Read(ctx, id)
	if err != nil {
		return fmt.Errorf("couldn't fetch plan: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// var ifMatchEtag *azcore.ETag = nil
	// if etag, ok := h.GetEtag(item); ok {
	//   ifMatchEtag = (*azcore.ETag)(&etag)
	// }
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		// IfMatchEtag:                  ifMatchEtag,
	}

	if err = d.deletePlan(ctx, plan, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete plan: %w", err)
	}
	return nil
}

func (d deleter) deletePlan(ctx context.Context, plan *workflow.Plan, itemOpt *azcosmos.ItemOptions) error {
	if err := d.deleteChecks(ctx, plan.PreChecks, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete plan prechecks: %w", err)
	}
	if err := d.deleteChecks(ctx, plan.PostChecks, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete plan postchecks: %w", err)
	}
	if err := d.deleteChecks(ctx, plan.ContChecks, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete plan contchecks: %w", err)
	}
	if err := d.deleteChecks(ctx, plan.DeferredChecks, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete plan deferredchecks: %w", err)
	}
	if err := d.deleteBlocks(ctx, plan.Blocks, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete blocks: %w", err)
	}

	// Do I care about etag here? I don't have it for the other items unless I read.
	// Once we're at the point where a plan needs to be deleted, I assume it's completed (whether failed or successful)
	// and no important operations will take place on this other than to delete.
	var ifMatchEtag *azcore.ETag = nil
	if plan.State.ETag != "" {
		ifMatchEtag = (*azcore.ETag)(&plan.State.ETag)
	}
	itemOpt.IfMatchEtag = ifMatchEtag

	if _, err := d.cc.GetPlansClient().DeleteItem(ctx, d.cc.GetPK(), plan.ID.String(), itemOpt); err != nil {
		return fmt.Errorf("failed to delete plan through Cosmos DB API: %w", err)
	}

	return nil
}

func (d deleter) deleteBlocks(ctx context.Context, blocks []*workflow.Block, itemOpt *azcosmos.ItemOptions) error {
	if len(blocks) == 0 {
		return nil
	}

	for _, block := range blocks {
		if err := d.deleteChecks(ctx, block.PreChecks, itemOpt); err != nil {
			return fmt.Errorf("couldn't delete block prechecks: %w", err)
		}
		if err := d.deleteChecks(ctx, block.PostChecks, itemOpt); err != nil {
			return fmt.Errorf("couldn't delete block postchecks: %w", err)
		}
		if err := d.deleteChecks(ctx, block.ContChecks, itemOpt); err != nil {
			return fmt.Errorf("couldn't delete block contchecks: %w", err)
		}
		if err := d.deleteChecks(ctx, block.DeferredChecks, itemOpt); err != nil {
			return fmt.Errorf("couldn't delete block deferredchecks: %w", err)
		}
		if err := d.deletesSeqs(ctx, block.Sequences, itemOpt); err != nil {
			return fmt.Errorf("couldn't delete block sequences: %w", err)
		}
	}

	// should skip not found errors if part of the plan was deleted previously and this is being retried?
	for _, block := range blocks {
		if _, err := d.cc.GetBlocksClient().DeleteItem(ctx, d.cc.GetPK(), block.ID.String(), itemOpt); err != nil {
			return fmt.Errorf("failed to delete block through Cosmos DB API: %w", err)
		}
	}
	return nil
}

func (d deleter) deleteChecks(ctx context.Context, checks *workflow.Checks, itemOpt *azcosmos.ItemOptions) error {
	if checks == nil {
		return nil
	}

	if err := d.deleteActions(ctx, checks.Actions, itemOpt); err != nil {
		return fmt.Errorf("couldn't delete checks actions: %w", err)
	}

	if _, err := d.cc.GetChecksClient().DeleteItem(ctx, d.cc.GetPK(), checks.ID.String(), itemOpt); err != nil {
		return fmt.Errorf("failed to delete checks through Cosmos DB API: %w", err)
	}
	return nil
}

func (d deleter) deletesSeqs(ctx context.Context, seqs []*workflow.Sequence, itemOpt *azcosmos.ItemOptions) error {
	if len(seqs) == 0 {
		return nil
	}

	for _, seq := range seqs {
		if err := d.deleteActions(ctx, seq.Actions, itemOpt); err != nil {
			return fmt.Errorf("couldn't delete sequence actions: %w", err)
		}
	}

	for _, seq := range seqs {
		if _, err := d.cc.GetSequencesClient().DeleteItem(ctx, d.cc.GetPK(), seq.ID.String(), itemOpt); err != nil {
			return fmt.Errorf("failed to delete sequence through Cosmos DB API: %w", err)
		}
	}
	return nil
}

func (d deleter) deleteActions(ctx context.Context, actions []*workflow.Action, itemOpt *azcosmos.ItemOptions) error {
	if len(actions) == 0 {
		return nil
	}

	for _, action := range actions {
		if _, err := d.cc.GetActionsClient().DeleteItem(ctx, d.cc.GetPK(), action.ID.String(), itemOpt); err != nil {
			return fmt.Errorf("failed to delete action through Cosmos DB API: %w", err)
		}
	}
	return nil
}
