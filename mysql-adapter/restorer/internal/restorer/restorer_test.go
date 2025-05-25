package restorer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_Redtore_UploadValidDump(t *testing.T) {
	ctx := context.Background()
	dbUser := "testuser"
	dbPass := "testpassword"
	dbName := "testdb"

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

	mysqlC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() {
		err := mysqlC.Terminate(ctx)
		if err != nil {
			panic(err)
		}
	}()

	dbhost, _ := mysqlC.ContainerIP(ctx)
	dbPort, _ := mysqlC.MappedPort(ctx, "3306")
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
