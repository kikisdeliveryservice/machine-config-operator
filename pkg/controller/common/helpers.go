package common

import (
	"io/ioutil"
	"reflect"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	validate2 "github.com/coreos/ignition/config/validate"
	ign3types "github.com/coreos/ignition/v2/config/v3_0/types"
	validate3 "github.com/coreos/ignition/v2/config/validate"
	ignconverter "github.com/coreos/ign-converter/ign2to3"
	"github.com/golang/glog"
	errors "github.com/pkg/errors"
)

// NewIgnConfig returns an empty ignition config with version set as latest version
func NewIgnConfig() ign2types.Config {
	return ign2types.Config{
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

// ConvertIgnition3to2 takes an igntion v3 config and returns a v2 config
ConvertIgnition3to2(ignconfig ign3types.Config) ign2types.Config, error {
	converted2, err := ignitionconverter.Translate3to2(ignconfig)

	if err = nil {
		return converted2, nil
	}
	return nil, err
}

// ValidateIgnition validates both igntion2 and ignition 3 configs
func ValidateIgnition(ignconfig interface{}) error {
	switch ign := ignconfig.(type) {
	case ign2types.Config:
		return validateIgnition2(ign)
	case ign3types.Config:
		return validateIgnition3(ign)
	default:
		return errors.Errorf("unrecognized ignition type")

	}
}

// ValidateIgnition2 wraps the underlying Ignition validation, but explicitly supports
// a completely empty Ignition config as valid.  This is because we
// want to allow MachineConfig objects which just have e.g. KernelArguments
// set, but no Ignition config.
// Returns nil if the config is valid (per above) or an error containing a Report otherwise.
func validateIgnition2(cfg ign2types.Config) error {
	// only validate if Ignition Config is not empty
	if reflect.DeepEqual(ign2types.Config{}, cfg) {
		return nil
	}
	if report := validate2.ValidateWithoutSource(reflect.ValueOf(cfg)); report.IsFatal() {
		return errors.Errorf("invalid Ignition config found: %v", report)
	}
	return nil
}

// ValidateIgnition3 wraps the underlying Ignition validation, but explicitly supports
// a completely empty Ignition config as valid.  This is because we
// want to allow MachineConfig objects which just have e.g. KernelArguments
// set, but no Ignition config.
// Returns nil if the config is valid (per above) or an error containing a Report otherwise.
func validateIgnition3(cfg ign3types.Config) error {
	// only validate if Ignition Config is not empty
	if reflect.DeepEqual(ign3types.Config{}, cfg) {
		return nil
	}
	if report := validate3.ValidateWithContext(cfg, nil); report.IsFatal() {
		return errors.Errorf("invalid Ignition config found: %v", report)
	}
	return nil
}
