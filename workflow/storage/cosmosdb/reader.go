package cosmosdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unsafe"

	"github.com/element-of-surprise/coercion/plugins/registry"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
)

// reader implements the storage.PlanReader interface.
type reader struct {
	cc  *CosmosDBClient
	reg *registry.Register
}

// IsNotFound checks if the error that Azure returned is 404.
func IsNotFound(err error) bool {
	var resErr *azcore.ResponseError
	return errors.As(err, &resErr) && resErr.StatusCode == http.StatusNotFound
}

// IsNotFound checks if the error indicates there is a resource conflict.
// Useful to check if a resource already exists in testing.
func IsConflict(err error) bool {
	var resErr *azcore.ResponseError
	return errors.As(err, &resErr) && resErr.StatusCode == http.StatusConflict
}

// Exists returns true if the Plan ID exists in the storage.
func (r reader) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var itemOpt = &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}

	_, err := r.cc.plansClient.ReadItem(ctx, r.cc.partitionKey, id.String(), itemOpt)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		// return p, fmt.Errorf("failed to read item through Cosmos DB API: %w", cosmosErr(err))
		return false, fmt.Errorf("couldn't fetch block by id: %w", err)
	}
	return true, nil
}

// ReadPlan returns a Plan from the storage.
func (r reader) Read(ctx context.Context, id uuid.UUID) (*workflow.Plan, error) {
	return r.fetchPlan(ctx, id)
}

// SearchPlans returns a list of Plan IDs that match the filter.
func (r reader) Search(ctx context.Context, filters storage.Filters) (chan storage.Stream[storage.ListResult], error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	q, parameters := r.buildSearchQuery(filters)

	// results := make(chan storage.Stream[storage.ListResult], 1)
	pager := r.cc.GetPlansClient().NewQueryItemsPager(q, r.cc.partitionKey, &azcosmos.QueryOptions{QueryParameters: parameters})
	// results := make(chan storage.Stream[storage.ListResult], 1)
	resultsStream := make(chan storage.Stream[storage.ListResult])
	// results := make([]any, 0)
	results := make([]storage.ListResult, 0)
	for pager.More() {
		res, err := pager.NextPage(ctx)
		if err != nil {
			resultsStream <- storage.Stream[storage.ListResult]{
				Err: err,
			}
			return resultsStream, err
		}
		for _, item := range res.Items {
			result, err := r.listResultsFunc(item)
			if err != nil {
				resultsStream <- storage.Stream[storage.ListResult]{
					Err: err,
				}
				return resultsStream, fmt.Errorf("problem listing plans: %w", err)
			}
			results = append(results, result)
		}
	}

	// resultsStream a= make(chan storage.Stream[storage.ListResult], len(results))
	for _, r := range results {
		resultsStream <- storage.Stream[storage.ListResult]{Result: r}
	}

	// go func() {
	// 	err := cosmosdbx.Execute(
	// 		conn,
	// 		q,
	// 		&cosmosdbx.ExecOptions{
	// 			Args:  args,
	// 			Named: named,
	// 			ResultFunc: func(stmt *cosmosdb.Stmt) error {
	// 				r, err := r.listResultsFunc(stmt)
	// 				if err != nil {
	// 					return fmt.Errorf("problem searching plans: %w", err)
	// 				}
	// 				select {
	// 				case <-ctx.Done():
	// 					results <- storage.Stream[storage.ListResult]{
	// 						Err: ctx.Err(),
	// 					}
	// 					return ctx.Err()
	// 				case results <- storage.Stream[storage.ListResult]{Result: r}:
	// 					return nil
	// 				}
	// 			},
	// 		},
	// 	)

	// 	if err != nil {
	// 		results <- storage.Stream[storage.ListResult]{Err: fmt.Errorf("couldn't complete list plans: %w", err)}
	// 	}
	// }()
	return resultsStream, nil
}

func (r reader) buildSearchQuery(filters storage.Filters) (string, []azcosmos.QueryParameter) {
	parameters := []azcosmos.QueryParameter{}

	const sel = `SELECT id, group_id, name, descr, submit_time, state_status, state_start, state_end FROM plans WHERE`

	build := strings.Builder{}
	build.WriteString(sel)

	numFilters := 0

	if len(filters.ByIDs) > 0 {
		numFilters++
		// ARRAY_CONTAINS(@ids, id) instead? I'm not sure is IN and $ids will work with params struct
		build.WriteString(" id IN @ids")
	}
	if len(filters.ByGroupIDs) > 0 {
		if numFilters > 0 {
			build.WriteString(" AND")
		}
		numFilters++
		build.WriteString(" group_id IN @group_ids")
	}
	if len(filters.ByStatus) > 0 {
		if numFilters > 0 {
			build.WriteString(" AND")
		}
		numFilters++ // I know this says inEffectual assignment and it is, but it is here for completeness.
		for i, s := range filters.ByStatus {
			name := fmt.Sprintf("$status%d", i)
			// named[name] = int64(s)
			if i == 0 {
				build.WriteString(fmt.Sprintf(" state_status = %s", name))
			} else {
				build.WriteString(fmt.Sprintf(" AND state_status = %s", name))
			}
			parameters = append(parameters, azcosmos.QueryParameter{
				Name:  "@status",
				Value: int64(s),
			})
		}
	}
	build.WriteString(" ORDER BY submit_time DESC;")
	query := build.String()

	if len(filters.ByIDs) > 0 {
		var idArgs []any
		_, idArgs = replaceWithIDs(query, "$id", filters.ByIDs)
		// args = append(args, idArgs...)
		parameters = append(parameters, azcosmos.QueryParameter{
			Name:  "@ids",
			Value: idArgs,
		})
	}
	if len(filters.ByGroupIDs) > 0 {
		var groupArgs []any
		// _, groupArgs = replaceWithIDs(query, "$group_id", filters.ByGroupIDs)
		_, groupArgs = replaceWithIDs(query, "$group_id", filters.ByGroupIDs)
		// args = append(args, groupArgs...)
		parameters = append(parameters, azcosmos.QueryParameter{
			Name:  "$group_ids",
			Value: groupArgs,
		})
	}
	return query, parameters // args, named
}

// List returns a list of Plan IDs in the storage in order from newest to oldest. This should
// return with most recent submiited first. Limit sets the maximum number of
// entrie to return
func (r reader) List(ctx context.Context, limit int) (chan storage.Stream[storage.ListResult], error) {
	// list all plans without parameters
	const listPlans = `SELECT id, group_id, name, descr, submit_time, state_status, state_start, state_end FROM plans ORDER BY submit_time DESC`

	named := map[string]any{}

	q := listPlans
	if limit > 0 {
		q += " LIMIT $limit;"
		named["$limit"] = limit
	}

	pager := r.cc.GetPlansClient().NewQueryItemsPager(q, r.cc.partitionKey, &azcosmos.QueryOptions{QueryParameters: []azcosmos.QueryParameter{}})
	// results := make(chan storage.Stream[storage.ListResult], 1)
	resultsStream := make(chan storage.Stream[storage.ListResult])
	// results := make([]any, 0)
	results := make([]storage.ListResult, 0)
	for pager.More() {
		res, err := pager.NextPage(ctx)
		if err != nil {
			resultsStream <- storage.Stream[storage.ListResult]{
				Err: err,
			}
			return resultsStream, err
		}
		for _, item := range res.Items {
			result, err := r.listResultsFunc(item)
			if err != nil {
				resultsStream <- storage.Stream[storage.ListResult]{
					Err: err,
				}
				return resultsStream, fmt.Errorf("problem listing plans: %w", err)
			}
			results = append(results, result)
		}
	}

	// resultsStream a= make(chan storage.Stream[storage.ListResult], len(results))
	for _, r := range results {
		resultsStream <- storage.Stream[storage.ListResult]{Result: r}
	}

	// results := make(chan storage.Stream[storage.ListResult], 1)
	// go func() {
	// 		result, err := r.listResultsFunc(stmt)
	// 		if err != nil {
	// 			return fmt.Errorf("problem listing plans: %w", err)
	// 		}
	// 		select {
	// 		case <-ctx.Done():
	// 			results <- storage.Stream[storage.ListResult]{
	// 				Err: ctx.Err(),
	// 			}
	// 			return ctx.Err()
	// 		case results <- storage.Stream[storage.ListResult]{Result: result}:
	// 			return nil
	// 		}

	// 	if err != nil {
	// 		results <- storage.Stream[storage.ListResult]{Err: fmt.Errorf("couldn't complete list plans: %w", err)}
	// 	}
	// }()
	return resultsStream, nil
}

// listResultsFunc is a helper function to convert a SQLite statement into a ListResult.
func (r reader) listResultsFunc(item []byte) (storage.ListResult, error) {
	// still need to iterate through items here: response has Items of type [][]byte
	var err error
	var resp plansEntry
	err = json.Unmarshal(item, &resp)
	if err != nil {
		return storage.ListResult{}, err
	}

	result := storage.ListResult{}
	result.ID, err = uuid.Parse(resp.ID)
	if err != nil {
		return result, fmt.Errorf("couldn't convert ID to UUID: %w", err)
	}
	gid := resp.GroupID
	if gid == "" {
		result.GroupID = uuid.Nil
	} else {
		result.GroupID, err = uuid.Parse(resp.GroupID)
		if err != nil {
			return result, fmt.Errorf("couldn't convert GroupID to UUID: %w", err)
		}
	}
	result.Name = resp.Name
	result.Descr = resp.Descr
	result.SubmitTime = time.Unix(0, resp.SubmitTime)
	result.State = &workflow.State{
		Status: workflow.Status(resp.StateStatus),
		Start:  time.Unix(0, resp.StateStart),
		End:    time.Unix(0, resp.StateEnd),
	}
	return result, nil
}

// // listResultsFunc is a helper function to convert a SQLite statement into a ListResult.
// func (r reader) listResultsFunc(resp azcosmos.QueryItemsResponse) (storage.ListResult, error) {
// 	// still need to iterate through items here: response has Items of type [][]byte
// 	result := storage.ListResult{}
// 	var err error
// 	result.ID, err = fieldToID("id", stmt)
// 	if err != nil {
// 		return storage.ListResult{}, fmt.Errorf("couldn't get ID: %w", err)
// 	}
// 	result.GroupID, err = fieldToID("group_id", stmt)
// 	if err != nil {
// 		return storage.ListResult{}, fmt.Errorf("couldn't get group ID: %w", err)
// 	}
// 	result.Name = stmt.GetText("name")
// 	result.Descr = stmt.GetText("descr")
// 	result.SubmitTime = time.Unix(0, stmt.GetInt64("submit_time"))
// 	result.State = &workflow.State{
// 		Status: workflow.Status(stmt.GetInt64("state_status")),
// 		Start:  time.Unix(0, stmt.GetInt64("state_start")),
// 		End:    time.Unix(0, stmt.GetInt64("state_end")),
// 	}
// 	return result, nil
// }

func (r reader) private() {
	return
}

// fieldToID returns a uuid.UUID from a field "field" in the Stmt that must be a TEXT field.
func fieldToID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

// fieldToIDs returns the IDs from the statement field. Field must the a blob
// encoded as a JSON array that has string UUIDs in v7 format.
func strToIDs(id string) ([]uuid.UUID, error) {
	contents := strToBytes(id)
	if contents == nil {
		return nil, fmt.Errorf("actions IDs are nil")
	}
	strIDs := []string{}
	if err := json.Unmarshal(contents, &strIDs); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal action ids: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(strIDs))
	for _, id := range strIDs {
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse id(%s): %w", id, err)
		}
		ids = append(ids, u)
	}

	return ids, nil
}

func strToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// // fieldToBytes returns the bytes of the field from the statement.
// func fieldToBytes(val any) []byte {
// 	b := make([]byte, val)
// 	return b
// }

func timeFromInt64(unixTime int64) (time.Time, error) {
	t := time.Unix(0, unixTime)
	if t.Before(zeroTime) {
		return time.Time{}, nil
	}
	return t, nil
}

// fieldToState pulls the state_start, state_end and state_status from a stmt
// and turns them into a *workflow.State.
func fieldToState(stateStatus, stateStart, stateEnd int64) (*workflow.State, error) {
	start, err := timeFromInt64(stateStart)
	if err != nil {
		return nil, err
	}
	end, err := timeFromInt64(stateEnd)
	if err != nil {
		return nil, err
	}
	return &workflow.State{
		Status: workflow.Status(stateStatus),
		Start:  start,
		End:    end,
	}, nil
}
