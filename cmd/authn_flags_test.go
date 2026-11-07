package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/srlmgr/backend/cmd/config"
)

func TestPersistentSessionsFlag(t *testing.T) {
	old := config.AuthnPersistentSessionsEnabled
	t.Cleanup(func() { config.AuthnPersistentSessionsEnabled = old })
	cmd := &cobra.Command{}
	registerAuthnFlags(cmd)
	if config.AuthnPersistentSessionsEnabled {
		t.Fatal("persistent sessions must be disabled by default")
	}

	if err := cmd.ParseFlags([]string{"--authn-persistent-sessions-enabled"}); err != nil {
		t.Fatal(err)
	}
	if !config.AuthnPersistentSessionsEnabled {
		t.Fatal("persistent sessions flag was not applied")
	}
}

func TestPersistentSessionsConfiguration(t *testing.T) {
	old := config.AuthnPersistentSessionsEnabled
	t.Cleanup(func() { config.AuthnPersistentSessionsEnabled = old })
	t.Setenv("BACKEND_AUTHN_PERSISTENT_SESSIONS_ENABLED", "true")
	cmd := &cobra.Command{}
	registerAuthnFlags(cmd)
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	bindFlags(cmd, viper.New())
	if !config.AuthnPersistentSessionsEnabled {
		t.Fatal("persistent sessions environment variable was not applied")
	}
}
