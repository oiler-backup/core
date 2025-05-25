// Package restorer contains entities to restore backup of Mongo Database.
package restorer

import (
	"context"
	"fmt"
	"os/exec"
)

// A Restorer restore backup of Mongo Database.
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
// It uses mongorestore command with appropriate flags.
func (r Restorer) Restore(ctx context.Context) error {
	cmd := exec.Command("mongorestore",
		"--host", r.dbHost,
		"--port", r.dbPort,
		"--username", r.dbUser,
		"--password", r.dbPass,
		"--gzip",
		"--drop",
		fmt.Sprintf("--nsInclude=%s.*", r.dbName),
		fmt.Sprint("--archive=", r.backupPath),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed executing mongorestore: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
