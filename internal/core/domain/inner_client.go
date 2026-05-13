package domain

// PowerState represents the power status of a simulated inner client.
type PowerState string

const (
	PowerStateOn         PowerState = "on"
	PowerStateOff        PowerState = "off"
	PowerStateRestarting PowerState = "restarting"
)

// CrippleMode determines how a crippled inner client misbehaves.
type CrippleMode string

const (
	CrippleModeNone           CrippleMode = ""
	CrippleModeFailPackageOps CrippleMode = "fail_package_ops"
	CrippleModeFailConfig     CrippleMode = "fail_config"
	CrippleModeSilent         CrippleMode = "silent"
)

// ClientState holds the mutable internal state of a simulated inner client.
type ClientState struct {
	Packages                map[string]string `json:"packages"`
	ConfigVersion           int               `json:"config_version"`
	PowerState              PowerState        `json:"power_state"`
	IsCrippled              bool              `json:"is_crippled"`
	CrippleMode             CrippleMode       `json:"cripple_mode"`
	CrippleRecoveryAttempts int               `json:"cripple_recovery_attempts"`
}

// InnerClient represents a simulated client managed by the super-client.
type InnerClient struct {
	ClientID string
	State    ClientState
}

// NewInnerClient creates a new inner client with default state.
func NewInnerClient(clientID string) *InnerClient {
	return &InnerClient{
		ClientID: clientID,
		State: ClientState{
			Packages:      make(map[string]string),
			ConfigVersion: 1,
			PowerState:    PowerStateOn,
			IsCrippled:    false,
			CrippleMode:   CrippleModeNone,
		},
	}
}

// Clone returns a deep copy of the ClientState.
func (s ClientState) Clone() ClientState {
	packages := make(map[string]string, len(s.Packages))
	for k, v := range s.Packages {
		packages[k] = v
	}
	return ClientState{
		Packages:                packages,
		ConfigVersion:           s.ConfigVersion,
		PowerState:              s.PowerState,
		IsCrippled:              s.IsCrippled,
		CrippleMode:             s.CrippleMode,
		CrippleRecoveryAttempts: s.CrippleRecoveryAttempts,
	}
}
