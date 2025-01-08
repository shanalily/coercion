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

// fieldToSequences converts the "sequences" field in a cosmosdb row to a list of workflow.Sequences.
func (p reader) strToSequences(ctx context.Context, sequenceIDs string) ([]*workflow.Sequence, error) {
	ids, err := strToIDs(sequenceIDs)
	if err != nil {
		return nil, fmt.Errorf("couldn't read plan sequence ids: %w", err)
	}

	sequences := make([]*workflow.Sequence, 0, len(ids))
	for _, id := range ids {
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
	s.ID, err = uuid.Parse(resp.id)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse sequence id: %w", err)
	}
	k := resp.key
	if k != "" {
		s.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse sequence key: %w", err)
		}
	}
	s.Name = resp.name
	s.Descr = resp.descr
	s.State, err = fieldToState(resp.stateStatus, resp.stateStart, resp.stateEnd)
	if err != nil {
		return nil, fmt.Errorf("couldn't get sequence state: %w", err)
	}
	s.Actions, err = p.strToActions(ctx, resp.actions)
	if err != nil {
		return nil, fmt.Errorf("couldn't read sequence actions: %w", err)
	}

	return s, nil
}
