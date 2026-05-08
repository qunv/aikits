// Package agent defines the Agent strategy interface and the registry of
// supported agents for aikits commands (init, skill add, etc.).
package agent

import "fmt"

// Agent describes an AI coding agent that aikits can configure.
type Agent interface {
	// Name returns the canonical agent identifier (e.g. "copilot").
	Name() string
	// DestDir returns the agent's root directory relative to the repo root
	// (e.g. ".github" for copilot, ".claude" for claude).
	DestDir() string
}

// all is the ordered list of supported agents.
// New agents should be appended here to appear last in prompts.
var all = []Agent{
	copilotAgent{},
	claudeAgent{},
}

// All returns all supported agents in canonical order.
func All() []Agent {
	result := make([]Agent, len(all))
	copy(result, all)
	return result
}

// Get returns the agent with the given name, or an error if not found.
func Get(name string) (Agent, error) {
	for _, a := range all {
		if a.Name() == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("unsupported agent %q — supported: %s", name, Names())
}

// Names returns the names of all supported agents as a comma-separated string.
func Names() string {
	s := ""
	for i, a := range all {
		if i > 0 {
			s += ", "
		}
		s += a.Name()
	}
	return s
}

// Index returns the position of agent a in the canonical order.
// Unknown agents return len(all).
func Index(name string) int {
	for i, a := range all {
		if a.Name() == name {
			return i
		}
	}
	return len(all)
}
