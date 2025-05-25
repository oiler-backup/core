// Package restorer contains entities to restore backup of Postgres Database.
package restorer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"

	_ "github.com/lib/pq"
)

type Restorer struct {
	dbHost string
	dbPort string
	dbUser string
	dbPass string
	dbName string

	backupPath string
}

// NewRestorer is a constructor for Restorer.
// Accepts parameters to connect to database and backupPath where backup will be stored locally.
func NewRestorer(dbHost, dbPort, dbUser, dbPassword, dbName, backupPath string) Restorer {
	return Restorer{
		dbHost:     dbHost,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPass:     dbPassword,
		dbName:     dbName,
		backupPath: backupPath,
	}
}

// Restore restores backup from local file.
// It uses postgres command with appropriate flags.
func (r Restorer) Restore(ctx context.Context) error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		r.dbHost, r.dbPort, r.dbUser, r.dbPass, r.dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open driver for database: %v", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	cmd := exec.Command("pg_restore",
		"-h", r.dbHost,
		"-p", r.dbPort,
		"-U", r.dbUser,
		"-d", r.dbName,
		"--no-owner",
		"--clean",
		r.backupPath,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", r.dbPass))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed executing pg_dump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
