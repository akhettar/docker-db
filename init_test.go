package dockertest

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

var db *sql.DB

// Test Fixture start the DB container and run the test suite
func TestMain(m *testing.M) {

	flag.Parse()

	schema, err := filepath.Abs("initialise_db.sql")
	if err != nil {
		log.Println("Running postgres without the initialisation script")
	}

	// Start container
	c := StartPostgresContainerWithInitialisationScript("test", schema)
	connStr := fmt.Sprintf("postgres://%s:%s@%s/test?sslmode=disable", c.userName, c.password, c.host)
	db, err = sql.Open("postgres", connStr)
	defer db.Close()

	// Run the test suite
	retCode := m.Run()

	// Kill the container
	c.Destroy()

	// call with result of m.Run()
	os.Exit(retCode)
}
