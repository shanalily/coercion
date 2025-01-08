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

// strToCheck reads a field from the statement and returns a workflow.Checks  object. stmt must be
// from a Plan or Block query.
func (p reader) strToCheck(ctx context.Context, strID string) (*workflow.Checks, error) {
	if strID == "" {
		return nil, nil
	}
	id, err := uuid.Parse(strID)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert ID to UUID: %w", err)
	}
	return p.fetchChecksByID(ctx, id)
}

// fetchChecksByID fetches a Checks object by its ID.
func (p reader) fetchChecksByID(ctx context.Context, id uuid.UUID) (*workflow.Checks, error) {
	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	res, err := p.cc.GetChecksClient().ReadItem(ctx, p.cc.partitionKey, id.String(), itemOpt)
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
		return nil, err
	}

	c := &workflow.Checks{}
	c.ID, err = uuid.Parse(resp.id)
	if err != nil {
		return nil, fmt.Errorf("checksRowToChecks: couldn't convert ID to UUID: %w", err)
	}
	k := resp.key
	if k != "" {
		c.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse check key: %w", err)
		}
	}
	c.Delay = time.Duration(resp.delay)
	c.State, err = fieldToState(resp.stateStatus, resp.stateStart, resp.stateEnd)
	if err != nil {
		return nil, fmt.Errorf("checksRowToChecks: %w", err)
	}
	c.Actions, err = p.strToActions(ctx, resp.actions)
	if err != nil {
		return nil, fmt.Errorf("couldn't get actions ids: %w", err)
	}

	return c, nil
}
