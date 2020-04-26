package dbtest

import "fmt"

// Examples on how to fire up containers: mongo and postgres
func ExampleContainer() {

	// Starting Postgres container with an initialised schema file
	con1 := StartPostgresContainerWithInitialisationScript("dbname", "schema.sql")
	fmt.Printf("mongo container with id running %s", con1.id)

	// Staring mongo container
	con2 := StartMongoContainer()
	fmt.Printf("mongo container with id running %s", con2.id)
}
