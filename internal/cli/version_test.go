package cli

import (
	"testing"
)

func TestVersionCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if versionCmd.Use != "version" {
			t.Errorf("expected Use to be 'version', got %s", versionCmd.Use)
		}

		if versionCmd.Short != "Show version information" {
			t.Errorf("expected Short to be 'Show version information', got %s", versionCmd.Short)
		}

		if versionCmd.Run == nil {
			t.Error("expected Run to be set")
		}
	})

	t.Run("version output", func(t *testing.T) {

		if versionCmd.Run == nil {
			t.Error("expected Run function to be set")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("version command panicked: %v", r)
			}
		}()

		versionCmd.Run(versionCmd, []string{})
	})
}
