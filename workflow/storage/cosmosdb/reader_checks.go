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
	var check *workflow.Checks
	// do := func(conn *cosmosdb.Conn) (err error) {
	// 	err = cosmosdbx.Execute(
	// 		conn,
	// 		fetchChecksByID,
	// 		&cosmosdbx.ExecOptions{
	// 			Named: map[string]any{
	// 				"$id": id.String(),
	// 			},
	// 			ResultFunc: func(stmt *cosmosdb.Stmt) error {
	// 				c, err := p.checksRowToChecks(ctx, conn, stmt)
	// 				if err != nil {
	// 					return fmt.Errorf("couldn't convert row to checks: %w", err)
	// 				}
	// 				check = c
	// 				return nil
	// 			},
	// 		},
	// 	)
	// 	if err != nil {
	// 		return fmt.Errorf("couldn't fetch checks by id: %w", err)
	// 	}
	// 	return nil
	// }
	// if err := do(conn); err != nil {
	// 	return nil, fmt.Errorf("couldn't fetch checks by ids: %w", err)
	// }
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
		return err
	}

	c := &workflow.Checks{}
	c.ID, err = uuid.Parse(resp.id)
	if err != nil {
		return nil, fmt.Errorf("checksRowToChecks: couldn't convert ID to UUID: %w", err)
	}
	k := resp.Key
	if k != "" {
		c.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse check key: %w", err)
		}
	}
	c.Delay = time.Duration(resp.delay)
	c.State, err = strToState(stmt)
	if err != nil {
		return nil, fmt.Errorf("checksRowToChecks: %w", err)
	}
	// c.Actions, err = p.strToActions(ctx, conn, stmt)
	// if err != nil {
	// 	return nil, fmt.Errorf("couldn't get actions ids: %w", err)
	// }

	return c, nil
}
