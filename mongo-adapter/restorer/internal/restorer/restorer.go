package restorer

import (
	"context"
	"fmt"
	"os/exec"
)

type Resotrer struct {
	dbHost string
	dbPort string
	dbUser string
	dbPass string
	dbName string

	backupPath string
}

func NewRestorer(dbHost, dbPort, dbUser, dbPassword, dbName, backupPath string) Resotrer {
	return Resotrer{
		dbHost:     dbHost,
		dbPort:     dbPort,
		dbUser:     dbUser,
		dbPass:     dbPassword,
		dbName:     dbName,
		backupPath: backupPath,
	}
}

func (r Resotrer) Restore(ctx context.Context) error {
	cmd := exec.Command("mongorestore",
		"--host", r.dbHost,
		"--port", r.dbPort,
		"--username", r.dbUser,
		"--password", r.dbPass,
		"-db", r.dbName,
		"--drop",
		r.backupPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed executing pg_dump: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
