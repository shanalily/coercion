package cosmosdb

import (
	"sync"

	"github.com/element-of-surprise/coercion/internal/private"
	"github.com/element-of-surprise/coercion/workflow/storage"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	// "github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

var _ storage.Updater = updater{}

// updater implements the storage.updater interface.
type updater struct {
	planUpdater
	checksUpdater
	blockUpdater
	sequenceUpdater
	actionUpdater

	private.Storage
}

func newUpdater(mu *sync.Mutex, cc *CosmosDBClient) updater {
	return updater{
		planUpdater:     planUpdater{mu: mu, cc: cc.GetPlansClient(), pk: cc.GetPK()},
		checksUpdater:   checksUpdater{mu: mu, cc: cc.GetChecksClient(), pk: cc.GetPK()},
		blockUpdater:    blockUpdater{mu: mu, cc: cc.GetBlocksClient(), pk: cc.GetPK()},
		sequenceUpdater: sequenceUpdater{mu: mu, cc: cc.GetSequencesClient(), pk: cc.GetPK()},
		actionUpdater:   actionUpdater{mu: mu, cc: cc.GetActionsClient(), pk: cc.GetPK()},
	}
}
