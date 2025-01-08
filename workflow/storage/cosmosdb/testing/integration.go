package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/element-of-surprise/coercion/plugins/registry"
	"github.com/element-of-surprise/coercion/workflow/storage/cosmosdb"
	"github.com/element-of-surprise/coercion/workflow"
)

var (
	msi = flag.String("msi", "", "the identity with vmss contributor role")
)

const (
	accountName = "medbaydb"
	dbName      = "medbaydb"
	pk          = "underlayName"
)

func main() {
	var err error

	defer func() {
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	}()

	ctx := context.Background()
	logger := slog.Default()

	reg := registry.New()

	cred, err := msiCred(*msi)
	if err != nil {
		fatalErr(logger, "Failed to create credential: %v", err)
	}

	endpoint := fmt.Sprintf("https://%s.documents.azure.com:443/", accountName)
	vault, err := cosmosdb.New(ctx, endpoint, dbName, pk, cred, reg) //, options...)
	if err != nil {
		fatalErr(logger, "Failed to create vault: %v", err)
	}
	fmt.Println(vault)

	planID, err := uuid.NewV7()
	if err != nil {
		fatalErr(logger, "Failed to create plan ID: %v", err)
	}

	plan := &workflow.Plan{
		ID: planID,
		Name:  "plan name",
		Descr: "plan descr",
		PreChecks: &workflow.Checks{},
		Blocks: []*workflow.Block{
			{
				Name:  "block name",
				Descr: "block descr",
				Sequences: []*workflow.Sequence{
					{
						Name:  "sequence name",
						Descr: "sequence descr",
						Actions: []*workflow.Action{
							{
								Name:    "action name",
								Descr:   "action descr",
								Plugin:  "plugin/name",
								Timeout: 15 * time.Minute,
								Retries: 3,
							},
						},
					},
				},
			},
		},
	}

	if err := vault.Create(context.Background(), plan); err != nil {
		fatalErr(logger, "Failed to create plan entry: %v", err)
	}
}

// msiCred returns a managed identity credential.
func msiCred(msi string) (azcore.TokenCredential, error) {
	if msi != "" {
		msiResc := azidentity.ResourceID(msi)
		msiOpts := azidentity.ManagedIdentityCredentialOptions{ID: msiResc}
		cred, err := azidentity.NewManagedIdentityCredential(&msiOpts)
		if err != nil {
			return nil, err
		}
		log.Println("Authentication is using identity token.")
		return cred, nil
	}
	// Otherwise, allow authentication via az login
	// Need following roles comosdb sql roles:
	// https://learn.microsoft.com/en-us/azure/cosmos-db/nosql/security/how-to-grant-data-plane-role-based-access?tabs=built-in-definition%2Ccsharp&pivots=azure-interface-cli
	azOptions := &azidentity.AzureCLICredentialOptions{}
	azCred, err := azidentity.NewAzureCLICredential(azOptions)
	if err != nil {
		return nil, fmt.Errorf("creating az cli credential: %s", err)
	}

	log.Println("Authentication is using az cli token.")
	return azCred, nil
}

func fatalErr(logger *slog.Logger, msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	logger.Error(s, "fatal", "true")
	os.Exit(1)
}
