package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type config struct {
	port string
}

func main() {
	envCfg := envConfig()

	fmt.Println("hello world")
	log.Println("hello console")
	log.Printf("env conf: %v", envCfg)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
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
