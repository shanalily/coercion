package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/google/uuid"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	// "github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// creator implements the storage.creator interface.
type creator struct {
	mu *sync.Mutex
	cc *CosmosDBClient
	// pool   *cosmosdbx.Pool
	reader reader

	private.Storage
}

// Create writes Plan data to storage, and all underlying data.
func (u creator) Create(ctx context.Context, plan *workflow.Plan) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if plan.ID == uuid.Nil {
		return fmt.Errorf("plan ID cannot be nil")
	}

	exist, err := u.reader.Exists(ctx, plan.ID)
	if err != nil {
		return err
	}

	if exist {
		return fmt.Errorf("plan with ID(%s) already exists", plan.ID)
	}

	// conn, err := u.pool.Take(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't get a connection from the pool: %w", err)
	// }
	// defer u.pool.Put(conn)

	return commitPlan(ctx, plan)
}
