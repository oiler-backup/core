package backuper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_Backup_CreatesValidDump(t *testing.T) {
	ctx := context.Background()

	req := tc.ContainerRequest{
		Image:           "mysql:8.0",
		ExposedPorts:    []string{"3306/tcp"},
		AlwaysPullImage: false,
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "rootpassword",
			"MYSQL_USER":          "testuser",
			"MYSQL_PASSWORD":      "testpassword",
			"MYSQL_DATABASE":      "testdb",
		},
		WaitingFor: wait.ForListeningPort("3306/tcp"),
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
	host, _ := postgresC.ContainerIP(ctx)

	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "backup.dump")

	b := NewBackuper(
		host,
		"3306",
		"testuser",
		"testpassword",
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
