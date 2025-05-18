package backuper

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"

	_ "github.com/go-sql-driver/mysql"
)

type ErrBackup = error

func buildBackupError(msg string, opts ...any) ErrBackup {
	return fmt.Errorf(msg, opts...)
}

type Backuper struct {
	dbHost string
	dbPort string
	dbUser string
	dbPass string
	dbName string

	backupPath string
}

func NewBackuper(dbHost, dbPort, dbUser, dbPassword, dbName, backupPath string) Backuper {
	return Backuper{
		dbHost:     dbHost,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPass:     dbPassword,
		dbName:     dbName,
		backupPath: backupPath,
	}
}

func (b Backuper) Backup(ctx context.Context) error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", b.dbUser, b.dbPass, b.dbHost, b.dbPort, b.dbName)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return buildBackupError("Failed to open driver for database: %+v", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil { // coverage-ignore
		return buildBackupError("Failed to connect to database: %+v", err)
	}

	dumpCmd := exec.CommandContext(ctx, "mysqldump",
		"-h", b.dbHost,
		"-P", b.dbPort,
		"-u", b.dbUser,
		fmt.Sprintf("-p%s", b.dbPass),
		b.dbName,
		"--ssl-mode=DISABLED",
		"--result-file", b.backupPath,
	)

	output, err := dumpCmd.CombinedOutput()
	if err != nil { // coverage-ignore
		return buildBackupError("Failed executing mysqldump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
