package cosmosdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	// "sync/atomic"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/fake/server"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/go-json-experiment/json"
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
	// can I use this to make sure the right methods are called?
	server Server
	cc     *azcosmos.ContainerClient

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

func NewContainerClient() (*FakeContainerClient, error) {
	// first, create an instance of the fake server for the client you wish to test.
	// the type name of the server will be similar to the corresponding client, with
	// the suffix "Server" instead of "Client".
	fakeServer := Server{
		// next, provide implementations for the APIs you wish to fake.
		// this fake corresponds to the Client.Resources() API.
		ReadItem: func(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, o *azcosmos.ItemOptions) (resp azfake.Responder[azcosmos.ItemResponse], errResp azfake.ErrorResponder) {
			// the values of ctx, query, and options come from the API call.

			// the named return values resp and errResp are used to construct the response
			// and are meant to be mutually exclusive. if both responses have been constructed,
			// the error response is selected.

			properties := map[string]string{
				"id":   "id",
				"name": "name",
			}

			jsonString, err := json.Marshal(properties)
			if err != nil {
				errResp.SetResponseError(http.StatusInternalServerError, "ThisIsASimulatedError")
				return
			}
			sessionToken := "testSessionToken"
			// use resp to set the desired response
			queryResp := azcosmos.ItemResponse{
				Value:        jsonString,
				SessionToken: &sessionToken,
			}
			// queryResp.Count = to.Ptr[int64](12345)
			resp.SetResponse(http.StatusOK, queryResp, nil)

			// to simulate the failure case, use errResp
			// errResp.SetResponseError(http.StatusBadRequest, "ThisIsASimulatedError")

			return
		},
	}

	endpoint := "https://medbaydb.documents.azure.com"

	// now create the corresponding client, connecting the fake server via the client options
	client, err := azcosmos.NewClient(endpoint, &azfake.TokenCredential{}, &azcosmos.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: NewServerTransport(&fakeServer),
		},
	})
	if err != nil {
		return nil, err
	}
	dbName := "fakeDBName"
	containerName := "fakeContainerName"
	cc, err := client.NewContainer(dbName, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cosmos DB container client: container=%s. %w", containerName, err)
	}
	if err != nil {
		return nil, err
	}
	fakeContainerClient := FakeContainerClient{
		server: fakeServer,
		cc:     cc,
	}

	return &fakeContainerClient, nil
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

	// call the API. the provided values will be passed to the fake's implementation.
	// the response or error values returned by the API call are from the fake.
	resp, err := cc.cc.ReadItem(ctx, partitionKey, itemId, &azcosmos.ItemOptions{})
	if err != nil {
		return azcosmos.ItemResponse{}, fmt.Errorf("failed to read item: %s", err)
	}

	fmt.Println(resp)

	// APIs that haven't been faked will return an error
	// _, err = client.ResourcesHistory(context.TODO(), armresourcegraph.ResourcesHistoryRequest{}, nil)
	fmt.Println(err.Error())

	return resp, nil
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

// use fake server from this?
// https://github.com/Azure/azure-sdk-for-go/blob/sdk/data/azcosmos/v1.2.0/sdk/data/azcosmos/cosmos_container_response_test.go
// or maybe use this as an exmaple instead? https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/resourcemanager/resourcegraph/armresourcegraph/fake/server.go

// Server is a fake server for instances of the armresourcegraph.Client type.
type Server struct {
	CreateItem         func(ctx context.Context, partitionKey azcosmos.PartitionKey, item []byte, o *azcosmos.ItemOptions) (resp azfake.Responder[azcosmos.ItemResponse], errResp azfake.ErrorResponder)
	ReplaceItem        func(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, item []byte, o *azcosmos.ItemOptions) (resp azfake.Responder[azcosmos.ItemResponse], errResp azfake.ErrorResponder)
	DeleteItem         func(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, o *azcosmos.ItemOptions) (resp azfake.Responder[azcosmos.ItemResponse], errResp azfake.ErrorResponder)
	NewQueryItemsPager func(query string, partitionKey azcosmos.PartitionKey, o *azcosmos.QueryOptions) *runtime.Pager[azcosmos.QueryItemsResponse]
	Read               func(ctx context.Context, o *azcosmos.ReadContainerOptions) (resp azfake.Responder[azcosmos.ContainerResponse], errResp azfake.ErrorResponder)
	ReadItem           func(ctx context.Context, partitionKey azcosmos.PartitionKey, itemId string, o *azcosmos.ItemOptions) (resp azfake.Responder[azcosmos.ItemResponse], errResp azfake.ErrorResponder)
}

// NewServerTransport creates a new instance of ServerTransport with the provided implementation.
// The returned ServerTransport instance is connected to an instance of armresourcegraph.Client via the
// azcore.ClientOptions.Transporter field in the client's constructor parameters.
func NewServerTransport(srv *Server) *ServerTransport {
	return &ServerTransport{srv: srv}
}

// ServerTransport connects instances of armresourcegraph.Client to instances of Server.
// Don't use this type directly, use NewServerTransport instead.
type ServerTransport struct {
	srv *Server
}

// Do implements the policy.Transporter interface for ServerTransport.
func (s *ServerTransport) Do(req *http.Request) (*http.Response, error) {
	rawMethod := req.Context().Value(runtime.CtxAPINameKey{})
	method, ok := rawMethod.(string)
	if !ok {
		return nil, nonRetriableError{errors.New("unable to dispatch request, missing value for CtxAPINameKey")}
	}

	var resp *http.Response
	var err error

	switch method {
	case "Client.CreateItem":
		resp, err = s.dispatchReadItem(req)
	case "Client.ReplaceItem":
		resp, err = s.dispatchReadItem(req)
	case "Client.DeleteItem":
		resp, err = s.dispatchReadItem(req)
	case "Client.NewQueryItemsPager":
		resp, err = s.dispatchReadItem(req)
	case "Client.Read":
		resp, err = s.dispatchReadItem(req)
	case "Client.ReadItem":
		resp, err = s.dispatchReadItem(req)
	default:
		err = fmt.Errorf("unhandled API %s", method)
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *ServerTransport) dispatchReadItem(req *http.Request) (*http.Response, error) {
	fmt.Println(req)

	if s.srv.ReadItem == nil {
		return nil, &nonRetriableError{errors.New("fake for method ReadItem not implemented")}
	}
	// body, err := server.UnmarshalRequestAsJSON[azcosmos.ReadItemRequest](req)
	// if err != nil {
	// 	return nil, err
	// }
	// respr, errRespr := s.srv.ReadItem(req.Context(), body, nil)
	respr, errRespr := s.srv.ReadItem(req.Context(), partitionKey("val"), "itemid", nil)
	if respErr := server.GetError(errRespr, req); respErr != nil {
		return nil, respErr
	}
	respContent := server.GetResponseContent(respr)
	if !contains([]int{http.StatusOK}, respContent.HTTPStatus) {
		return nil, &nonRetriableError{fmt.Errorf("unexpected status code %d. acceptable values are http.StatusOK", respContent.HTTPStatus)}
	}
	resp, err := server.MarshalResponseAsJSON(respContent, server.GetResponse(respr), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type nonRetriableError struct {
	error
}

func (nonRetriableError) NonRetriable() {
	// marker method
}

func contains[T comparable](s []T, v T) bool {
	for _, vv := range s {
		if vv == v {
			return true
		}
	}
	return false
}

func newTracker[T any]() *tracker[T] {
	return &tracker[T]{
		items: map[string]*T{},
	}
}

type tracker[T any] struct {
	items map[string]*T
	mu    sync.Mutex
}

func (p *tracker[T]) get(req *http.Request) *T {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item, ok := p.items[server.SanitizePagerPollerPath(req.URL.Path)]; ok {
		return item
	}
	return nil
}

func (p *tracker[T]) add(req *http.Request, item *T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[server.SanitizePagerPollerPath(req.URL.Path)] = item
}

func (p *tracker[T]) remove(req *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, server.SanitizePagerPollerPath(req.URL.Path))
}
