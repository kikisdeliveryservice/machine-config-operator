package common

import (
	"io/ioutil"
	"reflect"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	validate "github.com/coreos/ignition/config/validate"
	ign3types "github.com/coreos/ignition/v2/config/v3_0/types"
	"github.com/golang/glog"
	errors "github.com/pkg/errors"
)

// NewIgnConfig returns an empty ignition config with version set as latest version
func NewIgnConfig() ign2types.Config {
	return igntypes.Config{
		Ignition: ign2types.Ignition{
			Version: ign2types.MaxVersion.String(),
		},
	}
}

// WriteTerminationError writes to the Kubernetes termination log.
func WriteTerminationError(err error) {
	msg := err.Error()
	ioutil.WriteFile("/dev/termination-log", []byte(msg), 0644)
	glog.Fatal(msg)
}

// ValidateIgnition2 wraps the underlying Ignition validation, but explicitly supports
// a completely empty Ignition config as valid.  This is because we
// want to allow MachineConfig objects which just have e.g. KernelArguments
// set, but no Ignition config.
// Returns nil if the config is valid (per above) or an error containing a Report otherwise.
func ValidateIgnition2(cfg ign2types.Config) error {
	// only validate if Ignition Config is not empty
	if reflect.DeepEqual(ign2types.Config{}, cfg) {
		return nil
	}
	if report := validate.ValidateWithoutSource(reflect.ValueOf(cfg)); report.IsFatal() {
		return errors.Errorf("invalid Ignition config found: %v", report)
	}
	return nil
}

// ValidateIgnition3 wraps the underlying Ignition validation, but explicitly supports
// a completely empty Ignition config as valid.  This is because we
// want to allow MachineConfig objects which just have e.g. KernelArguments
// set, but no Ignition config.
// Returns nil if the config is valid (per above) or an error containing a Report otherwise.
func ValidateIgnition3(cfg ign3types.Config) error {
	// only validate if Ignition Config is not empty
	if reflect.DeepEqual(ign3types.Config{}, cfg) {
		return nil
	}
	if report := validate.ValidateWithoutSource(reflect.ValueOf(cfg)); report.IsFatal() {
		return errors.Errorf("invalid Ignition config found: %v", report)
	}
	return nil
}
