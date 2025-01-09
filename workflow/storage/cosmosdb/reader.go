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

const (
	// beginning of query to list plans with a filter
	searchPlans = `SELECT p.id, p.groupID, p.name, p.descr, p.submitTime, p.stateStatus, p.stateStart, p.stateEnd FROM plans p WHERE`
	// list all plans without parameters
	// const listPlans = `SELECT id, groupID, name, descr, submitTime, stateStatus, stateStart, stateEnd FROM plans ORDER BY submitTime DESC`
	// can I list by submitTime without indexing?
	listPlans = `SELECT p.id, p.groupID, p.name, p.descr, p.submitTime, p.stateStatus, p.stateStart, p.stateEnd FROM plans p ORDER BY p.submitTime DESC`
)

// reader implements the storage.PlanReader interface.
type reader struct {
	cc  Client
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

	_, err := r.cc.GetPlansClient().ReadItem(ctx, r.cc.GetPK(), id.String(), itemOpt)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("couldn't fetch plan by id: %w", err)
	}
	return true, nil
}

// Read returns a Plan from the storage.
func (r reader) Read(ctx context.Context, id uuid.UUID) (*workflow.Plan, error) {
	return r.fetchPlan(ctx, id)
}

// Search returns a list of Plan IDs that match the filter.
func (r reader) Search(ctx context.Context, filters storage.Filters) (chan storage.Stream[storage.ListResult], error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	q, parameters := r.buildSearchQuery(filters)

	pager := r.cc.GetPlansClient().NewQueryItemsPager(q, r.cc.GetPK(), &azcosmos.QueryOptions{QueryParameters: parameters})
	resultsStream := make(chan storage.Stream[storage.ListResult])
	results := make([]storage.ListResult, 0)
	go func() {
		defer close(resultsStream)
		// need to check for ctx timeout
		for pager.More() {
			res, err := pager.NextPage(ctx)
			if err != nil {
				resultsStream <- storage.Stream[storage.ListResult]{
					Err: fmt.Errorf("problem listing plans: %w", err),
				}
				return
			}
			for _, item := range res.Items {
				result, err := r.listResultsFunc(item)
				if err != nil {
					resultsStream <- storage.Stream[storage.ListResult]{
						Err: fmt.Errorf("problem listing items in plans: %w", err),
					}
					return
				}
				results = append(results, result)
			}
		}

		// resultsStream a= make(chan storage.Stream[storage.ListResult], len(results))
		for _, r := range results {
			resultsStream <- storage.Stream[storage.ListResult]{Result: r}
		}
		return
	}()
	return resultsStream, nil
}

func (r reader) buildSearchQuery(filters storage.Filters) (string, []azcosmos.QueryParameter) {
	parameters := []azcosmos.QueryParameter{}

	build := strings.Builder{}
	build.WriteString(searchPlans)

	numFilters := 0

	if len(filters.ByIDs) > 0 {
		numFilters++
		build.WriteString(" ARRAY_CONTAINS(@ids, p.id)")
	}
	if len(filters.ByGroupIDs) > 0 {
		if numFilters > 0 {
			build.WriteString(" AND")
		}
		numFilters++
		build.WriteString(" ARRAY_CONTAINS(@group_ids, p.groupID)")
	}
	if len(filters.ByStatus) > 0 {
		if numFilters > 0 {
			build.WriteString(" AND")
		}
		numFilters++ // I know this says inEffectual assignment and it is, but it is here for completeness.
		for i, s := range filters.ByStatus {
			name := fmt.Sprintf("@status%d", i)
			if i == 0 {
				build.WriteString(fmt.Sprintf(" p.stateStatus = %s", name))
			} else {
				build.WriteString(fmt.Sprintf(" AND p.stateStatus = %s", name))
			}
			parameters = append(parameters, azcosmos.QueryParameter{
				Name:  name,
				Value: int64(s),
			})
		}
	}
	build.WriteString(" ORDER BY p.submitTime DESC")
	query := build.String()

	if len(filters.ByIDs) > 0 {
		parameters = append(parameters, azcosmos.QueryParameter{
			Name:  "@ids",
			Value: filters.ByIDs,
		})
	}
	if len(filters.ByGroupIDs) > 0 {
		parameters = append(parameters, azcosmos.QueryParameter{
			Name:  "@group_ids",
			Value: filters.ByGroupIDs,
		})
	}
	return query, parameters
}

// List returns a list of Plan IDs in the storage in order from newest to oldest. This should
// return with most recent submiited first. Limit sets the maximum number of
// entrie to return
func (r reader) List(ctx context.Context, limit int) (chan storage.Stream[storage.ListResult], error) {
	q := listPlans
	if limit > 0 {
		q += fmt.Sprintf(" OFFSET 0 LIMIT %d", limit)
	}

	pager := r.cc.GetPlansClient().NewQueryItemsPager(q, r.cc.GetPK(), &azcosmos.QueryOptions{QueryParameters: []azcosmos.QueryParameter{}})
	resultsStream := make(chan storage.Stream[storage.ListResult])
	results := make([]storage.ListResult, 0)
	go func() {
		defer close(resultsStream)
		for pager.More() {
			res, err := pager.NextPage(ctx)
			if err != nil {
				resultsStream <- storage.Stream[storage.ListResult]{
					Err: fmt.Errorf("problem listing plans: %w", err),
				}
				return
			}
			for _, item := range res.Items {
				result, err := r.listResultsFunc(item)
				if err != nil {
					resultsStream <- storage.Stream[storage.ListResult]{
						Err: fmt.Errorf("problem listing items in plans: %w", err),
					}
					return
				}
				results = append(results, result)
			}
		}

		for _, r := range results {
			resultsStream <- storage.Stream[storage.ListResult]{Result: r}
		}
	}()
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
	result.ID = resp.ID
	gid := resp.GroupID
	if gid == uuid.Nil {
		result.GroupID = uuid.Nil
	} else {
		result.GroupID = resp.GroupID
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

// fieldToState pulls the stateStart, stateEnd and stateStatus from a stmt
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
