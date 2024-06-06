package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func main() {
	log.Println("start...")

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Name:           "mariadb",
		FromDockerfile: FromDockerfile(),
		ExposedPorts:   []string{"3306/tcp"},
		Env: map[string]string{
			// "MARIADB_USER": "root", // ?? is MARIADB_USER same as `root` user or an additional user, which is than corresponding with MARIADB_PASSWORD?
			"MARIADB_ROOT_PASSWORD": "example",
		},
		WaitingFor: wait.ForAll(
			wait.ForExposedPort(),
			wait.ForListeningPort("3306/tcp"),
			wait.ForLog("mariadbd: ready for connections"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("couldn't start container: %s", err)
	}

	cmd := []string{"bash", "-c", "ls -al"}
	exitCode, cmdLog, err := execContaincerCmd(ctx, container, cmd)
	if err != nil {
		log.Fatalf("failed executing command in container: exitCode='%d', err='%s', cmdLog:\n%s", exitCode, err, cmdLog)
	}

	log.Println("cmd logs:", cmdLog)
}

func FromDockerfile() testcontainers.FromDockerfile {
	df := `
		FROM mariadb:11.3.2

		RUN set -e; \
			test "1" = "$(cat /etc/os-release | grep -i 'Name="Ubuntu"' | wc -l)"; \
			test "1" = "$(cat /etc/os-release | grep 'VERSION="22.04.4 LTS (Jammy Jellyfish)"' | wc -l)";

		#RUN set -e; \
		#	test "$(cat /etc/os-release | grep 'NAME="Ubuntu"' | wc -l)" = "1"; \
		#	test "$(cat /etc/os-release | grep 'VERSION="22.04.4 LTS (Jammy Jellyfish)"' | wc -l)" = "1"; \
		#	;

		RUN set -e; \
			apt-get update; \
			apt-get install -y \
				curl \
			;
		
		RUN mkdir -p /scripts
		# COPY ./corporas_export.sh /scripts/corporas_export.sh
		# 
		# COPY ./corpora_download.sh /scripts/corpora_download.sh
		# RUN set -e; mkdir -p /datasets; /scripts/corpora_download.sh /datasets

		ENV ENV GO_VERSION=1.22.4
		# https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
		# https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz
		# RUN  rm -rf /usr/local/go && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
		# RUN export PATH=$PATH:/usr/local/go/bin
		# RUN  go version
	`

	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	// ... add some files
	// log.Printf("df reflect size: %v", reflect.TypeOf(df).Size())
	// log.Printf("df len: %v", len(df))
	// log.Printf("df what works: %v", int64(reflect.TypeOf(df).Size())+311)
	hdr := tar.Header{
		Name: "Dockerfile",
		Size: int64(len(df)),
	}
	err := tarWriter.WriteHeader(&hdr)
	if err != nil {
		log.Fatalf("couldn't write tar header: %s", err)
	}

	_, err = tarWriter.Write([]byte(df))
	if err != nil {
		log.Fatalf("couldn't write tar file content: %s", err)
	}

	if err := tarWriter.Close(); err != nil {
		// do something with err
		log.Fatalf("couldn't close tar: %s", err)
	}

	reader := bytes.NewReader(buf.Bytes())

	fromDockerfile := testcontainers.FromDockerfile{
		ContextArchive: reader,
		Dockerfile:     "Dockerfile",
		PrintBuildLog:  true,
	}

	return fromDockerfile
}

func execContaincerCmd(ctx context.Context, c testcontainers.Container, cmd []string) (exitCode int, cmdLog string, err error) {
	exitCode, cmdExecLogReader, err := c.Exec(ctx, cmd)
	if err != nil {
		return -1, "", fmt.Errorf("couldn't exec cmd '%s' in container: %s", strings.Join(cmd, " "), err)
	}

	execCmdLog, err := io.ReadAll(cmdExecLogReader)
	if err != nil {
		return -1, "", fmt.Errorf("couldn't read exec cmd log: %s", err)
	}
	cmdLog = string(execCmdLog)

	if exitCode != 0 {
		return -1, "", fmt.Errorf("exec cmd in container failed with non zero exit code: exitCode=%d \n  logs:\n\n%s", exitCode, cmdLog)
	}

	return exitCode, cmdLog, nil
}
