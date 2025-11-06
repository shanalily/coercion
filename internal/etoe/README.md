# End to end tests

## Running

### Sqlite

`go test .`

This will also run recovery tests. These tests validate that our recovery of objects in inconsistent states work. This test is valid no matter what the storage is.

### CosmosDB

go test . -vault=cosmosdb -collection-name="[a collection name]" -db-name="[your db name]" -container-name="[choose one]" -cosmos_url="[the cosmos url]"

### Azblob

go test . -vault=azblob -azblob_url=[https://the url]

### Blob Storage Recovery Test

The `TestBlobStorageRecovery` test specifically validates the recovery functionality of blob storage vaults. It creates a long-running plan, starts execution, simulates an interruption, then creates a new vault/workstream to verify recovery works correctly.

To run the blob storage recovery test:

```bash
go test -run TestBlobStorageRecovery ./internal/etoe -blob_url="https://yourstorageaccount.blob.core.windows.net"
```

Optional flags:
- `-blob_msi=""`: Managed Service Identity resource ID (if empty, uses az cli)
- `-blob_prefix="coercion-recovery-test"`: Prefix for blob containers 
- `-skip_cleanup=false`: Skip cleanup of blob storage after test

**Note**: This test requires Azure credentials to be configured (either via MSI or Azure CLI login) and takes approximately 2 minutes to complete as it tests recovery of long-running operations.
