// Package backuper contains entities to perform backup of PostgreSQL Database.
package backuper

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"

	_ "github.com/lib/pq"
)

// An ErrBackup is required for more verbosity.
type ErrBackup = error

// buildBackupError builds ErrBackup.
// Operates over f-strings.
func buildBackupError(msg string, opts ...any) ErrBackup {
	return fmt.Errorf(msg, opts...)
}

// A Backuper performs backup of PostgreSQL Database.
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

// Backup performs backup of PostgreSQL Database by using pg_dump CLI.
func (b Backuper) Backup(ctx context.Context, secure bool) error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		b.dbHost, b.dbPort, b.dbUser, b.dbPass, b.dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil { // coverage-ignore
		return buildBackupError("Failed to open driver for database: %+v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}()
	err = db.PingContext(ctx)
	if err != nil { // coverage-ignore
		return buildBackupError("Failed to connect to database: %+v", err)
	}

	args := []string{
		"-h", b.dbHost,
		"-p", b.dbPort,
		"-U", b.dbUser,
		"-d", b.dbName,
		"-F",
		"c",
		"-f", b.backupPath,
	}

	dumpCmd := exec.CommandContext(ctx, "pg_dump",
		args...,
	)
	dumpCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", b.dbPass))

	output, err := dumpCmd.CombinedOutput()
	if err != nil { // coverage-ignore
		return buildBackupError("Failed executing pg_dump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
