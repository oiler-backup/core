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
		Image:           "mongo:8.0",
		ExposedPorts:    []string{"27017/tcp"},
		AlwaysPullImage: false,
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "root",
			"MONGO_INITDB_ROOT_PASSWORD": "pass",
		},
		WaitingFor: wait.ForListeningPort("27017/tcp"),
	}

	mongoC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer mongoC.Terminate(ctx)

	host, _ := mongoC.ContainerIP(ctx)

	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "backup.dump")

	b := NewBackuper(
		host,
		"27017",
		"root",
		"pass",
		"admin",
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
