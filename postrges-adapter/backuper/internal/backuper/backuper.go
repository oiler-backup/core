package backuper

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
)

type ErrBackup = error

func buildBackupError(msg string, opts ...any) ErrBackup {
	return fmt.Errorf(msg, opts)
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
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		b.dbHost, b.dbPort, b.dbUser, b.dbPass, b.dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return buildBackupError("Failed to open driver for database: %+v", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return buildBackupError("Failed to connect to database: %+v", err)
	}

	dumpCmd := exec.CommandContext(ctx, "pg_dump",
		"-h", b.dbHost,
		"-p", b.dbPort,
		"-U", b.dbUser,
		"-d", b.dbName,
		"-F",
		"c",
		"-f", b.backupPath,
	)
	dumpCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", b.dbPass))

	output, err := dumpCmd.CombinedOutput()
	if err != nil {
		return buildBackupError("Failed executing pg_dump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
