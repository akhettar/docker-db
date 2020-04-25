package dockertest

import (
	"database/sql"
	_ "github.com/lib/pq"
	"strings"
	"testing"
)

type Entries struct {
	Article string
	Dealer  string
	price   string
}

func Test_StartPostgresConnection(t *testing.T) {

	// db should have been initialised by init_test.go
	rows, err := db.Query("select * from shop;")

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	var expectedEntries []Entries = []Entries{
		{"1", "A", "3.45"},
		{"1", "B", "3.99"},
		{"2", "A", "10.99"}}

	// Fetch rows
	j := 0
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		expectedValues := []string{expectedEntries[j].Article, expectedEntries[j].Dealer, expectedEntries[j].price}

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string

		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = strings.TrimSpace(string(col))
			}

			if value != expectedValues[i] {
				t.Errorf("Found wrong entry %s it should have been %s", value, expectedEntries[i].Article)
			} else {
				t.Logf("Found the correct entry %s", value)
			}
		}
		j++
	}

}
