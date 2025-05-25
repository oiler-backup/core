package restorer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "backuper/internal/backuper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_Redtore_UploadValidDump(t *testing.T) {
	ctx := context.Background()
	dbUser := "root"
	dbPass := "pass"
	dbName := "admin"

	req := tc.ContainerRequest{
		Image:           "mongo:8.0",
		ExposedPorts:    []string{"27017/tcp"},
		AlwaysPullImage: false,
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": dbUser,
			"MONGO_INITDB_ROOT_PASSWORD": dbPass,
		},
		WaitingFor: wait.ForListeningPort("27017/tcp"),
	}

	mongoC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() {
		err := mongoC.Terminate(ctx)
		if err != nil {
			panic(err)
		}
	}()

	dbhost, _ := mongoC.ContainerIP(ctx)
	dbPort, _ := mongoC.MappedPort(ctx, "27017")
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
