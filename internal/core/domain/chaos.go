package domain

import "math/rand"

// ChaosSource provides randomness for chaos simulation.
// Using a seeded source allows deterministic testing.
type ChaosSource interface {
	Float64() float64
	Intn(n int) int
}

// defaultChaos uses the global math/rand functions.
type defaultChaos struct{}

func (defaultChaos) Float64() float64 { return rand.Float64() }
func (defaultChaos) Intn(n int) int   { return rand.Intn(n) }

// DefaultChaos is the production chaos source.
var DefaultChaos ChaosSource = defaultChaos{}

// ShouldActivityFail returns true 10% of the time.
func ShouldActivityFail(src ChaosSource) bool {
	return src.Float64() < 0.10
}

// ShouldCrippleClient returns true 3% of the time when an activity failed.
func ShouldCrippleClient(src ChaosSource, didFail bool) bool {
	if !didFail {
		return false
	}
	return src.Float64() < 0.03
}

// SelectCrippleMode picks a random cripple behavior.
func SelectCrippleMode(src ChaosSource) CrippleMode {
	modes := []CrippleMode{
		CrippleModeFailPackageOps,
		CrippleModeFailConfig,
		CrippleModeSilent,
	}
	return modes[src.Intn(len(modes))]
}

// ShouldDriftState returns true 5-10% of the time.
func ShouldDriftState(src ChaosSource) bool {
	return src.Float64() < 0.075 // midpoint of 5-10%
}

// ApplyDrift makes a spontaneous independent state change to simulate manual
// changes on the client (e.g., someone manually installing a package).
func ApplyDrift(src ChaosSource, state ClientState) ClientState {
	newState := state.Clone()
	choice := src.Intn(3)
	switch choice {
	case 0:
		// drift config version up by 1
		newState.ConfigVersion++
	case 1:
		// add a spontaneous package
		pkgName := randomDriftPackage(src)
		newState.Packages[pkgName] = "drifted"
	case 2:
		// remove a random package if any exist
		for pkg := range newState.Packages {
			delete(newState.Packages, pkg)
			break
		}
	}
	return newState
}

// IsCrippledForActivity returns true if the client's cripple mode prevents
// this activity from succeeding.
func IsCrippledForActivity(state ClientState, activityType ActivityType) bool {
	if !state.IsCrippled {
		return false
	}
	switch state.CrippleMode {
	case CrippleModeFailPackageOps:
		return activityType == ActivityInstallPackage ||
			activityType == ActivityUpgradePackage ||
			activityType == ActivityRemovePackage
	case CrippleModeFailConfig:
		return activityType == ActivityApplyConfig ||
			activityType == ActivityValidateConfig
	case CrippleModeSilent:
		return true // all activities silently fail
	}
	return false
}

func randomDriftPackage(src ChaosSource) string {
	names := []string{"curl", "wget", "vim", "jq", "htop", "tree", "tmux"}
	return names[src.Intn(len(names))]
}
