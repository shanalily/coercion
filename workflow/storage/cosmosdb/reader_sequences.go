package cosmosdb

import (
	"context"
	"fmt"

	"github.com/element-of-surprise/coercion/workflow"
	"github.com/google/uuid"
	// "zombiezen.com/go/cosmosdb"
	// "zombiezen.com/go/cosmosdb/cosmosdbx"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// fieldToSequences converts the "sequences" field in a cosmosdb row to a list of workflow.Sequences.
func (p reader) fieldToSequences(ctx context.Context) ([]*workflow.Sequence, error) {
	ids, err := fieldToIDs("sequences", stmt)
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
	sequence := &workflow.Sequence{}
	// do := func(conn *cosmosdb.Conn) (err error) {
	// 	err = cosmosdbx.Execute(
	// 		conn,
	// 		fetchSequencesByID,
	// 		&cosmosdbx.ExecOptions{
	// 			Named: map[string]interface{}{
	// 				"$id": id.String(),
	// 			},
	// 			ResultFunc: func(stmt *cosmosdb.Stmt) error {
	// 				sequence, err = p.sequenceRowToSequence(ctx, conn, stmt)
	// 				if err != nil {
	// 					return fmt.Errorf("couldn't convert row to sequence: %w", err)
	// 				}
	// 				return nil
	// 			},
	// 		},
	// 	)
	// 	if err != nil {
	// 		return fmt.Errorf("couldn't fetch sequence by id: %w", err)
	// 	}
	// 	return nil
	// }

	// if err := do(conn); err != nil {
	// 	return nil, fmt.Errorf("couldn't fetch sequence by id: %w", err)
	// }
	return sequence, nil
}

// sequenceRowToSequence converts a cosmosdb row to a workflow.Sequence.
func (p reader) sequenceRowToSequence(ctx context.Context) (*workflow.Sequence, error) {
	var err error
	s := &workflow.Sequence{}
	s.ID, err = fieldToID("id", stmt)
	if err != nil {
		return nil, fmt.Errorf("couldn't read block id: %w", err)
	}
	k := stmt.GetText("key")
	if k != "" {
		s.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse sequence key: %w", err)
		}
	}
	s.Name = stmt.GetText("name")
	s.Descr = stmt.GetText("descr")
	s.State, err = fieldToState(stmt)
	if err != nil {
		return nil, fmt.Errorf("sequenceRowToSequence: %w", err)
	}
	s.Actions, err = p.fieldToActions(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("couldn't read sequence actions: %w", err)
	}

	return s, nil
}
