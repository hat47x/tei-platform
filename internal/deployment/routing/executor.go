package routing

import (
	"context"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
)

// RoutedCommand is a deterministic binding result. Executor selection is
// configuration, not discovery: TEI never ranks or guesses an executor.
type RoutedCommand struct {
	ExecutorID core.CanonicalID `json:"executorId"`
	Command    executor.Command `json:"command"`
}

func RouteCommand(env environment.Environment, cmd executor.Command) (RoutedCommand, error) {
	type candidate struct {
		binding     environment.DeploymentExecutorBinding
		specificity int
	}
	var candidates []candidate
	for _, b := range env.DeploymentExecutors {
		if string(b.Action) != cmd.Kind {
			continue
		}
		spec := 0
		switch b.Scope {
		case "", environment.ExecutorScopeGlobal:
			spec = 0
		case environment.ExecutorScopeDatabase:
			if cmd.Database == "" || b.Target != cmd.Database {
				continue
			}
			spec = 1
		case environment.ExecutorScopeApplication:
			if cmd.Application == "" || b.Target != cmd.Application {
				continue
			}
			spec = 1
		default:
			continue
		}
		candidates = append(candidates, candidate{binding: b, specificity: spec})
	}
	if len(candidates) == 0 {
		return RoutedCommand{}, fmt.Errorf("environment %s has no executor binding for action %s target app=%s database=%s", env.ID, cmd.Kind, cmd.Application, cmd.Database)
	}
	max := -1
	for _, c := range candidates {
		if c.specificity > max {
			max = c.specificity
		}
	}
	var winners []environment.DeploymentExecutorBinding
	for _, c := range candidates {
		if c.specificity == max {
			winners = append(winners, c.binding)
		}
	}
	if len(winners) != 1 {
		return RoutedCommand{}, fmt.Errorf("environment %s has ambiguous executor bindings for action %s at specificity %d", env.ID, cmd.Kind, max)
	}
	return RoutedCommand{ExecutorID: winners[0].Executor, Command: cmd}, nil
}

func RouteCommands(env environment.Environment, cmds []executor.Command) ([]RoutedCommand, error) {
	out := make([]RoutedCommand, 0, len(cmds))
	for _, cmd := range cmds {
		r, err := RouteCommand(env, cmd)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Command.ActionID < out[j].Command.ActionID })
	return out, nil
}

// Router owns an explicit executor registry. The environment binding selects
// the executor ID; Supports is only a conformance check, never a selection
// mechanism.
type Router struct {
	Executors map[core.CanonicalID]executor.Executor
}

func (r Router) Dispatch(ctx context.Context, env environment.Environment, cmd executor.Command) (executor.Result, error) {
	routed, err := RouteCommand(env, cmd)
	if err != nil {
		return executor.Result{}, err
	}
	exec, ok := r.Executors[routed.ExecutorID]
	if !ok {
		return executor.Result{}, fmt.Errorf("executor %s is bound but not registered", routed.ExecutorID)
	}
	if exec.ID() != string(routed.ExecutorID) {
		return executor.Result{}, fmt.Errorf("executor registry key %s does not match executor ID %s", routed.ExecutorID, exec.ID())
	}
	if !exec.Supports(cmd) {
		return executor.Result{}, fmt.Errorf("executor %s does not support action %s", routed.ExecutorID, cmd.Kind)
	}
	result, err := exec.Execute(ctx, cmd)
	if err != nil {
		return executor.Result{}, err
	}
	if result.ExecutorID == "" {
		result.ExecutorID = exec.ID()
	}
	if result.ActionID == "" {
		result.ActionID = cmd.ActionID
	}
	if result.IdempotencyKey == "" {
		result.IdempotencyKey = cmd.IdempotencyKey
	}
	return result, nil
}

// ValidateResult verifies that an executor acknowledgement came from the
// executor selected by the environment binding for the action. This keeps the
// durable run provider-neutral while preventing a caller from recording an
// acknowledgement from an unrelated executor.
func ValidateResult(env environment.Environment, cmd executor.Command, result executor.Result) error {
	routed, err := RouteCommand(env, cmd)
	if err != nil {
		return err
	}
	if result.ExecutorID != string(routed.ExecutorID) {
		return fmt.Errorf("executor result for action %s came from %s, environment binds %s", cmd.ActionID, result.ExecutorID, routed.ExecutorID)
	}
	if result.ActionID != cmd.ActionID {
		return fmt.Errorf("executor result action %s does not match command %s", result.ActionID, cmd.ActionID)
	}
	if result.IdempotencyKey != cmd.IdempotencyKey {
		return fmt.Errorf("executor result idempotency key does not match command %s", cmd.ActionID)
	}
	return nil
}
