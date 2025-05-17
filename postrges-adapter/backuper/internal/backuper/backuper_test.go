package backuper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_Backup_CreatesValidDump(t *testing.T) {
	ctx := context.Background()

	req := tc.ContainerRequest{
		Image:        "postgres:14",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	postgresC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer postgresC.Terminate(ctx)

	host, _ := postgresC.Host(ctx)
	port, _ := postgresC.MappedPort(ctx, "5432")

	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "backup.dump")

	b := NewBackuper(
		host,
		port.Port(),
		"testuser",
		"testpass",
		"testdb",
		backupFile,
	)

	err = b.Backup(ctx)
	require.NoError(t, err)

	fileInfo, err := os.Stat(backupFile)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}

func Test_BuildBackup(t *testing.T) {
	message := "some message: %s"
	option := "option"
	err := buildBackupError(message, option)
	assert.Equal(t, fmt.Sprintf(message, option), err.Error())
}
