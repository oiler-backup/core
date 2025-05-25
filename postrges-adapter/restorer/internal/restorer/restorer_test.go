package restorer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_Redtore_UploadValidDump(t *testing.T) {
	ctx := context.Background()
	dbUser := "testuser"
	dbPass := "testpass"
	dbName := "testdb"

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
	defer func() {
		err := postgresC.Terminate(ctx)
		if err != nil {
			panic(err)
		}
	}()

	dbhost, _ := postgresC.ContainerIP(ctx)
	dbPort, _ := postgresC.MappedPort(ctx, "5432")
	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "backup.dump")

	b := NewBackuper(
		dbhost,
		dbPort.Port(),
		dbUser,
		dbPass,
		dbName,
		backupFile,
	)

	err = b.Backup(ctx, false)

	r := NewRestorer(
		dbhost,
		dbPort.Port(),
		dbUser,
		dbPass,
		dbName,
		backupFile,
	)

	err = r.Restore(ctx)
	require.NoError(t, err)

	fileInfo, err := os.Stat(backupFile)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}
