package scaffold

// InitResult holds the outcome of an init or agent-instructions operation.
type InitResult struct {
	TargetDir string
	Created   []string
	Skipped   []string
}

// SkillResult holds the outcome of installing one skill to one agent directory.
type SkillResult struct {
	SkillName string
	DestDir   string
	Created   []string
	Skipped   []string
}
