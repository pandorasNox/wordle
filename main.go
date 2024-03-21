package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"sync"

	//"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const SESSION_COOKIE_NAME = "session"
const SESSION_MAX_AGE_IN_SECONDS = 120

//go:embed templates/*.html.tmpl
var fs embed.FS

type env struct {
	port string
}

func (e env) String() string {
	s := fmt.Sprintf("port: %s\n", e.port)
	// s = s + fmt.Sprintf("foo: %s\n", e.port)
	return s
}

type session struct {
	id                 string
	expiresAt          time.Time
	maxAgeSeconds      int
	activeSolutionWord wordleWord
}

type wordleWord [5]rune

func (w wordleWord) String() string {
	out := ""
	for _, v := range w {
		out += string(v)
	}

	return out
}

func (w wordleWord) contains(letter rune) bool {
	found := false
	for _, v := range w {
		if v == letter {
			found = true
			break
		}
	}

	return found
}

func (w wordleWord) count(letter rune) int {
	count := 0
	for _, v := range w {
		if v == letter {
			count++
		}
	}

	return count
}

type sessions []session

func (ss sessions) String() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + " " + s.expiresAt.String() + "\n"
	}

	return out
}

type counterState struct {
	mu    sync.Mutex
	count int
}

type wordle struct {
	Bla     string
	Guesses [6]wordGuess
}

type letterGuess struct {
	Letter    rune
	HitOrMiss letterHitOrMiss
}

type letterHitOrMiss struct {
	Some  bool
	Exact bool
}

type wordGuess [5]letterGuess

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	log.Println("staring server...")
	log.Printf("env conf:\n%s", envCfg)

	t := template.Must(template.ParseFS(fs, "templates/index.html.tmpl", "templates/wordle-form.html.tmpl"))

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		handleSession(w, req, &sessions)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		//io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		log.Printf("sessions:\n%s", sessions)

		err := t.Execute(w, wordle{Bla: "test"})
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	})

	http.HandleFunc("/wordle", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions)
		_ = s

		// b, err := io.ReadAll(r.Body)
		// if err != nil {
		// 	// log.Fatalln(err)
		// 	log.Printf("error: %s", err)
		// }
		// log.Printf("word: %s\nbody:\n%s", s.activeWord, b)

		err := r.ParseForm()
		if err != nil {
			// log.Fatalln(err)
			log.Printf("error: %s", err)
		}

		wo := wordle{Bla: "test"}
		wo = parseForm(wo, r.PostForm, s.activeSolutionWord)

		// log.Println("")
		// log.Printf("rout '/wordle' - wordle sfter parseForm:'%v'\n", wo)
		// log.Println("")

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		//io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		//log.Printf("sessions:\n%s", sessions)
		//log.Println(wo)

		err = t.ExecuteTemplate(w, "wordle-form", wo)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/wordle' route: %s", err)
		}
	})

	counter := counterState{count: 0}
	http.HandleFunc("/counter", func(w http.ResponseWriter, req *http.Request) {
		// handleSession(w, req, &sessions)
		counter.mu.Lock()
		counter.count++
		defer counter.mu.Unlock()

		b, err := io.ReadAll(req.Body)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Method: %s\nbody:\n%s", req.Method, b)

		io.WriteString(w, fmt.Sprintf("<span>%d</span>", counter.count))

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

	sess.expiresAt = generateSession().expiresAt
	(*sessions)[i] = sess

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
		MaxAge:   s.maxAgeSeconds,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func generateSession() session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
	//activeWord := "ROATE"
	activeWord := wordleWord{'R', 'O', 'A', 'T', 'E'}
	return session{id, expiresAt, SESSION_MAX_AGE_IN_SECONDS, activeWord}
}

func parseForm(wo wordle, form url.Values, solutionWord wordleWord) wordle {

	// log.Println("")
	// log.Printf("parseForm() var solutionWord:'%v'\n", solutionWord)
	// log.Println("")

	for ri, _ := range wo.Guesses {
		// log.Println("")
		// log.Printf("parseForm() var form:%v\n", form)
		// log.Println("")

		guessedWord, ok := form[fmt.Sprintf("r%d", ri)]
		if !ok {
			// log.Println("")
			// log.Printf("continue map: r%d\n", ri)
			// log.Println("")
			continue
		}

		// log.Println("")
		// log.Println("parseForm() var guessedWord:", guessedWord)
		// log.Println("")

		wg := evaluateGuessedWord(guessedWord, solutionWord)

		wo.Guesses[ri] = wg
	}

	return wo
}

func evaluateGuessedWord(guessedWord []string, solutionWord wordleWord) wordGuess {
	guessedLetterCountMap := make(map[rune]int)

	resultWordGuess := wordGuess{}

	for i := range guessedWord {
		gr, _ := utf8.DecodeRuneInString(guessedWord[i])

		some := solutionWord.contains(gr)
		exact := solutionWord[i] == gr

		if some {
			guessedLetterCountMap[gr]++
		}

		s := (guessedLetterCountMap[gr] <= solutionWord.count(gr))
		resultWordGuess[i] = letterGuess{gr, letterHitOrMiss{s, exact}}
	}

	return resultWordGuess
}
