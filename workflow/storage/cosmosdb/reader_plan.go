package cosmosdb

import (
	"context"
	"fmt"

	"github.com/go-json-experiment/json"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/google/uuid"
)

// fetchPlan fetches a plan by its id.
func (p reader) fetchPlan(ctx context.Context, id uuid.UUID) (*workflow.Plan, error) {
	// plan := &workflow.Plan{}

	//  do I need	fetchPlanByID,
	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	// need to get partition key
	res, err := p.cc.GetPlansClient().ReadItem(ctx, p.cc.partitionKey, id.String(), itemOpt)
	if err != nil {
		// return p, fmt.Errorf("failed to read item through Cosmos DB API: %w", cosmosErr(err))
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

	plan := &workflow.Plan{}
	plan.ID, err = uuid.Parse(resp.ID)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert ID to UUID: %w", err)
	}
	gid := resp.GroupID
	if gid == "" {
		plan.GroupID = uuid.Nil
	} else {
		plan.GroupID, err = uuid.Parse(resp.GroupID)
		if err != nil {
			return nil, fmt.Errorf("couldn't convert GroupID to UUID: %w", err)
		}
	}
	plan.Name = resp.Name
	plan.Descr = resp.Descr
	plan.SubmitTime, err = timeFromInt64(resp.SubmitTime)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan submit time: %w", err)
	}
	plan.State, err = fieldToState(resp.StateStatus, resp.StateStart, resp.StateEnd)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan state: %w", err)
	}

	// if b := strToBytes("meta", stmt); b != nil {
	// 	plan.Meta = b
	// }
	plan.BypassChecks, err = p.strToCheck(ctx, resp.BypassChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan bypasschecks: %w", err)
	}
	plan.PreChecks, err = p.strToCheck(ctx, resp.PreChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan prechecks: %w", err)
	}
	plan.ContChecks, err = p.strToCheck(ctx, resp.ContChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan contchecks: %w", err)
	}
	plan.PostChecks, err = p.strToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan postchecks: %w", err)
	}
	plan.DeferredChecks, err = p.strToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get plan deferredchecks: %w", err)
	}
	plan.Blocks, err = p.strToBlocks(ctx, resp.Blocks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get blocks: %w", err)
	}
	return plan, nil
}
