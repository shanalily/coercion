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

// fieldToBlocks converts the "$blocks" field in a cosmosdb row to a list of workflow.Blocks.
func (p reader) strToBlocks(ctx context.Context, blockIDs string) ([]*workflow.Block, error) {
	ids, err := strToIDs(blockIDs)
	if err != nil {
		return nil, fmt.Errorf("couldn't read plan block ids: %w", err)
	}

	// opt := azcosmos.QueryOptions{
	// 	QueryParameters: []azcosmos.QueryParameter{
	// 		{"@value", "2"},
	// 	},
	// }
	// pk := azcosmos.NewPartitionKeyString("myPartitionKeyValue")
	// queryPager := container.NewQueryItemsPager("select * from docs c where c.value = @value", pk, &opt)
	// for queryPager.More() {
	// 	queryResponse, err := queryPager.NextPage(context)
	// 	if err != nil {
	// 		handle(err)
	// 	}

	// 	for _, item := range queryResponse.Items {
	// 		var itemResponseBody map[string]interface{}
	// 		json.Unmarshal(item, &itemResponseBody)
	// 	}
	// }

	blocks := make([]*workflow.Block, 0, len(ids))
	for _, id := range ids {
		block, err := p.fetchBlockByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch block(%s)by id: %w", id, err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// fetchBlockByID fetches a block by its id.
func (p reader) fetchBlockByID(ctx context.Context, id uuid.UUID) (*workflow.Block, error) {
	// block := &workflow.Block{}

	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	res, err := p.cc.GetBlocksClient().ReadItem(ctx, p.cc.GetPK(), id.String(), itemOpt)
	if err != nil {
		// return p, fmt.Errorf("failed to read item through Cosmos DB API: %w", cosmosErr(err))
		return nil, fmt.Errorf("couldn't fetch block by id: %w", err)
	}

	return p.blockRowToBlock(ctx, &res)
}

// blockRowToBlock converts a cosmosdb row to a workflow.Block.
func (p reader) blockRowToBlock(ctx context.Context, response *azcosmos.ItemResponse) (*workflow.Block, error) {
	b := &workflow.Block{}

	var err error
	var resp blocksEntry
	err = json.Unmarshal(response.Value, &resp)
	if err != nil {
		return nil, err
	}

	b.ID, err = uuid.Parse(resp.ID)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert ID to UUID: %w", err)
	}

	k := resp.Key
	if k != "" {
		b.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse block key: %w", err)
		}
	}
	b.Name = resp.Name
	b.Descr = resp.Descr
	b.EntranceDelay = time.Duration(resp.EntranceDelay)
	b.ExitDelay = time.Duration(resp.ExitDelay)
	b.State, err = fieldToState(resp.StateStatus, resp.StateStart, resp.StateEnd)
	if err != nil {
		return nil, fmt.Errorf("blockRowToBlock: %w", err)
	}
	b.Concurrency = int(resp.Concurrency)
	b.ToleratedFailures = int(resp.ToleratedFailures)
	b.BypassChecks, err = p.strToCheck(ctx, resp.BypassChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block bypasschecks: %w", err)
	}
	b.PreChecks, err = p.strToCheck(ctx, resp.PreChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block prechecks: %w", err)
	}
	b.ContChecks, err = p.strToCheck(ctx, resp.ContChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block contchecks: %w", err)
	}
	b.PostChecks, err = p.strToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block postchecks: %w", err)
	}
	b.DeferredChecks, err = p.strToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block deferredchecks: %w", err)
	}
	b.Sequences, err = p.strToSequences(ctx, resp.Sequences)
	if err != nil {
		return nil, fmt.Errorf("couldn't read block sequences: %w", err)
	}

	return b, nil
}
