package domain

// ActivityType enumerates supported activity types.
type ActivityType string

const (
	ActivityReboot          ActivityType = "reboot"
	ActivityInstallPackage  ActivityType = "install_package"
	ActivityUpgradePackage  ActivityType = "upgrade_package"
	ActivityRemovePackage   ActivityType = "remove_package"
	ActivityApplyConfig     ActivityType = "apply_config"
	ActivityValidateConfig  ActivityType = "validate_config"
	ActivityRunScript       ActivityType = "run_script"
)

// ValidActivityTypes is the set of all valid activity types.
var ValidActivityTypes = map[ActivityType]struct{}{
	ActivityReboot:         {},
	ActivityInstallPackage: {},
	ActivityUpgradePackage: {},
	ActivityRemovePackage:  {},
	ActivityApplyConfig:    {},
	ActivityValidateConfig: {},
	ActivityRunScript:      {},
}

// Activity represents a workflow activity dispatched to a client.
type Activity struct {
	Type   ActivityType           `json:"type"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// IsValid returns true if the activity type is recognized.
func (a Activity) IsValid() bool {
	_, ok := ValidActivityTypes[a.Type]
	return ok
}

// ResultStatus enumerates possible activity result statuses.
type ResultStatus string

const (
	ResultSuccess ResultStatus = "success"
	ResultFail    ResultStatus = "fail"
	ResultError   ResultStatus = "error"
)

// ActivityResult captures the outcome of executing an activity.
type ActivityResult struct {
	Status   ResultStatus           `json:"status"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	ErrorMsg string                 `json:"error_msg,omitempty"`
}

// DispatchMessage is a command from the server to a client.
type DispatchMessage struct {
	RunID    string                 `json:"run_id"`
	WfID     string                 `json:"wf_id"`
	ClientID string                 `json:"client_id"`
	Activity Activity               `json:"activity"`
}

// ResultMessage is the response from a client to the server.
type ResultMessage struct {
	RunID      string                 `json:"run_id"`
	WfID       string                 `json:"wf_id"`
	ClientID   string                 `json:"client_id"`
	Status     ResultStatus           `json:"status"`
	InnerState *ClientState           `json:"inner_state,omitempty"`
	ErrorMsg   string                 `json:"error_msg,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// ExecuteActivity applies an activity to a ClientState and returns the updated
// state and result. Pure domain logic, no side effects.
func ExecuteActivity(activity Activity, state ClientState) (ClientState, ActivityResult) {
	newState := state.Clone()

	switch activity.Type {
	case ActivityReboot:
		return executeReboot(newState)
	case ActivityInstallPackage:
		return executeInstallPackage(activity, newState)
	case ActivityUpgradePackage:
		return executeUpgradePackage(activity, newState)
	case ActivityRemovePackage:
		return executeRemovePackage(activity, newState)
	case ActivityApplyConfig:
		return executeApplyConfig(activity, newState)
	case ActivityValidateConfig:
		return executeValidateConfig(activity, newState)
	case ActivityRunScript:
		return executeRunScript(activity, newState)
	default:
		return newState, ActivityResult{
			Status:   ResultError,
			ErrorMsg: "unknown activity type: " + string(activity.Type),
		}
	}
}

func executeReboot(state ClientState) (ClientState, ActivityResult) {
	state.PowerState = PowerStateRestarting
	state.IsCrippled = false
	state.CrippleMode = CrippleModeNone
	state.CrippleRecoveryAttempts = 0
	state.PowerState = PowerStateOn
	return state, ActivityResult{Status: ResultSuccess}
}

func executeInstallPackage(activity Activity, state ClientState) (ClientState, ActivityResult) {
	pkg, version := extractPackageParams(activity)
	if pkg == "" {
		return state, ActivityResult{Status: ResultError, ErrorMsg: "missing package name"}
	}
	if _, exists := state.Packages[pkg]; exists {
		return state, ActivityResult{
			Status:   ResultFail,
			ErrorMsg: "package already installed: " + pkg,
		}
	}
	if version == "" {
		version = "latest"
	}
	state.Packages[pkg] = version
	return state, ActivityResult{
		Status:  ResultSuccess,
		Payload: map[string]interface{}{"package": pkg, "version": version},
	}
}

func executeUpgradePackage(activity Activity, state ClientState) (ClientState, ActivityResult) {
	pkg, version := extractPackageParams(activity)
	if pkg == "" {
		return state, ActivityResult{Status: ResultError, ErrorMsg: "missing package name"}
	}
	if _, exists := state.Packages[pkg]; !exists {
		return state, ActivityResult{
			Status:   ResultFail,
			ErrorMsg: "package not installed: " + pkg,
		}
	}
	if version == "" {
		version = "latest"
	}
	state.Packages[pkg] = version
	return state, ActivityResult{
		Status:  ResultSuccess,
		Payload: map[string]interface{}{"package": pkg, "version": version},
	}
}

func executeRemovePackage(activity Activity, state ClientState) (ClientState, ActivityResult) {
	pkg, _ := extractPackageParams(activity)
	if pkg == "" {
		return state, ActivityResult{Status: ResultError, ErrorMsg: "missing package name"}
	}
	if _, exists := state.Packages[pkg]; !exists {
		return state, ActivityResult{
			Status:   ResultFail,
			ErrorMsg: "package not installed: " + pkg,
		}
	}
	delete(state.Packages, pkg)
	return state, ActivityResult{
		Status:  ResultSuccess,
		Payload: map[string]interface{}{"package": pkg},
	}
}

func executeApplyConfig(activity Activity, state ClientState) (ClientState, ActivityResult) {
	version := extractConfigVersion(activity)
	state.ConfigVersion = version
	return state, ActivityResult{
		Status:  ResultSuccess,
		Payload: map[string]interface{}{"config_version": version},
	}
}

func executeValidateConfig(activity Activity, state ClientState) (ClientState, ActivityResult) {
	expected := extractConfigVersion(activity)
	if state.ConfigVersion != expected {
		return state, ActivityResult{
			Status:   ResultFail,
			ErrorMsg: "config version mismatch",
			Payload: map[string]interface{}{
				"expected": expected,
				"actual":   state.ConfigVersion,
			},
		}
	}
	return state, ActivityResult{
		Status:  ResultSuccess,
		Payload: map[string]interface{}{"config_version": state.ConfigVersion},
	}
}

func executeRunScript(activity Activity, state ClientState) (ClientState, ActivityResult) {
	script := ""
	if s, ok := activity.Params["script"].(string); ok {
		script = s
	}
	return state, ActivityResult{
		Status: ResultSuccess,
		Payload: map[string]interface{}{
			"exit_code": 0,
			"stdout":    "script executed: " + script,
			"stderr":    "",
		},
	}
}

func extractPackageParams(activity Activity) (pkg, version string) {
	if p, ok := activity.Params["package"].(string); ok {
		pkg = p
	}
	if v, ok := activity.Params["version"].(string); ok {
		version = v
	}
	return
}

func extractConfigVersion(activity Activity) int {
	if v, ok := activity.Params["config_version"].(int); ok {
		return v
	}
	if v, ok := activity.Params["config_version"].(float64); ok {
		return int(v)
	}
	return 0
}
