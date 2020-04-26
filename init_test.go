package dbtest

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
)

var (
	DB  *sql.DB
	err error
)

// Test Fixture start the DB container and run the test suite
func TestMain(m *testing.M) {

	// Start container
	c := StartPostgresContainerWithInitialisationScript("test", "initialise_db.sql")

	// open connection
	connStr := fmt.Sprintf("postgres://%s:%s@%s/test?sslmode=disable", c.userName, c.password, c.host)
	DB, err = sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("failed to open a connection")
	}
	defer DB.Close()

	// Run the test suite
	retCode := m.Run()

	// Kill the container
	c.Destroy()

	// call with result of m.Run()
	os.Exit(retCode)
}
