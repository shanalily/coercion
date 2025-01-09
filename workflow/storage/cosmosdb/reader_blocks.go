package cosmosdb

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
)

// idsToBlocks converts the "$blocks" field in a cosmosdb row to a list of workflow.Blocks.
func (p reader) idsToBlocks(ctx context.Context, blockIDs []uuid.UUID) ([]*workflow.Block, error) {
	blocks := make([]*workflow.Block, 0, len(blockIDs))
	for _, id := range blockIDs {
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

	b.ID = resp.ID
	k := resp.Key
	if k != uuid.Nil {
		b.Key = k
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
	b.BypassChecks, err = p.idToCheck(ctx, resp.BypassChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block bypasschecks: %w", err)
	}
	b.PreChecks, err = p.idToCheck(ctx, resp.PreChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block prechecks: %w", err)
	}
	b.ContChecks, err = p.idToCheck(ctx, resp.ContChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block contchecks: %w", err)
	}
	b.PostChecks, err = p.idToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block postchecks: %w", err)
	}
	b.DeferredChecks, err = p.idToCheck(ctx, resp.PostChecks)
	if err != nil {
		return nil, fmt.Errorf("couldn't get block deferredchecks: %w", err)
	}
	b.Sequences, err = p.idsToSequences(ctx, resp.Sequences)
	if err != nil {
		return nil, fmt.Errorf("couldn't read block sequences: %w", err)
	}

	return b, nil
}
