/*
Package dbtest provides a way of starting a MongoDB or Postgres docker
container prior to running the integration test suite.
This packages manages the life cycle of the of this docker container
it fires off the container, kill the container and remove its volume
after the test suite is completed.

Here is an example on how to run these supported db containers:

	import (
		"github.com/akhettar/docker-db"
	)
	func main() {

		// Starting Postgres container with an initialised schema file
		con1 := dbtest.StartPostgresContainerWithInitialisationScript("dbname", "schema.sql")
		fmt.Printf("mongo container with id running %s", con1.id)

		// Staring mongo container
		con2 := dbtest.StartMongoContainer()
		fmt.Printf("mongo container with id running %s", con2.id)
	}

*/
package dbtest
