package agent

type claudeAgent struct{}

func (claudeAgent) Name() string    { return "claude" }
func (claudeAgent) DestDir() string { return ".claude" }
