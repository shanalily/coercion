package cosmosdb

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/element-of-surprise/coercion/workflow"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	// "zombiezen.com/go/cosmosdb"
	// "zombiezen.com/go/cosmosdb/cosmosdbx"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// fieldToActions converts the "actions" field in a cosmosdb row to a list of workflow.Actions.
func (r reader) fieldToActions(ctx context.Context) ([]*workflow.Action, error) {
	ids, err := fieldToIDs("actions", stmt)
	if err != nil {
		return nil, fmt.Errorf("couldn't read action ids: %w", err)
	}

	actions, err := r.fetchActionsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch actions by ids: %w", err)
	}
	return actions, nil
}

// fetchActionsByIDs fetches a list of actions by their IDs.
func (r reader) fetchActionsByIDs(ctx context.Context, ids []uuid.UUID) ([]*workflow.Action, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	actions := make([]*workflow.Action, 0, len(ids))

	query, args := replaceWithIDs(fetchActionsByID, "$ids", ids)

	// err := cosmosdbx.Execute(
	// 	conn,
	// 	query,
	// 	&cosmosdbx.ExecOptions{
	// 		Args: args,
	// 		ResultFunc: func(stmt *cosmosdb.Stmt) error {
	// 			a, err := r.actionRowToAction(ctx, stmt)
	// 			if err != nil {
	// 				return fmt.Errorf("couldn't convert row to action: %w", err)
	// 			}
	// 			actions = append(actions, a)
	// 			return nil
	// 		},
	// 	},
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("couldn't fetch actions by ids: %w", err)
	// }
	return actions, nil
}

var emptyAttemptsJSON = []byte(`[]`)

// actionRowToAction converts a cosmosdb row to a workflow.Action.
func (r reader) actionRowToAction(ctx context.Context, response *azcosmos.ItemResponse) (*workflow.Action, error) {
	var err error
	var resp actionsEntry
	err = json.Unmarshal(response.Value, &resp)
	if err != nil {
		return err
	}

	var err error
	a := &workflow.Action{}
	a.ID, err = uuid.Parse(resp.id)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse action id: %w", err)
	}
	k := resp.key
	if k != "" {
		a.Key, err = uuid.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse action key: %w", err)
		}
	}
	a.Name = resp.name
	a.Descr = resp.desc
	a.Plugin = resp.plugin
	a.Timeout = time.Duration(resp.timeout)
	a.Retries = int(resp.retries)
	a.State, err = fieldToState(resp.stateStatus, resp.stateStart, resp.stateEnd)
	if err != nil {
		return fmt.Errorf("couldn't get action state: %w", err)
	}
	plug := r.reg.Plugin(a.Plugin)
	if plug == nil {
		return nil, fmt.Errorf("couldn't find plugin %s", a.Plugin)
	}

	b := fieldToBytes(resp.req)
	if len(b) > 0 {
		req := plug.Request()
		if req != nil {
			if reflect.TypeOf(req).Kind() != reflect.Pointer {
				if err := json.Unmarshal(b, &req); err != nil {
					return nil, fmt.Errorf("couldn't unmarshal request: %w", err)
				}
			} else {
				if err := json.Unmarshal(b, req); err != nil {
					return nil, fmt.Errorf("couldn't unmarshal request: %w", err)
				}
			}
			a.Req = req
		}
	}
	b = fieldToBytes(resp.attempts)
	if len(b) > 0 {
		a.Attempts, err = decodeAttempts(b, plug)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode attempts: %w", err)
		}
	}
	return a, nil
}
