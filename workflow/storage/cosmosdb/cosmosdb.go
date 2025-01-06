/*
Package cosmosdb provides a cosmosdb-based storage implementation for workflow.Plan data. This is used
to implement the storage.ReadWriter interface.

This package is for use only by the coercion.Workstream and any use outside of that is not
supported.
*/
package cosmosdb

import (
	"context"
	"fmt"
	// "os"
	// "path/filepath"
	"sync"

	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/plugins/registry"
	"github.com/element-of-surprise/coercion/workflow/storage"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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

// CosmosDBClient has the methods for all of Create/Update/Delete/Query operation
// on data model.
type CosmosDBClient struct {
	// publisher arg.Publisher

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

func (c *CosmosDBClient) GetPlansClient() *azcosmos.ContainerClient {
	return c.plansClient
}

func (c *CosmosDBClient) GetBlocksClient() *azcosmos.ContainerClient {
	return c.blocksClient
}

func (c *CosmosDBClient) GetChecksClient() *azcosmos.ContainerClient {
	return c.checksClient
}

func (c *CosmosDBClient) GetSequencesClient() *azcosmos.ContainerClient {
	return c.sequencesClient
}

func (c *CosmosDBClient) GetActionsClient() *azcosmos.ContainerClient {
	return c.actionsClient
}

type container struct {
	name string
	init func(*azcosmos.ContainerClient)
}

func (c *CosmosDBClient) containers() []container {
	cont := []container{
		{
			name: "plans",
			init: func(cc *azcosmos.ContainerClient) {
				c.plansClient = cc
			},
		},
		{
			name: "blocks",
			init: func(cc *azcosmos.ContainerClient) {
				c.blocksClient = cc
			},
		},
		{
			name: "checks",
			init: func(cc *azcosmos.ContainerClient) {
				c.checksClient = cc
			},
		},
		{
			name: "sequences",
			init: func(cc *azcosmos.ContainerClient) {
				c.sequencesClient = cc
			},
		},
		{
			name: "actions",
			init: func(cc *azcosmos.ContainerClient) {
				c.actionsClient = cc
			},
		},
	}

	return cont
}

// New is the constructor for *ReadWriter. root is the root path for the storage.
// If the root path does not exist, it will be created.
// Instead of sqlite package, create client to existing cosmosdb.
func New(ctx context.Context, endpoint string, cred azcore.TokenCredential, reg *registry.Register, options ...Option) (*Vault, error) {
	ctx = context.WithoutCancel(ctx)

	dbName := ""
	partitionKey := ""
	r := &Vault{
		dbName:       dbName,
		partitionKey: partitionKey,
	}
	for _, o := range options {
		if err := o(r); err != nil {
			return nil, err
		}
	}

	// o =
	client, err := azcosmos.NewClient(endpoint, cred, nil)
	if err != nil {
		return nil, err
	}

	// create container for customer? or container per table for all customers?
	// multiple containers per customer?
	// need containers/tables for plans, blocks, checks, sequences, and actions.

	cc, err := createContainerClients(ctx, dbName, endpoint, client)
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
	dbName, dbEndpoint string,
	azCosmosClient *azcosmos.Client) (*CosmosDBClient, error) {

	client := &CosmosDBClient{}

	// TODO (sehobbs) - Consider making container creation a concurrent operation.
	for _, c := range client.containers() {
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
func createContainer(ctx context.Context, database azcosmos.DatabaseClient, cname string) (string, error) {
	properties := azcosmos.ContainerProperties{
		ID: cname,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			// Paths: []string{"/id"},
			Paths: []string{"/underlayName"},
		},
	}

	throughput := azcosmos.NewManualThroughputProperties(400)
	response, err := database.CreateContainer(ctx, properties, &azcosmos.CreateContainerOptions{ThroughputProperties: &throughput})
	if err != nil {
		return "", err
	}
	return response.ActivityID, nil
}
