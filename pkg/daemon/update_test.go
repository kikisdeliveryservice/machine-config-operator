package daemon

import (
	"fmt"
	"testing"

	ignv2_2types "github.com/coreos/ignition/config/v2_2/types"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// TestUpdateOS verifies the return errors from attempting to update the OS follow expectations
func TestUpdateOS(t *testing.T) {
	// expectedError is the error we will use when expecting an error to return
	expectedError := fmt.Errorf("broken")

	// testClient is the NodeUpdaterClient mock instance that will front
	// calls to update the host.
	testClient := RpmOstreeClientMock{
		GetBootedOSImageURLReturns: []GetBootedOSImageURLReturn{},
		RunPivotReturns: []error{
			// First run will return no error
			nil,
			// Second rrun will return our expected error
			expectedError},
	}

	// Create a Daemon instance with mocked clients
	d := Daemon{
		name:              "nodeName",
		OperatingSystem:   MachineConfigDaemonOSRHCOS,
		NodeUpdaterClient: testClient,
		loginClient:       nil, // set to nil as it will not be used within tests
		client:            fake.NewSimpleClientset(),
		kubeClient:        k8sfake.NewSimpleClientset(),
		rootMount:         "/",
		bootedOSImageURL:  "test",
	}

	// Set up machineconfigs to pass to updateOS.
	mcfg := &mcfgv1.MachineConfig{}
	// differentMcfg has a different OSImageURL so it will force Daemon.UpdateOS
	// to trigger an update of the operatingsystem (as fronted by our testClient)
	differentMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			OSImageURL: "somethingDifferent",
		},
	}

	// Test when the machine configs match. No operation should occur
	if err := d.updateOS(mcfg, mcfg); err != nil {
		t.Errorf("Expected no error. Got %s.", err)
	}
	// When machine configs differ but pivot succeeds we should get no error.
	if err := d.updateOS(mcfg, mcfg); err != nil {
		t.Errorf("Expected no error. Got %s.", err)
	}
	// When machine configs differ but pivot fails we should get the expected error.
	if err := d.updateOS(mcfg, differentMcfg); err == expectedError {
		t.Error("Expected an error. Got none.")
	}
}

// checkReconcilableResults is a shortcut for verifying results that should be reconcilable
func checkReconcilableResults(t *testing.T, key string, err error, isReconcilable bool) {
	if err != nil {
		t.Errorf("Expected no error. Got %s.", err)
	}
	if isReconcilable != true {
		t.Errorf("Expected the %s values would cause reconcilable. Received irreconcilable.", key)
	}
}

// checkIrreconcilableResults is a shortcut for verifing results that should be Irreconcilable
func checkIrreconcilableResults(t *testing.T, key string, err error, isReconcilable bool) {
	if err != nil {
		t.Errorf("Expected no error. Got %s.", err)
	}
	if isReconcilable != false {
		t.Errorf("Expected %s values would cause irreconcilable. Received reconcilable.", key)
	}
}

// TestReconcilable attempts to verify the conditions in which configs would and would not be
// reconcilable. Welcome to the longest unittest you've ever read.
func TestReconcilable(t *testing.T) {

	d := Daemon{
		name:              "nodeName",
		OperatingSystem:   MachineConfigDaemonOSRHCOS,
		NodeUpdaterClient: nil,
		loginClient:       nil,
		client:            nil,
		kubeClient:        nil,
		rootMount:         "/",
		bootedOSImageURL:  "test",
	}

	// oldConfig is the current config of the fake system
	oldConfig := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Ignition: ignv2_2types.Ignition{
					Version: "0.0",
				},
			},
		},
	}

	// newConfig is the config that is being requested to apply to the system
	newConfig := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Ignition: ignv2_2types.Ignition{
					Version: "1.0",
				},
			},
		},
	}

	// Verify Ignition version changes react as expected
	isReconcilable, _, err := d.reconcilable(oldConfig, newConfig)
	checkIrreconcilableResults(t, "ignition", err, isReconcilable)

	// Match ignition versions
	oldConfig.Spec.Config.Ignition.Version = "1.0"
	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkReconcilableResults(t, "ignition", err, isReconcilable)

	// Verify Networkd unit changes react as expected
	oldConfig.Spec.Config.Networkd = ignv2_2types.Networkd{}
	newConfig.Spec.Config.Networkd = ignv2_2types.Networkd{
		Units: []ignv2_2types.Networkdunit{
			ignv2_2types.Networkdunit{
				Name: "test",
			},
		},
	}
	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkIrreconcilableResults(t, "networkd", err, isReconcilable)

	// Match Networkd
	oldConfig.Spec.Config.Networkd = newConfig.Spec.Config.Networkd

	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkReconcilableResults(t, "networkd", err, isReconcilable)

	// Verify Disk changes react as expected
	oldConfig.Spec.Config.Storage.Disks = []ignv2_2types.Disk{
		ignv2_2types.Disk{
			Device: "one",
		},
	}

	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkIrreconcilableResults(t, "disk", err, isReconcilable)

	// Match storage disks
	newConfig.Spec.Config.Storage.Disks = oldConfig.Spec.Config.Storage.Disks
	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkReconcilableResults(t, "disk", err, isReconcilable)

	// Verify Filesystems changes react as expected
	oldConfig.Spec.Config.Storage.Filesystems = []ignv2_2types.Filesystem{
		ignv2_2types.Filesystem{
			Name: "test",
		},
	}

	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkIrreconcilableResults(t, "filesystem", err, isReconcilable)

	// Match Storage filesystems
	newConfig.Spec.Config.Storage.Filesystems = oldConfig.Spec.Config.Storage.Filesystems
	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkReconcilableResults(t, "filesystem", err, isReconcilable)

	// Verify Raid changes react as expected
	oldConfig.Spec.Config.Storage.Raid = []ignv2_2types.Raid{
		ignv2_2types.Raid{
			Name: "test",
		},
	}

	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkIrreconcilableResults(t, "raid", err, isReconcilable)

	// Match storage raid
	newConfig.Spec.Config.Storage.Raid = oldConfig.Spec.Config.Storage.Raid
	isReconcilable, _, err = d.reconcilable(oldConfig, newConfig)
	checkReconcilableResults(t, "raid", err, isReconcilable)
}

func TestReconcilableSameSSH(t *testing.T) {
	// expectedError is the error we will use when expecting an error to return
	expectedError := fmt.Errorf("broken")

	// testClient is the NodeUpdaterClient mock instance that will front
	// calls to update the host.
	testClient := RpmOstreeClientMock{
		GetBootedOSImageURLReturns: []GetBootedOSImageURLReturn{},
		RunPivotReturns: []error{
			// First run will return no error
			nil,
			// Second rrun will return our expected error
			expectedError},
	}

	// Create a Daemon instance with mocked clients
	d := Daemon{
		name:              "nodeName",
		OperatingSystem:   MachineConfigDaemonOSRHCOS,
		NodeUpdaterClient: testClient,
		loginClient:       nil, // set to nil as it will not be used within tests
		client:            fake.NewSimpleClientset(),
		kubeClient:        k8sfake.NewSimpleClientset(),
		rootMount:         "/",
		bootedOSImageURL:  "test",
	}

	// Set up machineconfigs that have identical SSH keys
	tempUser1 := ignv2_2types.PasswdUser{SSHAuthorizedKeys: []ignv2_2types.SSHAuthorizedKey{"1234"}}
	oldMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Passwd: ignv2_2types.Passwd{
					Users: []ignv2_2types.PasswdUser{tempUser1},
				},
			},
		},
	}
	newMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Passwd: ignv2_2types.Passwd{
					Users: []ignv2_2types.PasswdUser{tempUser1},
				},
			},
		},
	}

	canReconcile, toChange, err := d.reconcilable(oldMcfg, newMcfg)

	checkReconcilableResults(t, "ssh", err, canReconcile)

	if len(toChange) != 0 {
		t.Errorf("List of user indices with SSH differences should be empty")
	}
}
func TestReconcilableChangedSSH(t *testing.T) {
	// expectedError is the error we will use when expecting an error to return
	expectedError := fmt.Errorf("broken")

	// testClient is the NodeUpdaterClient mock instance that will front
	// calls to update the host.
	testClient := RpmOstreeClientMock{
		GetBootedOSImageURLReturns: []GetBootedOSImageURLReturn{},
		RunPivotReturns: []error{
			// First run will return no error
			nil,
			// Second rrun will return our expected error
			expectedError},
	}

	// Create a Daemon instance with mocked clients
	d := Daemon{
		name:              "nodeName",
		OperatingSystem:   MachineConfigDaemonOSRHCOS,
		NodeUpdaterClient: testClient,
		loginClient:       nil, // set to nil as it will not be used within tests
		client:            fake.NewSimpleClientset(),
		kubeClient:        k8sfake.NewSimpleClientset(),
		rootMount:         "/",
		bootedOSImageURL:  "test",
	}

	// Set up machineconfigs that are identical except for SSH keys
	tempUser1 := ignv2_2types.PasswdUser{SSHAuthorizedKeys: []ignv2_2types.SSHAuthorizedKey{"1234"}}
	tempUser2 := ignv2_2types.PasswdUser{SSHAuthorizedKeys: []ignv2_2types.SSHAuthorizedKey{"5678"}}
	oldMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Passwd: ignv2_2types.Passwd{
					Users: []ignv2_2types.PasswdUser{tempUser1},
				},
			},
		},
	}
	newMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Passwd: ignv2_2types.Passwd{
					Users: []ignv2_2types.PasswdUser{tempUser2},
				},
			},
		},
	}

	canReconcile, toChange, err := d.reconcilable(oldMcfg, newMcfg)

	checkReconcilableResults(t, "ssh", err, canReconcile)

	if len(toChange) == 0 {
		t.Errorf("List of user indices with SSH differences should not be empty")
	}
}

func TestWriteSSHKeys(t *testing.T) {
	// expectedError is the error we will use when expecting an error to return
	expectedError := fmt.Errorf("broken")

	// testClient is the NodeUpdaterClient mock instance that will front
	// calls to update the host.
	testClient := RpmOstreeClientMock{
		GetBootedOSImageURLReturns: []GetBootedOSImageURLReturn{},
		RunPivotReturns: []error{
			// First run will return no error
			nil,
			// Second rrun will return our expected error
			expectedError},
	}

	mockFS := &FsClientMock{MkdirAllReturns: []error{nil}, WriteFileReturns: []error{nil}}

	// Create a Daemon instance with mocked clients
	d := Daemon{
		name:              "nodeName",
		OperatingSystem:   MachineConfigDaemonOSRHCOS,
		NodeUpdaterClient: testClient,
		loginClient:       nil, // set to nil as it will not be used within tests
		client:            fake.NewSimpleClientset(),
		kubeClient:        k8sfake.NewSimpleClientset(),
		rootMount:         "/",
		bootedOSImageURL:  "test",
		fileSystemClient:  mockFS,
	}

	// Set up machineconfigs that are identical except for SSH keys
	tempUser1 := ignv2_2types.PasswdUser{SSHAuthorizedKeys: []ignv2_2types.SSHAuthorizedKey{"1234"}}
	tempUser2 := ignv2_2types.PasswdUser{SSHAuthorizedKeys: []ignv2_2types.SSHAuthorizedKey{"5678"}}
	oldMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Passwd: ignv2_2types.Passwd{
					Users: []ignv2_2types.PasswdUser{tempUser1},
				},
			},
		},
	}
	newMcfg := &mcfgv1.MachineConfig{
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignv2_2types.Config{
				Passwd: ignv2_2types.Passwd{
					Users: []ignv2_2types.PasswdUser{tempUser2},
				},
			},
		},
	}

	tempIndices := []int{0}
	err := d.writeSSHKeys(oldMcfg.Spec.Config.Passwd.Users, newMcfg.Spec.Config.Passwd.Users, tempIndices)
	if err != nil {
		t.Errorf("Expected no error. Got %s.", err)
	}

	newMcfg.Spec.Config.Passwd.Users = oldMcfg.Spec.Config.Passwd.Users
	err = d.writeSSHKeys(oldMcfg.Spec.Config.Passwd.Users, newMcfg.Spec.Config.Passwd.Users, tempIndices)
	if err == nil {
		t.Errorf("Expected error, SSHKeys identical, nothing to write")
	}
}
