# Docker Test
![Master CI](https://github.com/akhettar/docker-db/workflows/Master%20CI/badge.svg?branch=master)
[![GoDoc](https://godoc.org/github.com/akhettar/docker-db?status.svg)](https://godoc.org/github.com/akhettar/docker-db)


![hard working man](pushing-cart.png)

This is a Go library to run database containers as part of running the integration tests. The following databases are supported:

* Postgres
* MongoDB - DocumentDB

# How to use

## Running Postgres container

An example of running `postgres` container is present in this project. See the following files: 
* [Integratino test](docker_test.go)
* [Test fixture](init_test.go)

## Running MongoDB container

`Integration test snipppet`

```go
func TestPublishAppStatus_WithInvalidAppPlatformReturnBadRequestResponse(t *testing.T) {

	t.Logf("Given the app status api is up and running")
	{
		platform := "dummy"
		t.Logf("\tWhen Sending Publish App status request to endpoint with unsupported platform value:  \"%s\"", platform)
		{
		
            mockUnleash := test.GetMockUnleashClient(t)
			handler := NewAppStatusHandler(Repository, mockUnleash)
			router := handler.CreateRouter()
			version := "1.0"

			body := model.ReleaseRequest{Version: version, Platform: platform}
			req, err := test.HttpRequest(body, "/status", http.MethodPost, test.ValidToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// check call success
			test.Ok(err, t)

			if w.Code == http.StatusBadRequest {
				t.Logf("\t\tShould receive a \"%d\" status. %v", http.StatusBadRequest, test.CheckMark)
			} else {
				t.Errorf("\t\tShould receive a \"%d\" status. %v %v", http.StatusBadRequest, test.BallotX, w.Code)
			}
		}
	}
}
```

In the same package include a test file with the name: `init_test.go` and include the following

```go
import (
	"github.com/akhettar/app-features-manager/repository"
	"context"
	"flag"
	dockertdb "github.com/akhettar/docker-db"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"

	"net/http/httptest"
	"os"
	"testing"
)

const ProfileEnvVar = "PROFILE"

var (
	  
  // The repository instance configured to run against the Docker DB Test container
	Repository *repository.MongoRepository

)

// TestFixture wraps all tests with the needed initialized mock DB and fixtures
// This test runs before other integration test. It starts an instance of mongo db in the background (provided you have mongo
// installed on the server on which this test will be running) and shuts it down.
func TestMain(m *testing.M) {

	container := dockertest.StartMongoContainer()
	log.Printf("running mongo with Ip %s", container.Host())

	uri := fmt.Sprintf("mongodb://%s:%d", container.Host(), container.Port())
	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		panic(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err)
	}

	Repository = &repository.MongoRepository{client, repository.DBInfo{uri, repository.DefaultDBName, repository.DefaultCollection}}

	// Run the test suite
	retCode := m.Run()

	c.Destroy()

	// call with result of m.Run()
	os.Exit(retCode)
}

```

# License
[MIT](LICENSE)


