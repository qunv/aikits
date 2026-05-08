package scaffold

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"aikits/internal/command/agent"
	"aikits/internal/scaffold"
)

// AgentList returns all supported agents in canonical order.
func AgentList() []AgentInfo {
	all := agent.All()
	out := make([]AgentInfo, len(all))
	for i, a := range all {
		out[i] = AgentInfo{Name: a.Name(), DestDir: a.DestDir()}
	}
	return out
}

// FindGitRoot finds the nearest .git directory starting from dir.
func FindGitRoot(dir string) (string, error) {
	return scaffold.GitRoot(dir)
}

// SkillList returns available built-in skill names.
func SkillList(_ context.Context) ([]string, error) {
	return scaffold.SkillNames()
}

// Init initializes the AI workflow scaffold in opts.Target.
// Workspace docs are always written. Agent instructions are installed for
// each name in opts.Agents (empty → workspace only).
// Results are accumulated even when a per-agent error occurs; the first
// encountered error is returned alongside the partial InitResult.
func Init(ctx context.Context, opts InitOptions) (*InitResult, error) {
	target := opts.Target
	if target == "" {
		target = "."
	}

	wsResult, err := scaffold.Workspace(target)
	if err != nil {
		return &InitResult{TargetDir: wsResult.TargetDir}, err
	}

	result := &InitResult{
		TargetDir: wsResult.TargetDir,
		Created:   wsResult.Created,
		Skipped:   wsResult.Skipped,
	}

	var firstErr error
	for _, name := range opts.Agents {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		a, err := agent.Get(name)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		instResult, err := scaffold.Instructions(a, target)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		result.Created = append(result.Created, instResult.Created...)
		result.Skipped = append(result.Skipped, instResult.Skipped...)
	}

	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, firstErr
}

// InstallAgent installs instructions for a single named agent into targetDir.
// This is a lower-level alternative to Init when only agent instructions are needed.
func InstallAgent(ctx context.Context, target, agentName string) (*InitResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a, err := agent.Get(agentName)
	if err != nil {
		return nil, err
	}
	r, err := scaffold.Instructions(a, target)
	if err != nil {
		return nil, err
	}
	return &InitResult{
		TargetDir: r.TargetDir,
		Created:   r.Created,
		Skipped:   r.Skipped,
	}, nil
}

// InitWorkspace initializes only the workspace docs scaffold (no agent instructions).
func InitWorkspace(ctx context.Context, target string) (*InitResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, err := scaffold.Workspace(target)
	if err != nil {
		return nil, err
	}
	return &InitResult{
		TargetDir: r.TargetDir,
		Created:   r.Created,
		Skipped:   r.Skipped,
	}, nil
}
// It continues across agents on error; each SkillAddResult may carry its own Err.
func SkillAdd(ctx context.Context, opts SkillAddOptions) ([]SkillAddResult, error) {
	if len(opts.Agents) == 0 {
		return nil, fmt.Errorf("at least one agent is required")
	}

	gitRoot := opts.GitRoot
	if gitRoot == "" {
		var err error
		gitRoot, err = scaffold.GitRoot(".")
		if err != nil {
			return nil, fmt.Errorf("agent install requires a git repository: %w", err)
		}
	}

	var results []SkillAddResult
	var firstErr error

	for _, name := range opts.Agents {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		a, err := agent.Get(name)
		if err != nil {
			results = append(results, SkillAddResult{Agent: name, Err: err})
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		destRoot := filepath.Join(gitRoot, a.DestDir(), "skills")
		skillResult, err := scaffold.AddSkill(opts.SkillName, destRoot)

		r := SkillAddResult{
			SkillName: opts.SkillName,
			Agent:     name,
			Err:       err,
		}
		if skillResult != nil {
			r.DestDir = skillResult.DestDir
			r.Created = skillResult.Created
			r.Skipped = skillResult.Skipped
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
		results = append(results, r)
	}

	return results, firstErr
}

// AddSkillTo copies the named skill into destRoot/<skillName>/ directly.
// This is a lower-level alternative to SkillAdd when the destination path is
// already known and no agent/git-root resolution is needed.
func AddSkillTo(_ context.Context, skillName, destRoot string) (*SkillAddResult, error) {
	r, err := scaffold.AddSkill(skillName, destRoot)
	result := &SkillAddResult{
		SkillName: skillName,
		Err:       err,
	}
	if r != nil {
		result.DestDir = r.DestDir
		result.Created = r.Created
		result.Skipped = r.Skipped
	}
	return result, err
}
