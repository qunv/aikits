// Package scaffold provides the embedded AI workflow templates and the
// business logic for initializing workspaces and managing skills.
package scaffold

import "embed"

const templateRoot = "templates"
const agentTemplateRoot = "templates/agents"
const skillsTemplateRoot = "templates/agents/skills"

//go:embed templates/**
var templates embed.FS
