package cosmosdb

import (
	"context"
	"fmt"
	"time"

	"github.com/element-of-surprise/coercion/plugins"
	"github.com/element-of-surprise/coercion/workflow"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
)

var (
	zeroTime = time.Unix(0, 0)

	createOpt = defaultOpt
)

func defaultOpt() *azcosmos.ItemOptions {
	return &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}
}

// commitPlan commits a plan to the database. This commits the entire plan and all sub-objects.
func (u creator) commitPlan(ctx context.Context, p *workflow.Plan) (err error) {
	plan, err := planToEntry(ctx, u.cc.GetPKString(), p)
	if err != nil {
		return err
	}

	for _, c := range [5]*workflow.Checks{p.BypassChecks, p.PreChecks, p.PostChecks, p.ContChecks, p.DeferredChecks} {
		if err := u.commitChecks(ctx, p.ID, c); err != nil {
			return fmt.Errorf("planToEntry(commitChecks): %w", err)
		}
	}

	for i, b := range p.Blocks {
		if err := u.commitBlock(ctx, p.ID, i, b); err != nil {
			return fmt.Errorf("planToEntry(commitBlocks): %w", err)
		}
	}

	// save the JSON format document into Cosmos DB.
	itemJson, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	if _, err := u.cc.GetPlansClient().CreateItem(ctx, u.cc.GetPK(), itemJson, createOpt()); err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}

	return nil
}

func planToEntry(ctx context.Context, pk string, p *workflow.Plan) (plansEntry, error) {
	if p == nil {
		return plansEntry{}, fmt.Errorf("planToEntry: plan cannot be nil")
	}

	blocks, err := objsToIDs(p.Blocks)
	if err != nil {
		return plansEntry{}, fmt.Errorf("planToEntry(objsToIDs(blocks)): %w", err)
	}

	plan := plansEntry{
		PartitionKey: pk,
		ID:           p.ID,
		GroupID:      p.GroupID,
		Name:         p.Name,
		Descr:        p.Descr,
		Meta:         p.Meta,
		Blocks:       blocks,
		StateStatus:  p.State.Status,
		StateStart:   p.State.Start,
		StateEnd:     p.State.End,
		Reason:       p.Reason,
	}

	if p.BypassChecks != nil {
		plan.BypassChecks = p.BypassChecks.ID
	}
	if p.PreChecks != nil {
		plan.PreChecks = p.PreChecks.ID
	}
	if p.PostChecks != nil {
		plan.PostChecks = p.PostChecks.ID
	}
	if p.ContChecks != nil {
		plan.ContChecks = p.ContChecks.ID
	}
	if p.DeferredChecks != nil {
		plan.DeferredChecks = p.DeferredChecks.ID
	}

	if p.SubmitTime.Before(zeroTime) {
		plan.SubmitTime = zeroTime
	} else {
		plan.SubmitTime = p.SubmitTime
	}

	return plan, nil
}

func (u creator) commitChecks(ctx context.Context, planID uuid.UUID, c *workflow.Checks) error {
	if c == nil {
		return nil
	}

	checks, err := checkToEntry(ctx, u.cc.GetPKString(), planID, c)
	if err != nil {
		return err
	}

	for i, a := range c.Actions {
		if err := u.commitAction(ctx, planID, i, a); err != nil {
			return fmt.Errorf("commitAction: %w", err)
		}
	}
	itemJson, err := json.Marshal(checks)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	if _, err := u.cc.GetChecksClient().CreateItem(ctx, u.cc.GetPK(), itemJson, createOpt()); err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}

	return nil
}

func checkToEntry(ctx context.Context, pk string, planID uuid.UUID, c *workflow.Checks) (checksEntry, error) {
	if c == nil {
		return checksEntry{}, nil
	}

	actions, err := objsToIDs(c.Actions)
	if err != nil {
		return checksEntry{}, fmt.Errorf("objsToIDs(checks.Actions): %w", err)
	}
	return checksEntry{
		PartitionKey: pk,
		ID:           c.ID,
		Key:          c.Key,
		PlanID:       planID,
		Actions:      actions,
		Delay:        c.Delay,
		StateStatus:  c.State.Status,
		StateStart:   c.State.Start,
		StateEnd:     c.State.End,
	}, nil
}

func (u creator) commitBlock(ctx context.Context, planID uuid.UUID, pos int, b *workflow.Block) error {
	block, err := blockToEntry(ctx, u.cc.GetPKString(), planID, pos, b)
	if err != nil {
		return err
	}

	for _, c := range [5]*workflow.Checks{b.BypassChecks, b.PreChecks, b.PostChecks, b.ContChecks, b.DeferredChecks} {
		if err := u.commitChecks(ctx, planID, c); err != nil {
			return fmt.Errorf("commitBlock(commitChecks): %w", err)
		}
	}

	for i, seq := range b.Sequences {
		if err := u.commitSequence(ctx, planID, i, seq); err != nil {
			return fmt.Errorf("(commitSequence: %w", err)
		}
	}
	itemJson, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	if _, err := u.cc.GetBlocksClient().CreateItem(ctx, u.cc.GetPK(), itemJson, createOpt()); err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}

	return nil
}

func blockToEntry(ctx context.Context, pk string, planID uuid.UUID, pos int, b *workflow.Block) (blocksEntry, error) {
	sequences, err := objsToIDs(b.Sequences)
	if err != nil {
		return blocksEntry{}, fmt.Errorf("objsToIDs(sequences): %w", err)
	}

	block := blocksEntry{
		PartitionKey:      pk,
		ID:                b.ID,
		Key:               b.Key,
		PlanID:            planID,
		Name:              b.Name,
		Descr:             b.Descr,
		Pos:               pos,
		EntranceDelay:     b.EntranceDelay,
		ExitDelay:         b.ExitDelay,
		Sequences:         sequences,
		Concurrency:       b.Concurrency,
		ToleratedFailures: b.ToleratedFailures,
		StateStatus:       b.State.Status,
		StateStart:        b.State.Start,
		StateEnd:          b.State.End,
	}

	if b.BypassChecks != nil {
		block.BypassChecks = b.BypassChecks.ID
	}
	if b.PreChecks != nil {
		block.PreChecks = b.PreChecks.ID
	}
	if b.PostChecks != nil {
		block.PostChecks = b.PostChecks.ID
	}
	if b.ContChecks != nil {
		block.ContChecks = b.ContChecks.ID
	}
	if b.DeferredChecks != nil {
		block.DeferredChecks = b.DeferredChecks.ID
	}
	return block, nil
}

func (u creator) commitSequence(ctx context.Context, planID uuid.UUID, pos int, seq *workflow.Sequence) error {
	sequence, err := sequenceToEntry(ctx, u.cc.GetPKString(), planID, pos, seq)
	if err != nil {
		return err
	}

	for i, a := range seq.Actions {
		if err := u.commitAction(ctx, planID, i, a); err != nil {
			return fmt.Errorf("planToEntry(commitAction): %w", err)
		}
	}
	itemJson, err := json.Marshal(sequence)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	if _, err := u.cc.GetSequencesClient().CreateItem(ctx, u.cc.GetPK(), itemJson, createOpt()); err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}

	return nil
}

func sequenceToEntry(ctx context.Context, pk string, planID uuid.UUID, pos int, seq *workflow.Sequence) (sequencesEntry, error) {
	actions, err := objsToIDs(seq.Actions)
	if err != nil {
		return sequencesEntry{}, fmt.Errorf("objsToIDs(actions): %w", err)
	}

	return sequencesEntry{
		PartitionKey: pk,
		ID:           seq.ID,
		Key:          seq.Key,
		PlanID:       planID,
		Name:         seq.Name,
		Descr:        seq.Descr,
		Pos:          pos,
		Actions:      actions,
		StateStatus:  seq.State.Status,
		StateStart:   seq.State.Start,
		StateEnd:     seq.State.End,
	}, nil
}

func (u creator) commitAction(ctx context.Context, planID uuid.UUID, pos int, a *workflow.Action) error {
	action, err := actionToEntry(ctx, u.cc.GetPKString(), planID, pos, a)
	if err != nil {
		return err
	}

	itemJson, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	if _, err := u.cc.GetActionsClient().CreateItem(ctx, u.cc.GetPK(), itemJson, createOpt()); err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}

	return nil
}

func actionToEntry(ctx context.Context, pk string, planID uuid.UUID, pos int, a *workflow.Action) (actionsEntry, error) {
	req, err := json.Marshal(a.Req)
	if err != nil {
		return actionsEntry{}, fmt.Errorf("json.Marshal(req): %w", err)
	}
	attempts, err := encodeAttempts(a.Attempts)
	if err != nil {
		return actionsEntry{}, fmt.Errorf("can't encode action.Attempts: %w", err)
	}
	return actionsEntry{
		PartitionKey: pk,
		ID:           a.ID,
		Key:          a.Key,
		PlanID:       planID,
		Name:         a.Name,
		Descr:        a.Descr,
		Pos:          pos,
		Plugin:       a.Plugin,
		Timeout:      a.Timeout,
		Retries:      a.Retries,
		Req:          req,
		Attempts:     attempts,
		StateStatus:  a.State.Status,
		StateStart:   a.State.Start,
		StateEnd:     a.State.End,
	}, nil
}

// encodeAttempts encodes a slice of attempts into a JSON array hodling JSON encoded attempts as byte slices.
func encodeAttempts(attempts []*workflow.Attempt) ([]byte, error) {
	if len(attempts) == 0 {
		return nil, nil
	}
	var out [][]byte
	if len(attempts) > 0 {
		out = make([][]byte, 0, len(attempts))
		for _, a := range attempts {
			b, err := json.Marshal(a)
			if err != nil {
				return nil, fmt.Errorf("json.Marshal(attempt): %w", err)
			}
			out = append(out, b)
		}
	}
	return json.Marshal(out)
}

// decodeAttempts decodes a JSON array of JSON encoded attempts as byte slices into a slice of attempts.
func decodeAttempts(rawAttempts []byte, plug plugins.Plugin) ([]*workflow.Attempt, error) {
	rawList := make([][]byte, 0)
	if err := json.Unmarshal(rawAttempts, &rawList); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(rawAttempts): %w", err)
	}

	attempts := make([]*workflow.Attempt, 0, len(rawList))
	for _, raw := range rawList {
		var a = &workflow.Attempt{Resp: plug.Response()}
		if err := json.Unmarshal(raw, a); err != nil {
			return nil, fmt.Errorf("json.Unmarshal(raw): %w", err)
		}
		attempts = append(attempts, a)
	}
	return attempts, nil
}

type ider interface {
	GetID() uuid.UUID
}

func objsToIDs[T any](objs []T) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(objs))
	for _, o := range objs {
		if ider, ok := any(o).(ider); ok {
			id := ider.GetID()
			ids = append(ids, id)
		} else {
			return nil, fmt.Errorf("objsToIDs: object %T does not implement ider", o)
		}
	}
	return ids, nil
}
