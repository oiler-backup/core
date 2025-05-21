package restorer

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"

	_ "github.com/go-sql-driver/mysql"
)

type Restorer struct {
	dbHost string
	dbPort string
	dbUser string
	dbPass string
	dbName string

	backupPath string
}

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

func (r Restorer) Restore(ctx context.Context) error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", r.dbUser, r.dbPass, r.dbHost, r.dbPort, r.dbName)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("failed to open driver for database: %v", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	cmd := exec.Command("mysql",
		"-h", r.dbHost,
		"-P", r.dbPort,
		"-u", r.dbUser,
		fmt.Sprintf("-p%s", r.dbPass),
		r.dbName,
		"<", r.backupPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed executing mysql restore: %+v\n.Output:%s", err, string(output))
	}
	return nil
}
