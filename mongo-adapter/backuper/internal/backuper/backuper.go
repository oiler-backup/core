// Package backuper contains entities to perform backup of Mongo Database.
package backuper

import (
	"context"
	"fmt"
	"os/exec"
)

// An ErrBackup is required for more verbosity.
type ErrBackup = error

// buildBackupError builds ErrBackup.
// Operates over f-strings.
func buildBackupError(msg string, opts ...any) ErrBackup {
	return fmt.Errorf(msg, opts...)
}

// A Backuper performs backup of Mongo Database.
type Backuper struct {
	dbHost string
	dbPort string
	dbUser string
	dbPass string
	dbName string

	backupPath string
}

// NewBackuper is a constructor for Backuper.
// Accepts parameters to connect to database and backupPath where backup will be stored locally.
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

// Backup performs backup of Mongo Database by using mongodump CLI.
func (b Backuper) Backup(ctx context.Context, secure bool) error {
	args := []string{
		"--host", b.dbHost,
		"--port", b.dbPort,
		"--username", b.dbUser,
		"--password", b.dbPass,
		"--db", b.dbName,
		"--authenticationDatabase", "admin",
		fmt.Sprint("--archive=", b.backupPath),
	}

	if secure {
		args = append(args, "--tls-insecure")
	}
	dumpCmd := exec.CommandContext(ctx, "mongodump",
		args...,
	)

	output, err := dumpCmd.CombinedOutput()
	if err != nil {
		return buildBackupError("Failed executing mongodump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
