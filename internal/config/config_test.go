package config

import "testing"

func TestLoadUsesRailwayPortWhenListenAddrUnset(t *testing.T) {
	t.Setenv("MATRIX_API_LISTEN", "")
	t.Setenv("PORT", "8080")
	t.Setenv("GOMUKS_ROOT", "")
	t.Setenv("RAILWAY_VOLUME_MOUNT_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got, want := cfg.ListenAddr, "0.0.0.0:8080"; got != want {
		t.Fatalf("ListenAddr = %q, want %q", got, want)
	}
}

func TestLoadUsesRailwayVolumeMountWhenStateDirUnset(t *testing.T) {
	t.Setenv("MATRIX_API_LISTEN", "")
	t.Setenv("PORT", "")
	t.Setenv("GOMUKS_ROOT", "")
	t.Setenv("RAILWAY_VOLUME_MOUNT_PATH", "/data")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got, want := cfg.StateDir, "/data/gomuks"; got != want {
		t.Fatalf("StateDir = %q, want %q", got, want)
	}
}

func TestLoadUsesManageSecret(t *testing.T) {
	t.Setenv("MATRIX_API_LISTEN", "")
	t.Setenv("PORT", "")
	t.Setenv("GOMUKS_ROOT", "")
	t.Setenv("RAILWAY_VOLUME_MOUNT_PATH", "")
	t.Setenv("EASYMATRIX_MANAGE_SECRET", "super-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got, want := cfg.ManageSecret, "super-secret"; got != want {
		t.Fatalf("ManageSecret = %q, want %q", got, want)
	}
}

func TestLoadUsesEphemeralRootAndInlineAPNSKey(t *testing.T) {
	t.Setenv("GOMUKS_ROOT", "")
	t.Setenv("RAILWAY_VOLUME_MOUNT_PATH", "")
	t.Setenv("EASYMATRIX_EPHEMERAL_ROOT", "/tmp/easymatrix-test")
	t.Setenv("APNS_KEY", "private-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got, want := cfg.EphemeralDir, "/tmp/easymatrix-test"; got != want {
		t.Fatalf("EphemeralDir = %q, want %q", got, want)
	}
	if got, want := cfg.APNSKey, "private-key"; got != want {
		t.Fatalf("APNSKey = %q, want %q", got, want)
	}
}
