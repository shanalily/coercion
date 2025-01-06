package cosmosdb

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	// "sync/atomic"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	// "github.com/samber/lo"
	// "google.golang.org/protobuf/proto"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	// "go.goms.io/aks/rp/fleet/pkg/cosmosdb"
	// "go.goms.io/aks/rp/fleet/pkg/cosmosdb/datamodel"
)

// https://dev.azure.com/msazure/CloudNativeCompute/_git/aks-rp?path=/fleet/pkg/cosmosdb/testing/pager.go

//+gocover:ignore:file No need to test mock store.

var ErrCosmosDBNotFound error = fmt.Errorf("cosmosdb error: %w", &azcore.ResponseError{StatusCode: http.StatusNotFound})

// needs to implement cosmosdb.go client interface

// CosmosDBClient has the methods for all of Create/Update/Delete/Query operation
// on data model.
type FakeCosmosDBClient struct {
	// publisher arg.Publisher

	// implement types per client later? just use containerclient for now
	// is there no interface for the cosmosdb container client? I need functions like ReadItem.
	// Maybe this is part of why fleet needed to wrap things in custom clients?
	// I only need some of the methods if I wrap client:
	// CreateItem, DeleteItem, NewQueryItemsPager, ReadItem, ReplaceItem, maybe eothers
	plansClient     FakeContainerClient
	blocksClient    FakeContainerClient
	checksClient    FakeContainerClient
	sequencesClient FakeContainerClient
	actionsClient   FakeContainerClient
}

// type FakeContainerClient[C datamodel.CosmosDataModel, P datamodel.ProtoDataModel] struct {
// should p actually be a schema entry type?
type FakeContainerClient struct {
	mu sync.Mutex

	CreateInput string //P
	CreateErr   error

	UpdateCallCount int
	UpdateInput     string   //P
	UpdateInputs    []string //P
	UpdateErr       error
	UpdateErrs      []error

	DeleteInput    string //P
	DeleteInputMap map[string]struct{}
	DeleteErr      error

	// NewListItemsRawPagerResult datamodel.Pager[C]
	// NewListItemsRawPagerResult Pager[azcosmos.QueryItemsResponse]
	NewListItemsRawPagerResult *runtime.Pager[azcosmos.QueryItemsResponse]
}

// func (cc FakeContainerClient) CreateItem(ctx context.Context, p string) error {
func (cc FakeContainerClient) CreateItem(ctx context.Context, partitionKey azcosmos.PartitionKey, item []byte, o *azcosmos.ItemOptions) (azcosmos.ItemResponse, error) {
	cc.CreateInput = string(item) // p

	return azcosmos.ItemResponse{}, cc.CreateErr
}

// func (cc FakeContainerClient) UpdateItem(ctx context.Context, p string) (err error) {
func (cc FakeContainerClient) ReplaceItem(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, item []byte, o *azcosmos.ItemOptions) (azcosmos.ItemResponse, error) {
	cc.UpdateCallCount++
	// // cc.UpdateInput = p
	// cc.UpdateInput =  string(item)

	// // p2, ok := proto.Clone(p).(string)
	// // if !ok {
	// // 	return azcosmos.ItemResponse{}, fmt.Errorf("clone failed")
	// // }
	// p2 := string(item)

	// cc.UpdateInputs = append(cc.UpdateInputs, p2)

	// if len(cc.UpdateErrs) > 0 {
	// 	err, cc.UpdateErrs = cc.UpdateErrs[0], cc.UpdateErrs[1:]
	// 	return azcosmos.ItemResponse{}, err
	// }

	return azcosmos.ItemResponse{}, cc.UpdateErr
}

// func (cc FakeContainerClient) DeleteItem(ctx context.Context, p string) error {
func (cc FakeContainerClient) DeleteItem(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, o *azcosmos.ItemOptions) (azcosmos.ItemResponse, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.DeleteInputMap == nil {
		cc.DeleteInputMap = make(map[string]struct{})
	}

	// cc.DeleteInput = p
	// cc.DeleteInputMap[p] = struct{}{}

	return azcosmos.ItemResponse{}, cc.DeleteErr
}

// func (cc FakeContainerClient) NewListItemsRawPager() datamodel.Pager[C] {
func (cc FakeContainerClient) NewQueryItemsPager(query string, partitionKey azcosmos.PartitionKey, o *azcosmos.QueryOptions) *runtime.Pager[azcosmos.QueryItemsResponse] {
	return cc.NewListItemsRawPagerResult
}

func (cc FakeContainerClient) Read(ctx context.Context, o *azcosmos.ReadContainerOptions) (azcosmos.ContainerResponse, error) {
	return azcosmos.ContainerResponse{}, nil
}

func (cc FakeContainerClient) ReadItem(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, o *azcosmos.ItemOptions) (azcosmos.ItemResponse, error) {
	return azcosmos.ItemResponse{}, nil
}

func (c *FakeCosmosDBClient) GetPlansClient() ContainerClient {
	return c.plansClient
}

func (c *FakeCosmosDBClient) GetBlocksClient() ContainerClient {
	return c.blocksClient
}

func (c *FakeCosmosDBClient) GetChecksClient() ContainerClient {
	return c.checksClient
}

func (c *FakeCosmosDBClient) GetSequencesClient() ContainerClient {
	return c.sequencesClient
}

func (c *FakeCosmosDBClient) GetActionsClient() ContainerClient {
	return c.actionsClient
}

type Pager[T any] struct {
	Pages []Page[T]
}

type Page[T any] struct {
	Values []T
	Err    error
}

// https://dev.azure.com/msazure/CloudNativeCompute/_git/aks-rp?path=/fleet/pkg/cosmosdb/testing/pager.go
func (pager *Pager[T]) NextPage(ctx context.Context) ([]T, error) {
	values, err := pager.Pages[0].Values, pager.Pages[0].Err
	pager.Pages = pager.Pages[1:]
	return values, err
}

func (pager *Pager[T]) More() bool {
	return len(pager.Pages) > 0
}
