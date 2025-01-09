/*
Package cosmosdb provides a cosmosdb-based storage implementation for workflow.Plan data. This is used
to implement the storage.ReadWriter interface.

This package is for use only by the coercion.Workstream and any use outside of that is not
supported.
*/
package cosmosdb

import (
	// "strings"
	"context"
	"fmt"
	"log/slog"
	// "path/filepath"
	"sync"

	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/plugins/registry"
	"github.com/element-of-surprise/coercion/workflow/storage"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	_ "embed"
)

// This validates that the ReadWriter type implements the storage.ReadWriter interface.
var _ storage.Vault = &Vault{}

// Vault implements the storage.Vault interface.
type Vault struct {
	// dbName is the database name for the storage.
	dbName string
	// partitionKey is the partition key for the starage.
	// This assumes the service will use a single partition.
	partitionKey string

	reader
	creator
	updater
	closer
	deleter

	private.Storage
}

// Option is an option for configuring a ReadWriter.
type Option func(*Vault) error

// // WithInMemory creates an in-memory storage.
// func WithInMemory() Option {
// 	return func(r *Vault) error {
// 		r.openFlags = append(r.openFlags, cosmosdb.OpenMemory)
// 		return nil
// 	}
// }

// ContainerClient allows for faking the azcosmos container client.
type ContainerClient interface {
	CreateItem(context.Context, azcosmos.PartitionKey, []byte, *azcosmos.ItemOptions) (azcosmos.ItemResponse, error)
	DeleteItem(context.Context, azcosmos.PartitionKey, string, *azcosmos.ItemOptions) (azcosmos.ItemResponse, error)
	NewQueryItemsPager(string, azcosmos.PartitionKey, *azcosmos.QueryOptions) *runtime.Pager[azcosmos.QueryItemsResponse]
	Read(context.Context, *azcosmos.ReadContainerOptions) (azcosmos.ContainerResponse, error)
	ReadItem(context.Context, azcosmos.PartitionKey, string, *azcosmos.ItemOptions) (azcosmos.ItemResponse, error)
	ReplaceItem(context.Context, azcosmos.PartitionKey, string, []byte, *azcosmos.ItemOptions) (azcosmos.ItemResponse, error)
	PatchItem(context.Context, azcosmos.PartitionKey, string, azcosmos.PatchOperations, *azcosmos.ItemOptions) (azcosmos.ItemResponse, error)
}

func partitionKey(val string) azcosmos.PartitionKey {
	return azcosmos.NewPartitionKeyString(val)
}

type Client interface {
	GetPlansClient() ContainerClient     //*azcosmos.ContainerClient
	GetBlocksClient() ContainerClient    //*azcosmos.ContainerClient
	GetChecksClient() ContainerClient    // *azcosmos.ContainerClient
	GetSequencesClient() ContainerClient //*azcosmos.ContainerClient
	GetActionsClient() ContainerClient   //*azcosmos.ContainerClient
	GetPK() azcosmos.PartitionKey
	GetPKString() string
}

// CosmosDBClient has the methods for all of Create/Update/Delete/Query operation
// on data model.
type CosmosDBClient struct {
	partitionKey string

	// implement types per client later? just use containerclient for now
	plansClient     *azcosmos.ContainerClient
	blocksClient    *azcosmos.ContainerClient
	checksClient    *azcosmos.ContainerClient
	sequencesClient *azcosmos.ContainerClient
	actionsClient   *azcosmos.ContainerClient
}

// type Client[C datamodel.CosmosDataModel, P datamodel.ProtoDataModel] interface {
// 	CreateItem(ctx context.Context, item P) error
// 	UpdateItem(ctx context.Context, item P) error
// 	DeleteItem(ctx context.Context, item P) error
// 	NewListItemsRawPager() datamodel.Pager[C]
// }

func (c *CosmosDBClient) GetPlansClient() ContainerClient {
	return c.plansClient
}

func (c *CosmosDBClient) GetBlocksClient() ContainerClient {
	return c.blocksClient
}

func (c *CosmosDBClient) GetChecksClient() ContainerClient {
	return c.checksClient
}

func (c *CosmosDBClient) GetSequencesClient() ContainerClient {
	return c.sequencesClient
}

func (c *CosmosDBClient) GetActionsClient() ContainerClient {
	return c.actionsClient
}

func (c *CosmosDBClient) GetPK() azcosmos.PartitionKey {
	return partitionKey(c.partitionKey)
}

func (c *CosmosDBClient) GetPKString() string {
	return c.partitionKey
}

type container struct {
	name string
	init func(*azcosmos.ContainerClient)
	// only used if creating containers in test environment
	indexPaths []azcosmos.IncludedPath
}

func pathToScalar(path string) azcosmos.IncludedPath {
	return azcosmos.IncludedPath{
		Path: fmt.Sprintf("/%s/?", path),
	}
}

func (c *CosmosDBClient) containers() []container {
	cont := []container{
		{
			name: "plans",
			init: func(cc *azcosmos.ContainerClient) {
				c.plansClient = cc
			},
			indexPaths: []azcosmos.IncludedPath{
				pathToScalar("groupID"),
				pathToScalar("stateStatus"),
				pathToScalar("stateStart"),
				pathToScalar("stateEnd"),
				pathToScalar("reason"),
				// submitTime? do I need a composite index for more efficient ordering?
				pathToScalar("submitTime"),
			},
		},
		{
			name: "blocks",
			init: func(cc *azcosmos.ContainerClient) {
				c.blocksClient = cc
			},
			indexPaths: []azcosmos.IncludedPath{
				pathToScalar("key"),
				pathToScalar("planID"),
				pathToScalar("stateStatus"),
				pathToScalar("stateStart"),
				pathToScalar("stateEnd"),
			},
		},
		{
			name: "checks",
			init: func(cc *azcosmos.ContainerClient) {
				c.checksClient = cc
			},
			indexPaths: []azcosmos.IncludedPath{
				pathToScalar("key"),
				pathToScalar("planID"),
				pathToScalar("stateStatus"),
				pathToScalar("stateStart"),
				pathToScalar("stateEnd"),
			},
		},
		{
			name: "sequences",
			init: func(cc *azcosmos.ContainerClient) {
				c.sequencesClient = cc
			},
			indexPaths: []azcosmos.IncludedPath{
				// pathToScalar("id"),
				pathToScalar("key"),
				pathToScalar("planID"),
				pathToScalar("stateStatus"),
				pathToScalar("stateStart"),
				pathToScalar("stateEnd"),
			},
		},
		{
			name: "actions",
			init: func(cc *azcosmos.ContainerClient) {
				c.actionsClient = cc
			},
			indexPaths: []azcosmos.IncludedPath{
				// pathToScalar("id"),
				pathToScalar("key"),
				pathToScalar("planID"),
				pathToScalar("stateStatus"),
				pathToScalar("stateStart"),
				pathToScalar("stateEnd"),
				pathToScalar("plugin"),
				// do I need a composite index?
				pathToScalar("pos"),
			},
		},
	}

	return cont
}

// New is the constructor for *ReadWriter. root is the root path for the storage.
// If the root path does not exist, it will be created.
// Instead of sqlite package, create client to existing cosmosdb.
func New(ctx context.Context, endpoint, dbName, pk string, cred azcore.TokenCredential, reg *registry.Register, options ...Option) (*Vault, error) {
	ctx = context.WithoutCancel(ctx)

	r := &Vault{
		dbName:       dbName,
		partitionKey: pk,
	}
	for _, o := range options {
		if err := o(r); err != nil {
			return nil, err
		}
	}

	// o =
	client, err := azcosmos.NewClient(endpoint, cred, nil)
	// client, err := azcosmos.NewClientWithKey(endpoint, cosmosDBCred, nil)
	if err != nil {
		return nil, err
	}

	// create container for customer? or container per table for all customers?
	// multiple containers per customer?
	// need containers/tables for plans, blocks, checks, sequences, and actions.
	cc, err := createContainerClients(ctx, dbName, endpoint, pk, client)
	if err != nil {
		return nil, err
	}

	// create pool to limit clients connections or nah?

	mu := &sync.Mutex{}

	r.reader = reader{cc: cc, reg: reg}
	r.creator = creator{mu: mu, cc: cc, reader: r.reader}
	r.updater = newUpdater(mu, cc)
	r.closer = closer{cc: cc}
	r.deleter = deleter{mu: mu, cc: cc, reader: r.reader}
	return r, nil
}

// Use this function to create a new CosmosDBClient struct.
// dbEndpoint - the Cosmos DB's https endpoint
func createContainerClients(
	ctx context.Context,
	dbName, dbEndpoint, pk string,
	azCosmosClient *azcosmos.Client) (*CosmosDBClient, error) {

	client := &CosmosDBClient{
		partitionKey: pk,
	}

	dc, err := azCosmosClient.NewDatabase(dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cosmos DB database client: %w", err)
	}

	// TODO (sehobbs) - Consider making container creation a concurrent operation.
	for _, c := range client.containers() {
		// what if already created?
		activityID, err := createContainer(ctx, dc, c.name, c.indexPaths)
		if err != nil {
			switch {
			case IsConflict(err):
				slog.Default().Warn(fmt.Sprintf("Container %s already exists: %s", c.name, err))
			default:
				return nil, fmt.Errorf("failed to create Cosmos DB container: container=%s. %w", c.name, err)
			}
		} else {
			slog.Default().Info(activityID)
		}
		// activityID, err := recreateContainer(ctx, azCosmosClient, dbName, dbEndpoint, c.name, c.indexPaths)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to create Cosmos DB container: container=%s. %w", c.name, err)
		// } else {
		// 	slog.Default().Info(activityID)
		// }

		cc, err := azCosmosClient.NewContainer(dbName, c.name)
		if err != nil {
			return nil, fmt.Errorf("failed to create Cosmos DB container client: container=%s. %w", c.name, err)
		}

		c.init(cc)

		if _, err = cc.Read(ctx, nil); err != nil {
			return nil, fmt.Errorf(
				"failed to connect to Cosmos DB container: endpoint=%q, container=%q. %w",
				dbEndpoint,
				c.name,
				err,
			)
		}
	}

	return client, nil
}

func createContainerClient(client *azcosmos.Client, dbName, containerName string) (*azcosmos.ContainerClient, error) {
	container, err := client.NewContainer(dbName, containerName)
	return container, err
}

// For creating if it doesn't exist? should probably get precreated for multiple customers in bicep template though.
func createContainer(ctx context.Context, database *azcosmos.DatabaseClient, cname string, indexPaths []azcosmos.IncludedPath) (string, error) {
	properties := azcosmos.ContainerProperties{
		ID: cname,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/partitionKey"},
		},
		IndexingPolicy: &azcosmos.IndexingPolicy{
			IncludedPaths: indexPaths,
			// exclude by default
			ExcludedPaths: []azcosmos.ExcludedPath{
				{
					Path: "/*",
				},
			},
			Automatic:    true,
			IndexingMode: azcosmos.IndexingModeConsistent,
		},
	}

	throughput := azcosmos.NewManualThroughputProperties(400)
	response, err := database.CreateContainer(ctx, properties, &azcosmos.CreateContainerOptions{ThroughputProperties: &throughput})
	if err != nil {
		return "", err
	}
	return response.ActivityID, nil
}

func recreateContainer(ctx context.Context, client *azcosmos.Client, dbName, dbEndpoint, cname string, indexPaths []azcosmos.IncludedPath) (string, error) {
	found := true
	cc, err := client.NewContainer(dbName, cname)
	if err != nil { //&& !IsNotFound(err) {
		// if !strings.Contains(err.Error(), "Resource Not Found") {
		return "", fmt.Errorf(
			"failed to connect to Cosmos DB container: endpoint=%q, container=%q. %w",
			dbEndpoint,
			cname,
			err,
		)
	}
	if _, err = cc.Read(ctx, nil); err != nil {
		if !IsNotFound(err) {
			return "", fmt.Errorf(
				"failed to connect to Cosmos DB container: endpoint=%q, container=%q. %w",
				dbEndpoint,
				cname,
				err,
			)
		}
		found = false
	}

	if found {
		_, err = deleteContainer(ctx, cc)
		if err != nil {
			return "", fmt.Errorf("failed to delete Cosmos DB container: container=%s. %w", cname, err)
		}
	}

	dc, err := client.NewDatabase(dbName)
	if err != nil {
		return "", fmt.Errorf("failed to create Cosmos DB database client: %w", err)
	}
	activityID, err := createContainer(ctx, dc, cname, indexPaths)
	if err != nil && !IsConflict(err) {
		return "", fmt.Errorf("failed to create Cosmos DB container: container=%s. %w", cname, err)
		// slog.Default().Warn(fmt.Sprintf("Container %s already exists: %s", c.name, err))
	}

	return activityID, nil
}

// For deleting in testing.
func deleteContainer(ctx context.Context, cc *azcosmos.ContainerClient) (string, error) {
	response, err := cc.Delete(ctx, &azcosmos.DeleteContainerOptions{})
	if err != nil {
		return "", err
	}
	return response.ActivityID, nil
}
