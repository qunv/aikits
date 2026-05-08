package agent

type copilotAgent struct{}

func (copilotAgent) Name() string    { return "copilot" }
func (copilotAgent) DestDir() string { return ".github" }
