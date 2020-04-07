package dockertest

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"testing"
	"time"

	"camlistore.org/pkg/netutil"
)

/// runLongTest checks all the conditions for running a docker container
// based on image.
func runLongTest(image string) {
	if testing.Short() {
		log.Println("skipping in short mode")
	}
	if !haveDocker() {
		log.Fatal("'docker' command not found")
	}
	if ok, err := haveImage(image); !ok || err != nil {
		if err != nil {
			log.Println("Error running docker to check for %s: %v", image, err)
		}
		log.Printf("Pulling docker image %s ...", image)
		if err := Pull(image); err != nil {
			log.Println("Error pulling %s: %v", image, err)
		}
	}
}

// haveDocker returns whether the "docker" command was found.
func haveDocker() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func haveImage(name string) (ok bool, err error) {
	out, err := exec.Command("docker", "images", "--no-trunc").Output()
	if err != nil {
		return
	}
	return bytes.Contains(out, []byte(name)), nil
}

func run(args ...string) (containerID string, err error) {
	cmd := exec.Command("docker", append([]string{"run"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err = cmd.Run(); err != nil {
		err = fmt.Errorf("%v%v", stderr.String(), err)
		return
	}
	containerID = strings.TrimSpace(stdout.String())
	if containerID == "" {
		return "", errors.New("unexpected empty output from `docker run`")
	}
	return
}

func KillContainer(container string) error {
	return exec.Command("docker", "kill", container).Run()
}

// Pull retrieves the docker image with 'docker pull'.
func Pull(image string) error {
	out, err := exec.Command("docker", "pull", image).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v: %s", err, out)
	}
	return err
}

// IP returns the IP address of the container.
func IP(containerID string) (string, error) {
	return "127.0.0.1", nil
}

type ContainerID string

func (c ContainerID) IP() (string, error) {
	return IP(string(c))
}

func (c ContainerID) Kill() error {
	return KillContainer(string(c))
}

// Remove runs "docker rm" on the container
func (c ContainerID) Remove() error {
	return exec.Command("docker", "rm", string(c)).Run()
}

// KillRemove calls Kill on the container, and then Remove if there was
// no error. It logs any error to t.
func (c ContainerID) KillRemove() {
	if err := c.Kill(); err != nil {
		log.Println(err)
		return
	}
	if err := c.Remove(); err != nil {
		log.Println(err)
	}
}

// lookup retrieves the ip address of the container, and tries to reach
// before timeout the tcp address at this ip and given port.
func (c ContainerID) lookup(port int, timeout time.Duration) (ip string, err error) {
	ip, err = c.IP()
	if err != nil {
		err = fmt.Errorf("error getting IP: %v", err)
		return
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	err = netutil.AwaitReachable(addr, timeout)
	return
}

// setupContainer sets up a container, using the start function to run the given image.
// It also looks up the IP address of the container, and tests this address with the given
// port and timeout. It returns the container ID and its IP address, or makes the test
// fail on error.
func setupContainer(image string, port int, timeout time.Duration,
	start func() (string, error)) (c ContainerID, ip string) {
	runLongTest(image)

	containerID, err := start()
	if err != nil {
		log.Fatalf("docker run: %v", err)
	}
	c = ContainerID(containerID)
	ip, err = c.lookup(port, timeout)
	if err != nil {
		c.KillRemove()
		log.Printf("Container %v setup failed: %v", c, err)
	}
	return
}

const (
	mongoImage       = "library/mongo"
	mysqlImage       = "library/mysql"
	MySQLUsername    = "root"
	MySQLPassword    = "root"
	postgresImage    = "library/postgres"
	PostgresUsername = "docker" // set up by the dockerfile of postgresImage
	PostgresPassword = "docker" // set up by the dockerfile of postgresImage
)

// SetupMongoContainer sets up a real MongoDB instance for testing purposes,
// using a Docker container. It returns the container ID and its IP address,
// or makes the test fail on error.
// Currently using https://hub.docker.com/_/mongo/
func SetupMongoContainer() (c ContainerID, ip string) {
	return setupContainer(mongoImage, 27017, 10*time.Second, func() (string, error) {
		return run("-d", "-p", "27017:27017", mongoImage)
	})
}

// SetupMySQLContainer sets up a real MySQL instance for testing purposes,
// using a Docker container. It returns the container ID and its IP address,
// or makes the test fail on error.
// Currently using https://hub.docker.com/_/mysql/
func SetupMySQLContainer(dbname string) (c ContainerID, ip string) {
	return setupContainer(mysqlImage, 3306, 10*time.Second, func() (string, error) {
		return run("-d", "-e", "MYSQL_ROOT_PASSWORD="+MySQLPassword, "-e", "MYSQL_DATABASE="+dbname, mysqlImage)
	})
}

// SetupPostgreSQLContainer sets up a real PostgreSQL instance for testing purposes,
// using a Docker container. It returns the container ID and its IP address,
// or makes the test fail on error.
// Currently using https://hub.docker.com/_/postgres/
func SetupPostgreSQLContainer(dbname string) (c ContainerID, ip string) {
	c, ip = setupContainer(postgresImage, 5432, 15*time.Second, func() (string, error) {
		return run("-d", postgresImage)
	})
	cleanupAndDie := func(err error) {
		c.KillRemove()
		log.Fatal(err)
	}
	rootdb, err := sql.Open("postgres",
		fmt.Sprintf("user=%s password=%s host=%s dbname=postgres sslmode=disable", PostgresUsername, PostgresPassword, ip))
	if err != nil {
		cleanupAndDie(fmt.Errorf("Could not open postgres rootdb: %v", err))
	}
	if _, err := sqlExecRetry(rootdb,
		"CREATE DATABASE "+dbname+" LC_COLLATE = 'C' TEMPLATE = template0",
		50); err != nil {
		cleanupAndDie(fmt.Errorf("Could not create database %v: %v", dbname, err))
	}
	return
}

// sqlExecRetry keeps calling http://golang.org/pkg/database/sql/#DB.Exec on db
// with stmt until it succeeds or until it has been tried maxTry times.
// It sleeps in between tries, twice longer after each new try, starting with
// 100 milliseconds.
func sqlExecRetry(db *sql.DB, stmt string, maxTry int) (sql.Result, error) {
	if maxTry <= 0 {
		return nil, errors.New("did not try at all")
	}
	interval := 100 * time.Millisecond
	try := 0
	var err error
	var result sql.Result
	for {
		result, err = db.Exec(stmt)
		if err == nil {
			return result, nil
		}
		try++
		if try == maxTry {
			break
		}
		time.Sleep(interval)
		interval *= 2
	}
	return result, fmt.Errorf("failed %v times: %v", try, err)
}
