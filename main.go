package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"math/rand"
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
//go:embed configs/*.txt
var fs embed.FS

type env struct {
	port string
}

func (e env) String() string {
	s := fmt.Sprintf("port: %s\n", e.port)
	// s = s + fmt.Sprintf("foo: %s\n", e.port)
	return s
}

type counterState struct {
	mu    sync.Mutex
	count int
}

type session struct {
	// todo: think about using mutex or channel for rw session
	id                   string
	expiresAt            time.Time
	maxAgeSeconds        int
	activeSolutionWord   wordleWord
	lastEvaluatedAttempt wordle
}

type sessions []session

func (ss sessions) String() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + " " + s.expiresAt.String() + "\n"
	}

	return out
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

type wordle struct {
	Debug   string
	Guesses [6]wordGuess
}

type wordGuess [5]letterGuess

type letterGuess struct {
	Letter    rune
	HitOrMiss letterHitOrMiss
}

type letterHitOrMiss struct {
	Some  bool
	Exact bool
}

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	log.Println("staring server...")
	log.Printf("env conf:\n%s", envCfg)

	t := template.Must(template.ParseFS(fs, "templates/index.html.tmpl", "templates/wordle-form.html.tmpl"))

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		sess := handleSession(w, req, &sessions)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		//io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		log.Printf("sessions:\n%s", sessions)

		wo := sess.lastEvaluatedAttempt
		wo.Debug = sess.activeSolutionWord.String()

		err := t.Execute(w, wo)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	})

	mux.HandleFunc("POST /wordle", func(w http.ResponseWriter, r *http.Request) {
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

		wo := wordle{Debug: s.activeSolutionWord.String()}
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
	mux.HandleFunc("POST /counter", func(w http.ResponseWriter, req *http.Request) {
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

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), mux))
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

	sess.expiresAt = generateSessionLifetime()
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
	expiresAt := generateSessionLifetime()
	activeWord, err := pickRandomWord()
	if err != nil {
		log.Printf("pick random word failed: %s", err)

		activeWord = wordleWord{'R', 'O', 'A', 'T', 'E'}
	}

	return session{id, expiresAt, SESSION_MAX_AGE_IN_SECONDS, activeWord, wordle{}}
}

func generateSessionLifetime() time.Time {
	return time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
}

func pickRandomWord() (wordleWord, error) {
	f, err := fs.Open("configs/en-en.words.v1.test.txt")
	if err != nil {
		return wordleWord{}, fmt.Errorf("pick random word failed when opening file: %s", err)
	}
	defer f.Close()

	lineCount, err := lineCounter(f)
	if err != nil {
		return wordleWord{}, fmt.Errorf("pick random word failed when reading line count: %s", err)
	}

	randsource := rand.NewSource(time.Now().UnixNano())
	randgenerator := rand.New(randsource)
	rolledLine := randgenerator.Intn(lineCount-1) + 1

	log.Printf("linecount: %d, roll: %d", lineCount, rolledLine)

	fc, err := fs.Open("configs/en-en.words.v1.test.txt")
	if err != nil {
		return wordleWord{}, fmt.Errorf("pick random word failed when opening file: %s", err)
	}
	defer fc.Close()
	scanner := bufio.NewScanner(fc)
	var line int
	var rollWord string
	for scanner.Scan() {
		if line == rolledLine {
			log.Println("hit if")
			log.Println(scanner.Text())
			rollWord = scanner.Text()
			log.Printf("rollWord: '%s'", rollWord)
			break
		}
		line++
	}
	if err := scanner.Err(); err != nil {
		return wordleWord{}, fmt.Errorf("pick random word failed with: %s", err)
	}

	rollWordRuneSlice := []rune(rollWord)

	pickedWord := wordleWord{}             //initialized an empty array
	copy(pickedWord[:], rollWordRuneSlice) //copy the elements of sl

	log.Printf("pickedWord: '%s'", pickedWord)

	return pickedWord, nil
}

func lineCounter(r io.Reader) (int, error) {
	var count int
	const lineBreak = '\n'

	buf := make([]byte, bufio.MaxScanTokenSize)

	for {
		bufferSize, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		var buffPosition int
		for {
			i := bytes.IndexByte(buf[buffPosition:], lineBreak)
			if i == -1 || bufferSize == buffPosition {
				break
			}
			buffPosition += i + 1
			count++
		}
		if err == io.EOF {
			break
		}
	}

	return count, nil
}

func parseForm(wo wordle, form url.Values, solutionWord wordleWord) wordle {

	// log.Println("")
	// log.Printf("parseForm() var solutionWord:'%v'\n", solutionWord)
	// log.Println("")

	for ri := range wo.Guesses {
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

		s := some && (guessedLetterCountMap[gr] <= solutionWord.count(gr))
		resultWordGuess[i] = letterGuess{gr, letterHitOrMiss{s, exact}}
	}

	return resultWordGuess
}
