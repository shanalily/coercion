package cosmosdb

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
)

// fetchPlan fetches a plan by its id.
func (p reader) fetchPlan(ctx context.Context, id uuid.UUID) (*workflow.Plan, error) {
	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	res, err := p.cc.GetPlansClient().ReadItem(ctx, p.cc.GetPK(), id.String(), itemOpt)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch plan: %w", err)
	}
	return p.convertToPlan(ctx, &res)
}

func (p reader) convertToPlan(ctx context.Context, response *azcosmos.ItemResponse) (*workflow.Plan, error) {
	var err error
	var resp plansEntry
	err = json.Unmarshal(response.Value, &resp)
	if err != nil {
		return nil, err
	}

	plan := &workflow.Plan{
		ID:         resp.ID,
		Name:       resp.Name,
		Descr:      resp.Descr,
		SubmitTime: resp.SubmitTime,
		State: &workflow.State{
			Status: resp.StateStatus,
			Start:  resp.StateStart,
			End:    resp.StateEnd,
		},
	}
	gid := resp.GroupID
	if gid == uuid.Nil {
		plan.GroupID = uuid.Nil
	} else {
		plan.GroupID = resp.GroupID
	}
	plan.BypassChecks, err = p.idToCheck(ctx, resp.BypassChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan bypasschecks: %w", err)
	}
	plan.PreChecks, err = p.idToCheck(ctx, resp.PreChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan prechecks: %w", err)
	}
	plan.ContChecks, err = p.idToCheck(ctx, resp.ContChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan contchecks: %w", err)
	}
	plan.PostChecks, err = p.idToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan postchecks: %w", err)
	}
	plan.DeferredChecks, err = p.idToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan deferredchecks: %w", err)
	}
	plan.Blocks, err = p.idsToBlocks(ctx, resp.Blocks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get blocks: %w", err)
	}
	return plan, nil
}
