package cosmosdb

import (
	"context"
	"fmt"

	"github.com/element-of-surprise/coercion/workflow"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// idsToSequences converts the "sequences" field in a cosmosdb row to a list of workflow.Sequences.
func (p reader) idsToSequences(ctx context.Context, sequenceIDs []uuid.UUID) ([]*workflow.Sequence, error) {
	sequences := make([]*workflow.Sequence, 0, len(sequenceIDs))
	for _, id := range sequenceIDs {
		sequence, err := p.fetchSequenceByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch sequence(%s)by id: %w", id, err)
		}
		sequences = append(sequences, sequence)
	}
	return sequences, nil
}

// fetchSequenceByID fetches a sequence by its id.
func (p reader) fetchSequenceByID(ctx context.Context, id uuid.UUID) (*workflow.Sequence, error) {
	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	res, err := p.cc.GetSequencesClient().ReadItem(ctx, p.cc.GetPK(), id.String(), itemOpt)
	if err != nil {
		// return p, fmt.Errorf("failed to read item through Cosmos DB API: %w", cosmosErr(err))
		return nil, fmt.Errorf("couldn't fetch sequence by id: %w", err)
	}
	return p.sequenceRowToSequence(ctx, &res)
}

// sequenceRowToSequence converts a cosmosdb row to a workflow.Sequence.
func (p reader) sequenceRowToSequence(ctx context.Context, response *azcosmos.ItemResponse) (*workflow.Sequence, error) {
	var err error
	var resp sequencesEntry
	err = json.Unmarshal(response.Value, &resp)
	if err != nil {
		return nil, err
	}

	s := &workflow.Sequence{}
	s.ID = resp.ID
	k := resp.Key
	if k != uuid.Nil {
		s.Key = k
	}
	s.Name = resp.Name
	s.Descr = resp.Descr
	s.State, err = fieldToState(resp.StateStatus, resp.StateStart, resp.StateEnd)
	if err != nil {
		return nil, fmt.Errorf("couldn't get sequence state: %w", err)
	}
	s.Actions, err = p.idsToActions(ctx, resp.Actions)
	if err != nil {
		return nil, fmt.Errorf("couldn't read sequence actions: %w", err)
	}

	return s, nil
}
