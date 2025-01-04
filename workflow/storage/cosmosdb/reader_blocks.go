package cosmosdb

import (
	"context"
	"fmt"
	"time"

	"github.com/element-of-surprise/coercion/workflow"
	"github.com/google/uuid"
	// "zombiezen.com/go/cosmosdb"
	// "zombiezen.com/go/cosmosdb/cosmosdbx"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// fieldToBlocks converts the "$blocks" field in a cosmosdb row to a list of workflow.Blocks.
func (p reader) fieldToBlocks(ctx context.Context) ([]*workflow.Block, error) {
	// ids, err := fieldToIDs("blocks", stmt)
	// if err != nil {
	// 	return nil, fmt.Errorf("couldn't read plan block ids: %w", err)
	// }

	blocks := make([]*workflow.Block, 0, len(ids))
	for _, id := range ids {
		block, err := p.fetchBlockByID(ctx, conn, id)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch block(%s)by id: %w", id, err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// fetchBlockByID fetches a block by its id.
func (p reader) fetchBlockByID(ctx context.Context, id uuid.UUID) (*workflow.Block, error) {
	block := &workflow.Block{}

	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	parititionKey := "underlayName"
	res, err := p.blocksClient.ReadItem(ctx, partitionKey, id.String(), itemOpt)
	if err != nil {
		return p, fmt.Errorf("failed to read item through Cosmos DB API: %w", cosmosErr(err))
	}

	// do := func(conn *cosmosdb.Conn) (err error) {
	// 	err = cosmosdbx.Execute(
	// 		conn,
	// 		fetchBlocksByID,
	// 		&cosmosdbx.ExecOptions{
	// 			Named: map[string]any{
	// 				"$id": id.String(),
	// 			},
	// 			ResultFunc: func(stmt *cosmosdb.Stmt) error {
	// 				block, err = p.blockRowToBlock(ctx, conn, stmt)
	// 				if err != nil {
	// 					return fmt.Errorf("couldn't convert row to block: %w", err)
	// 				}
	// 				return nil
	// 			},
	// 		},
	// 	)
	// 	if err != nil {
	// 		return fmt.Errorf("couldn't fetch block by id: %w", err)
	// 	}
	// 	return nil
	// }

	// if err := do(conn); err != nil {
	// 	return nil, fmt.Errorf("couldn't fetch block by id: %w", err)
	// }
	return blockRowToBlock(res)
}

// blockRowToBlock converts a cosmosdb row to a workflow.Block.
func (p reader) blockRowToBlock(response *azcosmos.ItemResponse) (*workflow.Block, error) {
	b := &workflow.Block{}

	var err error
	var resp blocksEntry
	err = json.Unmarshal(response.Value, &resp)
	if err != nil {
		return err
	}

	b.ID, err = uuid.Parse(resp.id)
	if err != nil {
		return fmt.Errorf("couldn't convert ID to UUID: %w", err)
	}

	k := resp.key
	if k != "" {
		b.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse block key: %w", err)
		}
	}
	b.Name = resp.Name
	b.Descr = resp.Descr
	b.EntranceDelay = time.Duration(resp.entranceDelay)
	b.ExitDelay = time.Duration(resp.exitDelay)
	b.State, err = fieldToState(resp.stateStatus, resp.stateStart, resp.stateEnd)
	if err != nil {
		return nil, fmt.Errorf("blockRowToBlock: %w", err)
	}
	b.Concurrency = int(resp.concurrency)
	b.ToleratedFailures = int(resp.toleratedFailures)
	b.BypassChecks, err = p.strToCheck(ctx, resp.bypassChecks)
	if err != nil {
		return fmt.Errorf("couldn't get block bypasschecks: %w", err)
	}
	b.PreChecks, err = p.strToCheck(ctx, resp.preChecks)
	if err != nil {
		return fmt.Errorf("couldn't get block prechecks: %w", err)
	}
	b.ContChecks, err = p.strToCheck(ctx, resp.contChecks)
	if err != nil {
		return fmt.Errorf("couldn't get block contchecks: %w", err)
	}
	b.PostChecks, err = p.strToCheck(ctx, resp.postChecks)
	if err != nil {
		return fmt.Errorf("couldn't get block postchecks: %w", err)
	}
	plan.DeferredChecks, err = p.strToCheck(ctx, resp.postChecks)
	if err != nil {
		return fmt.Errorf("couldn't get block deferredchecks: %w", err)
	}
	b.Sequences, err = p.fieldToSequences(ctx, conn, stmt)
	if err != nil {
		return nil, fmt.Errorf("couldn't read block sequences: %w", err)
	}

	return b, nil
}
