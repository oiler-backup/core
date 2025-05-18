package backuper

import (
	"context"
	"fmt"
	"os/exec"
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
	dumpCmd := exec.CommandContext(ctx, "mongodump",
		"--host", b.dbHost,
		"--port", b.dbPort,
		"--username", b.dbUser,
		"--password", b.dbPass,
		"--db", b.dbName,
		"--authenticationDatabase", "admin",
		fmt.Sprint("--archive=", b.backupPath),
		"--tlsInsecure",
	)

	output, err := dumpCmd.CombinedOutput()
	if err != nil {
		return buildBackupError("Failed executing mongodump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
