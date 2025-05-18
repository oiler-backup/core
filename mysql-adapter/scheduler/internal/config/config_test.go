package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetConfig_Success(t *testing.T) {
	os.Clearenv()
	err := os.Setenv("SYSTEM_NAMESPACE", "test-system")
	require.NoError(t, err)
	err = os.Setenv("BACKUPER_VERSION", "myorg/my-backuper:latest")
	require.NoError(t, err)
	err = os.Setenv("RESTORER_VERSION", "myorg/my-restorer:latest")
	require.NoError(t, err)
	err = os.Setenv("PORT", "8080")
	require.NoError(t, err)

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, "test-system", cfg.SystemNamespace)
	assert.Equal(t, "myorg/my-backuper:latest", cfg.BackuperVersion)
	assert.Equal(t, "myorg/my-restorer:latest", cfg.RestorerVersion)
	assert.Equal(t, int64(8080), cfg.Port)
}

func Test_GetConfig_MissingRequiredField(t *testing.T) {
	os.Clearenv()

	_, err := GetConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SYSTEM_NAMESPACE")
}

func Test_GetConfig_UsesDefaults(t *testing.T) {
	os.Clearenv()
	err := os.Setenv("SYSTEM_NAMESPACE", "default-system")
	require.NoError(t, err)

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, "default-system", cfg.SystemNamespace)
	assert.Equal(t, "ashadrinnn/mysqlbackuper:0.0.1-0", cfg.BackuperVersion)
	assert.Equal(t, "sveb00/mysqlrestorer:0.0.1-1", cfg.RestorerVersion)
	assert.Equal(t, int64(50051), cfg.Port)
}

func Test_GetConfig_DefaultsWithOverride(t *testing.T) {
	os.Clearenv()
	err := os.Setenv("SYSTEM_NAMESPACE", "default-system")
	require.NoError(t, err)
	err = os.Setenv("PORT", "9090")
	require.NoError(t, err)

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, "default-system", cfg.SystemNamespace)
	assert.Equal(t, "ashadrinnn/mysqlbackuper:0.0.1-0", cfg.BackuperVersion)
	assert.Equal(t, "sveb00/mysqlrestorer:0.0.1-1", cfg.RestorerVersion)
	assert.Equal(t, int64(9090), cfg.Port)
}
