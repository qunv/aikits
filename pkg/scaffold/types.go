// Package scaffold provides a programmatic API for initializing AI workflow
// scaffolding (docs, prompts, instructions) and managing bundled skills.
//
// All functions accept a context.Context as the first argument and return
// typed results. There are no interactive prompts; callers supply all
// parameters explicitly.
package scaffold

// InitOptions controls the behaviour of Init.
type InitOptions struct {
	// Target is the directory to initialize. Defaults to "." when empty.
	Target string
	// Agents is the list of agent names to install instructions for
	// (e.g. "copilot", "claude"). When empty, only the workspace scaffold
	// (docs/ai/…) is written — no agent-specific files are installed.
	Agents []string
}

// InitResult is the aggregate outcome of an Init call.
type InitResult struct {
	TargetDir string
	Created   []string
	Skipped   []string
}

// SkillAddOptions controls the behaviour of SkillAdd.
type SkillAddOptions struct {
	// SkillName is the name of the built-in skill to install (e.g. "tdd").
	SkillName string
	// Agents is the list of agent names to install the skill for.
	// Required (at least one).
	Agents []string
	// GitRoot is the repository root. When empty, it is auto-detected by
	// walking up from the current directory.
	GitRoot string
}

// SkillAddResult is the outcome for one agent during a SkillAdd call.
type SkillAddResult struct {
	SkillName string
	Agent     string
	DestDir   string
	Created   []string
	Skipped   []string
	// Err holds any error that occurred for this agent. On error, Created
	// and Skipped may be partially populated.
	Err error
}

// AgentInfo describes a supported AI coding agent.
type AgentInfo struct {
	Name    string
	DestDir string
}
