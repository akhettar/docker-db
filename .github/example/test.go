package example

import (
	"context"
	"fmt"
	dockerdb "github.com/akhettar/docker-db"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"testing"
)

func Test_testmongo(t *testing.T) {

	// Start mongo container
	c := dockerdb.StartMongoContainer()

	uri := fmt.Sprintf("mongodb://%s:%d", c.Host(), c.Port())
	clientOptions := options.Client().ApplyURI(uri)

	// instantiate mongo client
	client, err := mongo.Connect(context.TODO(), clientOptions)

}
