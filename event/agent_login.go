package event

// AgentLogin trigger when agent logs in
type AgentLogin struct {
	Privilege []string
	Agent     string `AMI:"Agent"`
	UniqueID  string `AMI:"Uniqueid"`
	Channel   string `AMI:"Channel"`
}

func init() {
	RegisterEvent((*AgentLogin)(nil), "AgentLogin")
}
