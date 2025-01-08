package cosmosdb

import (
	"context"
	"fmt"
	"time"

	"github.com/element-of-surprise/coercion/plugins"
	"github.com/element-of-surprise/coercion/workflow"

	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	// "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

const insertPlan = `
	INSERT INTO plans (
		id,
		group_id,
		name,
		descr,
		meta,
		bypasschecks,
		prechecks,
		postchecks,
		contchecks,
		deferredchecks,
		blocks,
		state_status,
		state_start,
		state_end,
		submit_time,
		reason
	) VALUES ($id, $group_id, $name, $descr, $meta, $bypasschecks, $prechecks, $postchecks, $contchecks, $deferredchecks,
	$blocks, $state_status, $state_start, $state_end, $submit_time, $reason)`

var zeroTime = time.Unix(0, 0)

// commitPlan commits a plan to the database. This commits the entire plan and all sub-objects.
func (u creator) commitPlan(ctx context.Context, p *workflow.Plan) (err error) {
	if p == nil {
		return fmt.Errorf("planToSQL: plan cannot be nil")
	}

	plan := plansEntry{
		PartitionKey: u.cc.GetPKString(),
		ID:           p.ID.String(),
		GroupID:      p.GroupID.String(),
		Name:         p.Name,
		Descr:        p.Descr,
		// meta: p.Meta,
		StateStatus: int64(p.State.Status),
		StateStart:  p.State.Start.UnixNano(),
		StateEnd:    p.State.End.UnixNano(),
		Reason:      int64(p.Reason),
	}

	// stmt.SetBytes("$meta", p.Meta)
	if p.BypassChecks != nil {
		plan.BypassChecks = p.BypassChecks.ID.String()
	}
	if p.PreChecks != nil {
		plan.PreChecks = p.PreChecks.ID.String()
	}
	if p.PostChecks != nil {
		plan.PostChecks = p.PostChecks.ID.String()
	}
	if p.ContChecks != nil {
		plan.ContChecks = p.ContChecks.ID.String()
	}
	if p.DeferredChecks != nil {
		plan.DeferredChecks = p.DeferredChecks.ID.String()
	}

	blocks, err := idsToJSON(p.Blocks)
	if err != nil {
		return fmt.Errorf("planToSQL(idsToJSON(blocks)): %w", err)
	}
	plan.Blocks = string(blocks)
	if p.SubmitTime.Before(zeroTime) {
		plan.SubmitTime = zeroTime.UnixNano()
	} else {
		plan.SubmitTime = p.SubmitTime.UnixNano()
	}

	if err := u.commitChecks(ctx, p.ID, p.BypassChecks); err != nil {
		return fmt.Errorf("planToSQL(commitChecks(bypasschecks)): %w", err)
	}
	if err := u.commitChecks(ctx, p.ID, p.PreChecks); err != nil {
		return fmt.Errorf("planToSQL(commitChecks(prechecks)): %w", err)
	}
	if err := u.commitChecks(ctx, p.ID, p.PostChecks); err != nil {
		return fmt.Errorf("planToSQL(commitChecks(postchecks)): %w", err)
	}
	if err := u.commitChecks(ctx, p.ID, p.ContChecks); err != nil {
		return fmt.Errorf("planToSQL(commitChecks(contchecks)): %w", err)
	}
	if err := u.commitChecks(ctx, p.ID, p.DeferredChecks); err != nil {
		return fmt.Errorf("planToSQL(commitChecks(deferredchecks)): %w", err)
	}
	for i, b := range p.Blocks {
		if err := u.commitBlock(ctx, p.ID, i, b); err != nil {
			return fmt.Errorf("planToSQL(commitBlocks): %w", err)
		}
	}

	// save the JSON format document into Cosmos DB.
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}
	itemJson, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	res, err := u.cc.GetPlansClient().CreateItem(ctx, u.cc.GetPK(), itemJson, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res)

	return nil
}

const insertChecks = `
	INSERT INTO checks (
		id,
		key,
		plan_id,
		actions,
		delay,
		state_status,
		state_start,
		state_end
	) VALUES ($id, $key, $plan_id, $actions, $delay,
	$state_status, $state_start, $state_end)`

func (u creator) commitChecks(ctx context.Context, planID uuid.UUID, c *workflow.Checks) error {
	if c == nil {
		return nil
	}

	actions, err := idsToJSON(c.Actions)
	if err != nil {
		return fmt.Errorf("idsToJSON(checks.Actions): %w", err)
	}
	checks := checksEntry{
		PartitionKey: u.cc.GetPKString(),
		ID:           c.ID.String(),
		Key:          c.Key.String(),
		PlanID:       planID.String(),
		// meta: p.Meta,
		Actions: string(actions),
	}

	// stmt.SetInt64("$delay", int64(checks.Delay))
	// stmt.SetInt64("$state_status", int64(checks.State.Status))
	// stmt.SetInt64("$state_start", checks.State.Start.UnixNano())
	// stmt.SetInt64("$state_end", checks.State.End.UnixNano())

	for i, a := range c.Actions {
		if err := u.commitAction(ctx, planID, i, a); err != nil {
			return fmt.Errorf("commitAction: %w", err)
		}
	}
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}
	itemJson, err := json.Marshal(checks)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	res, err := u.cc.GetChecksClient().CreateItem(ctx, u.cc.GetPK(), itemJson, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res)

	return nil
}

const insertBlock = `
	INSERT INTO blocks (
		id,
		key,
		plan_id,
		name,
		descr,
		pos,
		entrancedelay,
		exitdelay,
		bypasschecks,
		prechecks,
		postchecks,
		contchecks,
		deferredchecks,
		sequences,
		concurrency,
		toleratedfailures,
		state_status,
		state_start,
		state_end
	) VALUES ($id, $key, $plan_id, $name, $descr, $pos, $entrancedelay, $exitdelay, $bypasschecks, $prechecks, $postchecks, $contchecks, $deferredchecks,
	$sequences, $concurrency, $toleratedfailures,$state_status, $state_start, $state_end)`

func (u creator) commitBlock(ctx context.Context, planID uuid.UUID, pos int, b *workflow.Block) error {
	block := blocksEntry{
		PartitionKey:      u.cc.GetPKString(),
		ID:                b.ID.String(),
		Key:               b.Key.String(),
		PlanID:            planID.String(),
		Name:              b.Name,
		Descr:             b.Descr,
		Pos:               int64(pos),
		EntranceDelay:     int64(b.EntranceDelay),
		ExitDelay:         int64(b.ExitDelay),
		Concurrency:       int64(b.Concurrency),
		ToleratedFailures: int64(b.ToleratedFailures),
		StateStatus:       int64(b.State.Status),
		StateStart:        b.State.Start.UnixNano(),
		StateEnd:          b.State.End.UnixNano(),
		// meta: p.Meta,
	}

	for _, c := range [5]*workflow.Checks{b.BypassChecks, b.PreChecks, b.PostChecks, b.ContChecks, b.DeferredChecks} {
		if err := u.commitChecks(ctx, planID, c); err != nil {
			return fmt.Errorf("commitBlock(commitChecks): %w", err)
		}
	}

	sequences, err := idsToJSON(b.Sequences)
	if err != nil {
		return fmt.Errorf("idsToJSON(sequences): %w", err)
	}

	if b.BypassChecks != nil {
		block.BypassChecks = b.BypassChecks.ID.String()
	}
	if b.PreChecks != nil {
		block.PreChecks = b.PreChecks.ID.String()
	}
	if b.PostChecks != nil {
		block.PostChecks = b.PostChecks.ID.String()
	}
	if b.ContChecks != nil {
		block.ContChecks = b.ContChecks.ID.String()
	}
	if b.DeferredChecks != nil {
		block.DeferredChecks = b.DeferredChecks.ID.String()
	}
	block.Sequences = string(sequences)

	for i, seq := range b.Sequences {
		if err := u.commitSequence(ctx, planID, i, seq); err != nil {
			return fmt.Errorf("(commitSequence: %w", err)
		}
	}
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}
	itemJson, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	res, err := u.cc.GetBlocksClient().CreateItem(ctx, u.cc.GetPK(), itemJson, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res)

	return nil
}

const insertSequence = `
	INSERT INTO sequences (
		id,
		key,
		plan_id,
		name,
		descr,
		pos,
		actions,
		state_status,
		state_start,
		state_end
	) VALUES ($id, $key, $plan_id, $name, $descr, $pos, $actions, $state_status, $state_start, $state_end)`

func (u creator) commitSequence(ctx context.Context, planID uuid.UUID, pos int, seq *workflow.Sequence) error {
	actions, err := idsToJSON(seq.Actions)
	if err != nil {
		return fmt.Errorf("idsToJSON(actions): %w", err)
	}

	sequence := sequencesEntry{
		PartitionKey: u.cc.GetPKString(),
		ID:           seq.ID.String(),
		Key:          seq.Key.String(),
		PlanID:       planID.String(),
		Name:         seq.Name,
		Descr:        seq.Descr,
		Pos:          int64(pos),
		Actions:      string(actions),
		StateStatus:  int64(seq.State.Status),
		StateStart:   seq.State.Start.UnixNano(),
		StateEnd:     seq.State.End.UnixNano(),
		// meta: p.Meta,
	}

	for i, a := range seq.Actions {
		if err := u.commitAction(ctx, planID, i, a); err != nil {
			return fmt.Errorf("planToSQL(commitAction): %w", err)
		}
	}
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}
	itemJson, err := json.Marshal(sequence)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	res, err := u.cc.GetSequencesClient().CreateItem(ctx, u.cc.GetPK(), itemJson, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res)

	return nil
}

const insertAction = `
	INSERT INTO actions (
		id,
		key,
		plan_id,
		name,
		descr,
		pos,
		plugin,
		timeout,
		retries,
		req,
		attempts,
		state_status,
		state_start,
		state_end
	) VALUES ($id, $key, $plan_id, $name, $descr, $pos, $plugin, $timeout, $retries, $req, $attempts,
	$state_status, $state_start, $state_end)`

func (u creator) commitAction(ctx context.Context, planID uuid.UUID, pos int, a *workflow.Action) error {
	// stmt, err := conn.Prepare(insertAction)
	// if err != nil {
	// 	return err
	// }

	req, err := json.Marshal(a.Req)
	if err != nil {
		return fmt.Errorf("json.Marshal(req): %w", err)
	}

	// attempts, err := encodeAttempts(a.Attempts)
	// if err != nil {
	// 	return fmt.Errorf("can't encode action.Attempts: %w", err)
	// }
	action := actionsEntry{
		PartitionKey: u.cc.GetPKString(),
		ID:           a.ID.String(),
		Key:          a.Key.String(),
		PlanID:       planID.String(),
		Name:         a.Name,
		Descr:        a.Descr,
		Pos:          int64(pos),
		Req:          string(req),
		// Attempts: string(attempts),
		StateStatus: int64(a.State.Status),
		StateStart:  a.State.Start.UnixNano(),
		StateEnd:    a.State.End.UnixNano(),
		// meta: p.Meta,
	}

	// stmt.SetText("$plugin", action.Plugin)
	// stmt.SetInt64("$timeout", int64(action.Timeout))
	// stmt.SetInt64("$retries", int64(action.Retries))
	// stmt.SetBytes("$req", req)
	// if attempts != nil {
	// 	stmt.SetBytes("$attempts", attempts)
	// }
	// stmt.SetInt64("$state_status", int64(action.State.Status))
	// stmt.SetInt64("$state_start", action.State.Start.UnixNano())
	// stmt.SetInt64("$state_end", action.State.End.UnixNano())

	// _, err = stmt.Step()
	// if err != nil {
	// 	return err
	// }
	itemOpt := &azcosmos.ItemOptions{
		EnableContentResponseOnWrite: true,
	}
	itemJson, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	res, err := u.cc.GetActionsClient().CreateItem(ctx, u.cc.GetPK(), itemJson, itemOpt)
	if err != nil {
		return fmt.Errorf("failed to write item through Cosmos DB API: %w", err)
	}
	fmt.Println(res)

	return nil
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

func idsToJSON[T any](objs []T) ([]byte, error) {
	ids := make([]string, 0, len(objs))
	for _, o := range objs {
		if ider, ok := any(o).(ider); ok {
			id := ider.GetID()
			ids = append(ids, id.String())
		} else {
			return nil, fmt.Errorf("idsToJSON: object %T does not implement ider", o)
		}
	}
	return json.Marshal(ids)
}
