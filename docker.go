// Copyright 2020 Ayache Khettar. All rights reserved.
// Use of this source file is governed by MIT license
// license that can be found in LICENSE file.

// Package dbtest provides a way of starting a MongoDB or Postgres docker
// container prior to running the integration test suite.
// This packages manages the life cycle of the of this docker container
// it fires off the container, kill the container and remove its volume
// after the test suite is completed.

package dbtest

import (
	"bytes"
	"camlistore.org/pkg/netutil"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	mongoImage       = "mongo"
	postgresImage    = "postgres"
	PostgresUsername = "docker"
	PostgresPassword = "docker"
	IPAddressRegex   = "(.*IPAddress\"\\: \")(.*)(\",)"
	VolumeRegex      = "(volumes\\/)(.*)(\\/)"
)

// Container holding the docker container info
type Container struct {
	name     string
	id       string
	host     string
	port     int
	userName string
	password string
	volume   string
}

// Host container defaulting to local host
func (c Container) Host() string {
	return c.host
}

// Port on which the container is running on
func (c Container) Port() int {
	return c.port
}

// Username is the user name database
func (c Container) Username() string {
	return c.userName
}

// Password is the database password
func (c Container) Password() string {
	return c.password
}

// Kill the container
func kill(name string) error {
	return exec.Command("docker", "kill", name).Run()
}

// remove the container after the test is completed
func remove(name string) error {
	return exec.Command("docker", "rm", name).Run()
}

// delete the volume of the container
func deleteVolume(volume string) error {
	return exec.Command("docker", "volume", "rm", volume).Run()
}

// check all the conditions for running a docker container based on image.
func check(image string) {

	if !haveDocker() {
		log.Fatal("'docker' command not found")
	}

	ok, err := haveImage(image)
	if err != nil {
		log.Printf("error running docker to check for %s: %v", image, err)
	}

	if !ok {
		log.Printf("Pulling docker image %s ...", image)
		if err := Pull(image); err != nil {
			log.Printf("error pulling %s: %v", image, err)
		}
	}

	// check if teh container is running
	stopIfContainerIsRunning(fmt.Sprintf("%s_%s", image, "container"))
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

func stopIfContainerIsRunning(name string) {
	if isContainerRunning(name) {
		if err := kill(name); err != nil {
			panic(err)
		}
		if err := remove(name); err != nil {
			panic(err)
		}
	} else {
		remove(name)
	}
}

func isContainerRunning(name string) bool {
	if ip, _, _ := inspectContainer(name); ip != "" {
		return true
	}
	return false
}

// run the image
func run(args ...string) (string, error) {
	cmd := exec.Command("docker", append([]string{"run"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%v%v", stderr.String(), err)
		return "", err
	}
	if containerID := strings.TrimSpace(stdout.String()); containerID != "" {
		return containerID, nil
	}
	return "", errors.New("unexpected empty output from `docker run`")
}

// Run inspect container command and returns container details , ip  and volume
func inspectContainer(containerID string) (string, string, error) {
	cmd := exec.Command("docker", "inspect", containerID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%v%v", stderr.String(), err)
		return "", "", err
	}
	ip := regexp.MustCompile(IPAddressRegex).FindStringSubmatch(stdout.String())[2]
	volume := regexp.MustCompile(VolumeRegex).FindStringSubmatch(stdout.String())[2]
	return ip, volume, nil
}

func inspectLogs(id string, text string) bool {
	cmd := exec.Command("docker", "logs", id)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%v%v", stderr.String(), err)
		return false
	}
	val := stdout.String()
	return strings.Contains(val, text)
}

// Pull retrieves the docker image with 'docker pull'.
func Pull(image string) error {
	out, err := exec.Command("docker", "pull", image).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v: %s", err, out)
	}
	return err
}

// Destroy stop and remove the container
func (c Container) Destroy() {
	if err := kill(c.name); err != nil {
		log.Println(err)
		return
	}
	if err := remove(c.name); err != nil {
		log.Println(err)
	}

	if err := deleteVolume(c.volume); err != nil {
		log.Println(err)
	}
}

// lookup tries to reach
// before timeout the tcp address at this ip and given port.
func lookup(port int, host string, timeout time.Duration, id string) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	return netutil.AwaitReachable(addr, timeout)
}

// setupContainer sets up a container, using the start function to run the given image.
func setupContainer(image string, port int, timeout time.Duration,
	start func() (string, error)) Container {

	// run basic check on the presence of docker command, if the container is running or not.
	check(image)
	name := fmt.Sprintf("%s_%s", image, "container")

	// start the image
	containerID, err := start()
	c := Container{id: containerID, name: name, host: "127.0.0.1"}

	err = lookup(port, "127.0.0.1", timeout, containerID)
	if err != nil {
		c.Destroy()
		log.Printf("Container %v setup failed: %v", c, err)
	}
	return c
}

// StartMongoContainer
func StartMongoContainer() Container {
	name := fmt.Sprintf("%s_%s", mongoImage, "container")
	c := setupContainer(mongoImage, 27017, 10*time.Second, func() (string, error) {
		return run("-d", "-p", "27017:27017", "--name", name, mongoImage)
	})
	time.Sleep(time.Second * 5)

	_, volume, _ := inspectContainer(c.id)

	c.name = name
	c.port = 27017
	c.volume = volume
	return c
}

// StartPostgresContainerWithInitialisationScript start a postgres container with an initialisation script.
func StartPostgresContainerWithInitialisationScript(dbname, schema string) Container {
	file, err := filepath.Abs(schema)
	if err != nil {
		log.Fatalf("failed to load the schema file %s", schema)
	}
	name := fmt.Sprintf("%s_%s", postgresImage, "container")
	port := 5432
	return startPostgres(dbname, port, func() (s string, e error) {
		return run("-d", "--name", name, "-e", "POSTGRES_PASSWORD="+PostgresPassword,
			"-e", "POSTGRES_USER="+PostgresUsername, "-e", "POSTGRES_DB="+dbname,
			"-v", fmt.Sprintf("%s:/docker-entrypoint-initdb.d/initialise_db.sql", file),
			"-p", fmt.Sprintf("%d:%d", port, port), postgresImage)
	})
}

// StartPostgresContainer starts a postgres container
func StartPostgresContainer(dbname string) Container {
	name := fmt.Sprintf("%s_%s", postgresImage, "container")
	port := 5432
	return startPostgres(dbname, port, func() (s string, e error) {
		return run("-d", "--name", name, "-e", "POSTGRES_PASSWORD="+PostgresPassword,
			"-e", "POSTGRES_USER="+PostgresUsername, "-e", "POSTGRES_DB="+dbname,
			"-p", fmt.Sprintf("%d:%d", port, port), postgresImage)
	})
}

// starts postgres container
func startPostgres(dbname string, port int, r func() (string, error)) Container {
	name := fmt.Sprintf("%s_%s", postgresImage, "container")

	c := setupContainer(postgresImage, port, 100*time.Second, r)

	destroy := func(err error) {
		c.Destroy()
		log.Fatal(err)
	}
	connectTestDb := dbname + "_" + "test"
	rootdb, err := sql.Open("postgres",
		fmt.Sprintf("user=%s password=%s host=%s dbname=postgres sslmode=disable", PostgresUsername, PostgresPassword, c.host))
	if err != nil {
		destroy(fmt.Errorf("Could not open postgres rootdb: %v", err))
	}

	if _, err := sqlExecRetry(rootdb, "CREATE DATABASE "+dbname+"_"+"test"+" LC_COLLATE = 'C' TEMPLATE = template0", 50); err != nil {
		destroy(fmt.Errorf("Could not create database %v: %v", connectTestDb, err))
	}

	_, volume, _ := inspectContainer(c.id)
	c.name = name
	c.port = 5432
	c.userName = PostgresUsername
	c.password = PostgresPassword
	c.volume = volume
	return c
}

// sqlExecRetry to check the connection
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
