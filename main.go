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

const SESSION_COOKIE_NAME = "session"

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	fmt.Println("hello world")
	log.Println("hello console")
	log.Printf("env conf: %v", envCfg)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		session := initSession(w, req, sessions)
		
		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions.toString()))
		io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
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
	if cookie != nil {
		sid := cookie.Value
		i := slices.IndexFunc(sessions, func (s session) bool{
			return s.id == sid 
		})

		if i != -1 {
			sess = sessions[i]
		} else {
			err = http.ErrNoCookie
		}
	}
	
	if err != nil {
		sess := generateSession()
		sessions = append(sessions, sess)			
	}

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
