package cosmosdb

import (
	"context"

	"github.com/element-of-surprise/coercion/internal/private"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	// "github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

type closer struct {
	cc *CosmosDBClient
	// pool *cosmosdbx.Pool

	private.Storage
}

func (c *closer) Close(ctx context.Context) error {
	// return c.pool.Close()
	return nil
}
