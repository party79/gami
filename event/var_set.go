package event

// VarSet triggered when a variable is set via agi or dialplan.
type VarSet struct {
	Privilege    []string
	Channel      string `AMI:"Channel"`
	VariableName string `AMI:"Variable"`
	Value        string `AMI:"Value"`
	UniqueID     string `AMI:"Uniqueid"`
}

func init() {
	RegisterEvent((*VarSet)(nil), "VarSet")
}
