package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/google/uuid"
)

type env struct {
	port string
}

func (e env) String() string {
	s := fmt.Sprintf("port: %s\n", e.port)
	// s = s + fmt.Sprintf("foo: %s\n", e.port)
	return s
}

type session struct {
	id        string
	expiresAt time.Time
}

type sessions []session

func (ss sessions) String() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + "\n"
	}

	return out
}

const SESSION_COOKIE_NAME = "session"
const SESSION_MAX_AGE_IN_SECONDS = 120

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	log.Println("staring server...")
	log.Printf("env conf:\n%s", envCfg)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		session := handleSession(w, req, &sessions)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		log.Printf("sessions:\n%s", sessions)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), nil))
}

func envConfig() env {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not provided")
	}

	return env{port}
}

func handleSession(w http.ResponseWriter, req *http.Request, sessions *sessions) session {
	var err error
	var sess session

	cookie, err := req.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		return newSession(w, sessions)
	}

	if cookie == nil {
		return newSession(w, sessions)
	}

	sid := cookie.Value
	i := slices.IndexFunc(*sessions, func(s session) bool {
		return s.id == sid
	})
	if i == -1 {
		return newSession(w, sessions)
	}

	sess = (*sessions)[i]

	c := constructCookie(sess)
	http.SetCookie(w, &c)

	return sess
}

func newSession(w http.ResponseWriter, sessions *sessions) session {
	sess := generateSession()
	*sessions = append(*sessions, sess)
	c := constructCookie(sess)
	http.SetCookie(w, &c)

	return sess
}

func constructCookie(s session) http.Cookie {
	return http.Cookie{
		Name:     SESSION_COOKIE_NAME,
		Value:    s.id,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		// Expires:  time.Now(),
	}
}

func generateSession() session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
	return session{id, expiresAt}
}
