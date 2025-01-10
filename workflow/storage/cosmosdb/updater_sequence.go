package cosmosdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/storage"
)

var _ storage.SequenceUpdater = sequenceUpdater{}

// sequenceUpdater implements the storage.sequenceUpdater interface.
type sequenceUpdater struct {
	mu *sync.Mutex
	cc ContainerClient
	pk azcosmos.PartitionKey

	private.Storage
}

// UpdateSequence implements storage.SequenceUpdater.UpdateSequence().
func (s sequenceUpdater) UpdateSequence(ctx context.Context, seq *workflow.Sequence) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	patch := azcosmos.PatchOperations{}
	patch.AppendReplace("/stateStatus", seq.State.Status)
	patch.AppendReplace("/stateStart", seq.State.Start)
	patch.AppendReplace("/stateEnd", seq.State.End)

	var ifMatchEtag *azcore.ETag = nil
	if seq.State.ETag != "" {
		ifMatchEtag = (*azcore.ETag)(&seq.State.ETag)
	}
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
		IfMatchEtag:                  ifMatchEtag,
	}

	// save the item into Cosmos DB
	res, err := s.cc.PatchItem(ctx, s.pk, seq.ID.String(), patch, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to patch item through Cosmos DB API: %w", err)
	}
	fmt.Println(res.ETag)

	return nil
}
