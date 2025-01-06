package cosmosdb

import (
	"context"
	"fmt"
	"testing"
	// "github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

func TestExampleServer(t *testing.T) {
	client, err := NewContainerClient()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.ReadItem(context.Background(), partitionKey("val"), "itemid", nil)
	fmt.Println(resp)
	fmt.Println(err)
}
