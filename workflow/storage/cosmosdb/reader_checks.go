package cosmosdb

import (
	"context"
	"fmt"
	"time"

	"github.com/element-of-surprise/coercion/workflow"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// idToCheck reads a field from the statement and returns a workflow.Checks  object. stmt must be
// from a Plan or Block query.
func (p reader) idToCheck(ctx context.Context, id uuid.UUID) (*workflow.Checks, error) {
	if id == uuid.Nil {
		return &workflow.Checks{}, nil
	}
	return p.fetchChecksByID(ctx, id)
}

// fetchChecksByID fetches a Checks object by its ID.
func (p reader) fetchChecksByID(ctx context.Context, id uuid.UUID) (*workflow.Checks, error) {
	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	res, err := p.cc.GetChecksClient().ReadItem(ctx, p.cc.GetPK(), id.String(), itemOpt)
	if err != nil {
		// return p, fmt.Errorf("failed to read item through Cosmos DB API: %w", cosmosErr(err))
		return nil, fmt.Errorf("couldn't fetch checks by id: %w", err)
	}
	check, err := p.checksRowToChecks(ctx, &res)
	if err != nil {
		return nil, err
	}
	if check == nil {
		return nil, fmt.Errorf("couldn't find checks by id(%s)", id)
	}
	return check, nil
}

func (p reader) checksRowToChecks(ctx context.Context, response *azcosmos.ItemResponse) (*workflow.Checks, error) {
	var err error
	var resp checksEntry
	err = json.Unmarshal(response.Value, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal check: %w", err)
	}

	c := &workflow.Checks{}
	c.ID = resp.ID
	k := resp.Key
	if k != uuid.Nil {
		c.Key = k
	}
	c.Delay = time.Duration(resp.Delay)
	c.State, err = fieldToState(resp.StateStatus, resp.StateStart, resp.StateEnd)
	if err != nil {
		return nil, fmt.Errorf("checksRowToChecks: %w", err)
	}
	c.Actions, err = p.idsToActions(ctx, resp.Actions)
	if err != nil {
		return nil, fmt.Errorf("couldn't get actions ids: %w", err)
	}

	return c, nil
}
