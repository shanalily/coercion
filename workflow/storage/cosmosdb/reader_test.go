package cosmosdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/google/uuid"
	"github.com/kylelemons/godebug/pretty"

	"github.com/element-of-surprise/coercion/workflow/storage"
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

func TestBuildSearchQuery(t *testing.T) {
	t.Parallel()

	id1, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	id2, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		filters    storage.Filters
		wantQuery  string
		wantParams []azcosmos.QueryParameter
	}{
		{
			name:       "Success: empty filters",
			filters:    storage.Filters{},
			wantQuery:  `SELECT id, group_id, name, descr, submit_time, state_status, state_start, state_end FROM plans WHERE ORDER BY submit_time DESC;`,
			wantParams: nil,
		},
		{
			name: "Success: by IDs with single ID",
			filters: storage.Filters{
				ByIDs: []uuid.UUID{
					id1,
				},
			},
			wantQuery: `SELECT id, group_id, name, descr, submit_time, state_status, state_start, state_end FROM plans WHERE id IN @ids ORDER BY submit_time DESC;`,
			wantParams: []azcosmos.QueryParameter{
				{
					Name:  "@ids",
					Value: "['" + id1.String() + "']",
				},
			},
		},
		{
			name: "Success: by IDs with multiple IDs",
			filters: storage.Filters{
				ByIDs: []uuid.UUID{
					id1,
					id2,
				},
			},
			wantQuery: `SELECT id, group_id, name, descr, submit_time, state_status, state_start, state_end FROM plans WHERE id IN @ids ORDER BY submit_time DESC;`,
			wantParams: []azcosmos.QueryParameter{
				{
					Name:  "@ids",
					Value: "['" + id1.String() + "','" + id2.String() + "']",
				},
			},
		},
	}

	for _, test := range tests {
		r := reader{}
		query, params := r.buildSearchQuery(test.filters)
		if test.wantQuery != query {
			t.Errorf("TestBuildSearchQuery(%s): got query == %s, want query == %s", test.name, query, test.wantQuery)
			continue
		}
		if diff := pretty.Compare(test.wantParams, params); diff != "" {
			t.Errorf("TestBuildSearchQuery(%s): returned params: -want/+got:\n%s", test.name, diff)
		}
	}
}
