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

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	log.Println("staring server...")
	log.Printf("env conf:\n%s", envCfg)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		session := initSession(w, req, sessions)

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

func initSession(w http.ResponseWriter, req *http.Request, sessions []session) session {
	var err error
	var sess session

	cookie, err := req.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		sess := generateSession()
		sessions = append(sessions, sess)
		c := constructCookie(sess)
		http.SetCookie(w, &c)

		return sess
	}

	if cookie == nil {
		sess := generateSession()
		sessions = append(sessions, sess)
		c := constructCookie(sess)
		http.SetCookie(w, &c)

		return sess
	}

	sid := cookie.Value
	i := slices.IndexFunc(sessions, func(s session) bool {
		return s.id == sid
	})
	if i == -1 {
		sess := generateSession()
		sessions = append(sessions, sess)
		c := constructCookie(sess)
		http.SetCookie(w, &c)

		return sess
	}

	sess = sessions[i]

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
	}
}

func generateSession() session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := time.Now().Add(120 * time.Second) // todo: 24 hour, sec now only for testing
	return session{id, expiresAt}
}
