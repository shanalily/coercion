package etoe

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	workstream "github.com/element-of-surprise/coercion"
	"github.com/element-of-surprise/coercion/plugins/registry"
	"github.com/element-of-surprise/coercion/workflow"
	"github.com/element-of-surprise/coercion/workflow/builder"
	"github.com/element-of-surprise/coercion/workflow/storage/azblob"
	"github.com/element-of-surprise/coercion/workflow/utils/clone"

	testplugin "github.com/element-of-surprise/coercion/internal/execute/sm/testing/plugins"
)

var (
	// Test configuration flags
	blobURL     = flag.String("blob_url", fmt.Sprintf("https://%s.blob.core.windows.net", os.Getenv("AZURE_BLOB_ACCOUNT")), "The endpoint of the azblob account for recovery tests.")
	blobMSI     = flag.String("blob_msi", "", "The identity for blob storage. If empty, az login is used.")
	blobPrefix  = flag.String("blob_prefix", "coercion-recovery-test", "The prefix for blob containers in recovery tests.")
	skipCleanup = flag.Bool("skip_cleanup", false, "Skip cleanup of blob storage after test.")
)

// TestBlobStorageRecovery tests the recovery functionality for blob storage vault.
// It creates multiple long-running plans (configurable via -plan_count flag),
// starts execution of all plans, simulates a crash, then recreates the vault
// and workstream to verify recovery works correctly. Only prints one plan result.
func TestBlobStorageRecovery(t *testing.T) {
	flag.Parse()

	if *blobURL == "" {
		t.Skip("blob_url not configured, skipping blob storage recovery test")
	}

	ctx := context.Background()

	// Create credentials
	cred, err := createBlobCredential(*blobMSI)
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}

	// Create a unique prefix for this test to avoid conflicts
	testPrefix := fmt.Sprintf("%s-%d", *blobPrefix, time.Now().Unix())

	// Cleanup function
	cleanup := func() {
		if !*skipCleanup {
			if err := azblob.Teardown(context.Background(), *blobURL, testPrefix, cred); err != nil {
				t.Logf("Warning: Failed to cleanup blob storage: %v", err)
			}
		}
	}
	defer cleanup()

	// Create plugin registry with test plugins
	reg := registry.New()
	reg.Register(&testplugin.Plugin{
		AlwaysRespond: true,
		IsCheckPlugin: true,
		PlugName:      "check",
	})
	reg.Register(&testplugin.Plugin{
		AlwaysRespond: true,
		PlugName:      testplugin.Name,
	})

	// Create the initial vault
	vault, err := azblob.New(ctx, testPrefix, *blobURL, cred, reg)
	if err != nil {
		t.Fatalf("Failed to create blob vault: %v", err)
	}

	// Create initial workstream
	ws, err := workstream.New(ctx, reg, vault)
	if err != nil {
		t.Fatalf("Failed to create workstream: %v", err)
	}

	// Create and submit multiple plans
	plan, err := createLongRunningPlan()
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	planID, err := ws.Submit(ctx, plan)
	if err != nil {
		t.Fatalf("Failed to submit plan: %v", err)
	}

	t.Logf("Submitted plan with ID: %s", planID)

	// Create a cancellable context for plan execution
	executionCtx, cancelExecution := context.WithCancel(ctx)

	// Start plan with the cancellable execution context
	if err := ws.Start(executionCtx, planID); err != nil {
		t.Fatalf("Failed to start plan %s: %v", planID, err)
	}
	t.Logf("Started plan %s, waiting for execution to begin...", planID)

	// Wait a short time to ensure execution has started
	time.Sleep(5 * time.Second)

	// Check that the first plan is running. Do I need to check all?
	status := ws.Status(ctx, planID, 1*time.Second)

	var lastResult *workflow.Plan
	for result := range status {
		if result.Err != nil {
			t.Fatalf("Error in status stream: %v", result.Err)
		}
		lastResult = result.Data
		break // Just get the first status update
	}

	pConfig.Print("Workflow result: \n", lastResult)

	if lastResult.State.Status != workflow.Running {
		t.Fatalf("Expected plan to be running, got status: %s", lastResult.State.Status)
	}

	t.Log("Plan is running, now simulating crash by canceling execution context...")

	// Simulate a crash by canceling the execution context first
	cancelExecution()

	// Wait a moment for the cancellation to take effect
	time.Sleep(1 * time.Second)

	// Then close the vault to simulate an abrupt shutdown
	if err := vault.Close(ctx); err != nil {
		t.Logf("Warning: Error closing vault: %v", err)
	}

	// Simulate system restart by creating new vault and workstream
	t.Log("Simulating system restart - creating new vault and workstream...")

	// time.Sleep(5 * time.Minute) // wait for "leader election"
	time.Sleep(5 * time.Second)

	// Create new vault with recovery enabled (default)
	recoveryVault, err := azblob.New(ctx, testPrefix, *blobURL, cred, reg)
	if err != nil {
		t.Fatalf("Failed to create recovery vault: %v", err)
	}
	defer func() {
		if err := recoveryVault.Close(context.Background()); err != nil {
			t.Logf("Warning: Error closing recovery vault: %v", err)
		}
	}()

	// Create new workstream for recovery
	recoveryWS, err := workstream.New(ctx, reg, recoveryVault)
	if err != nil {
		t.Fatalf("Failed to create recovery workstream: %v", err)
	}

	t.Log("Recovery workstream created, attempting to recover plan...")

	// Wait for the recovered first plan to complete or timeout
	// ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	// defer cancel()

	// start plan again after recovery
	// what should I check before running?
	if err := ws.Start(ctx, planID); err != nil {
		t.Fatalf("Failed to start plan %s: %v", planID, err)
	}
	t.Logf("Started plan %s after restart", planID)

	result, err := recoveryWS.Wait(ctx, planID)
	if err != nil {
		t.Fatalf("Failed to wait for recovered plan: %v", err)
	}
	if result.State.Status != workflow.Completed {
		t.Fatalf("Expected recovered plan to complete, got status: %s", result.State.Status)
	}
	// Additional validation: Check that some actions were actually executed
	if len(result.Blocks) == 0 {
		t.Fatal("Expected plan to have blocks")
	}

	for _, block := range result.Blocks {
		if block.State.Status != workflow.Completed {
			t.Errorf("Block %s did not complete, status: %s", block.ID, block.State.Status)
		}
		for _, seq := range block.Sequences {
			if seq.State.Status != workflow.Completed {
				t.Errorf("Sequence %s did not complete, status: %s", seq.ID, seq.State.Status)
			}
			for _, action := range seq.Actions {
				if action.State.Status != workflow.Completed {
					t.Errorf("Action %s did not complete, status: %s", action.ID, action.State.Status)
				}
			}
		}
	}

	t.Logf("Recovery test successful: plan %s completed after recovery", planID)

	pConfig.Print("Workflow result: \n", result)
	t.Log("All validation checks passed")
}

// createLongRunningPlan creates a plan with actions that sleep for a significant duration
func createLongRunningPlan() (*workflow.Plan, error) {
	ctx := context.Background()

	// Create checks with shorter sleep times for quicker test execution
	checks := &workflow.Checks{
		Delay: 1 * time.Second,
		Actions: []*workflow.Action{
			{
				Name:   "check",
				Descr:  "Quick check action",
				Plugin: "check",
				Req:    testplugin.Req{Arg: "planid"},
			},
		},
	}

	// Create sequences with long-running actions
	longSeq := &workflow.Sequence{
		Key:   workflow.NewV7(),
		Name:  "long-running-sequence",
		Descr: "Sequence with long-running actions for recovery testing",
		Actions: []*workflow.Action{
			{
				Name:    "quick-action",
				Descr:   "Quick action that completes fast",
				Plugin:  testplugin.Name,
				Timeout: 30 * time.Second,
				Req:     testplugin.Req{Sleep: 1 * time.Second, Arg: "quick"},
			},
			{
				Name:    "long-action",
				Descr:   "Long-running action for recovery testing",
				Plugin:  testplugin.Name,
				Timeout: 7 * time.Minute,                                     // Timeout for the long-running action
				Req:     testplugin.Req{Sleep: 5 * time.Minute, Arg: "long"}, // 30 second sleep (shortened for testing)
			},
			{
				Name:    "final-action",
				Descr:   "Final action after recovery",
				Plugin:  testplugin.Name,
				Timeout: 30 * time.Second,
				Req:     testplugin.Req{Sleep: 1 * time.Second, Arg: "final"},
			},
		},
	}

	// Build the plan
	build, err := builder.New("blob-recovery-test", "Test plan for blob storage recovery functionality")
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	// Add plan-level checks (using cloning to avoid register conflicts)
	build.AddChecks(builder.PreChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()
	build.AddChecks(builder.PostChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()
	build.AddChecks(builder.DeferredChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()

	// Add a block with the long-running sequence
	build.AddBlock(
		builder.BlockArgs{
			Key:           workflow.NewV7(),
			Name:          "recovery-test-block",
			Descr:         "Block for testing blob storage recovery",
			EntranceDelay: 1 * time.Second,
			ExitDelay:     1 * time.Second,
			Concurrency:   1, // Single concurrency to ensure predictable execution order
		},
	)

	// Add block-level checks (using cloning to avoid register conflicts)
	build.AddChecks(builder.PreChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()
	build.AddChecks(builder.PostChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()
	build.AddChecks(builder.ContChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()
	build.AddChecks(builder.DeferredChecks, clone.Checks(ctx, checks, cloneOpts...)).Up()

	// Add the long-running sequence (using cloning to avoid register conflicts)
	build.AddSequence(clone.Sequence(ctx, longSeq, cloneOpts...)).Up()

	plan, err := build.Plan()
	if err != nil {
		return nil, fmt.Errorf("failed to build plan: %w", err)
	}

	return plan, nil
}

// createBlobCredential creates the appropriate Azure credential for blob storage
func createBlobCredential(msi string) (azcore.TokenCredential, error) {
	if msi != "" {
		msiResc := azidentity.ResourceID(msi)
		msiOpts := azidentity.ManagedIdentityCredentialOptions{ID: msiResc}
		cred, err := azidentity.NewManagedIdentityCredential(&msiOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create managed identity credential: %w", err)
		}
		return cred, nil
	}

	// Use Azure CLI credential
	azOptions := &azidentity.AzureCLICredentialOptions{}
	azCred, err := azidentity.NewAzureCLICredential(azOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure CLI credential: %w", err)
	}

	return azCred, nil
}
