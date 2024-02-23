package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type config struct {
	port string
}

type session struct {
	id        string
	expiresAt time.Time
}

type sessions []session

func (ss sessions) toString() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + "\n"
	}

	return out
}

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	fmt.Println("hello world")
	log.Println("hello console")
	log.Printf("env conf: %v", envCfg)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		session := generateSession()
		sessions = append(sessions, session)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions.toString()))
		io.WriteString(w, "Hello, world!\n")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), nil))
}

func envConfig() config {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not provided")
	}

	return config{port}
}

func generateSession() session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := time.Now().Add(120 * time.Second) // todo: 24 hour, sec now only for testing
	return session{id, expiresAt}
}
