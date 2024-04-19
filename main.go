package main

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"strings"
	"sync"
	"unicode"

	iofs "io/fs"
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

//go:embed configs/*.txt
//go:embed templates/*.html.tmpl
//go:embed web/static/generated/*.js
//go:embed web/static/generated/*.css
var fs embed.FS

var ErrNotInWordList = errors.New("not in wordlist")

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
	activeSolutionWord   word
	lastEvaluatedAttempt puzzle
}

type sessions []session

func (ss sessions) String() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + " " + s.expiresAt.String() + "\n"
	}

	return out
}

func (ss *sessions) updateOrSet(sess session) {
	index := slices.IndexFunc((*ss), func(s session) bool {
		return s.id == sess.id
	})
	if index == -1 {
		*ss = append(*ss, sess)
		return
	}

	(*ss)[index] = sess
}

type word [5]rune

func (w word) String() string {
	out := ""
	for _, v := range w {
		out += string(v)
	}

	return out
}

func (w word) contains(letter rune) bool {
	found := false
	for _, v := range w {
		if v == letter {
			found = true
			break
		}
	}

	return found
}

func (w word) count(letter rune) int {
	count := 0
	for _, v := range w {
		if v == letter {
			count++
		}
	}

	return count
}

func (w word) ToLower() word {
	for i, v := range w {
		w[i] = unicode.ToLower(v)
	}

	return w
}

type puzzle struct {
	Debug   string
	Guesses [6]wordGuess
}

func (p puzzle) activeRow() uint8 {
	for i, wg := range p.Guesses {
		if !wg.isFilled() {
			return uint8(i)
		}
	}

	return uint8(len(p.Guesses))
}

func (p puzzle) isSolved() bool {
	if p.activeRow() > 0 {
		return p.Guesses[p.activeRow()-1].isSolved()
	}

	return false
}

func (p puzzle) isLoose() bool {
	for _, wg := range p.Guesses {
		if !wg.isFilled() || wg.isSolved() {
			return false
		}
	}

	return true
}

type wordGuess [5]letterGuess

func (wg wordGuess) isFilled() bool {
	for _, l := range wg {
		if l.Letter == 0 || l.Letter == 65533 {
			return false
		}
	}

	return true
}

func (wg wordGuess) isSolved() bool {
	for _, lg := range wg {
		if lg.HitOrMiss.Exact == false {
			return false
		}
	}

	return true
}

type letterGuess struct {
	Letter    rune
	HitOrMiss letterHitOrMiss
}

type letterHitOrMiss struct {
	Some  bool
	Exact bool
}

type FormData struct {
	Data                  puzzle
	Errors                map[string]string
	IsSolved              bool
	IsLoose               bool
	JSCachePurgeTimestamp int64
}

func (fd FormData) New() FormData {
	return FormData{
		Data:                  puzzle{},
		Errors:                make(map[string]string),
		JSCachePurgeTimestamp: time.Now().Unix(),
	}
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func getAllFilenames(efs iofs.FS) (files []string, err error) {
	if err := iofs.WalkDir(efs, ".", func(path string, d iofs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	log.Println("staring server...")
	log.Printf("env conf:\n%s", envCfg)

	t := template.Must(template.ParseFS(fs, "templates/index.html.tmpl", "templates/lettr-form.html.tmpl"))

	mux := http.NewServeMux()

	staticFS, err := iofs.Sub(fs, "web/static")
	if err != nil {
		log.Fatalf("subtree for 'static' dir of embed fs failed: %s", err)
	}

	mux.Handle(
		"GET /static/",
		http.StripPrefix("/static", http.FileServer(http.FS(staticFS))),
	)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		sess := handleSession(w, req, &sessions)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		//io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		log.Printf("sessions:\n%s", sessions)

		wo := sess.lastEvaluatedAttempt
		// log.Printf("debug '/' route - sess.lastEvaluatedAttempt:\n %v\n", wo)
		wo.Debug = sess.activeSolutionWord.String()
		sessions.updateOrSet(sess)

		fData := FormData{}.New()
		fData.Data = wo
		fData.IsSolved = wo.isSolved()
		fData.IsLoose = wo.isLoose()

		err := t.Execute(w, fData)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	})

	mux.HandleFunc("POST /lettr", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions)

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

		wo := s.lastEvaluatedAttempt
		wo.Debug = s.activeSolutionWord.String()

		// log.Printf("debug '/lettr' route - row compare: activeRow='%d' / formRowCount='%d' \n", s.lastEvaluatedAttempt.activeRow(), countFilledFormRows(r.PostForm))
		if s.lastEvaluatedAttempt.activeRow() != countFilledFormRows(r.PostForm)-1 {
			w.Header().Add("HX-Retarget", "#any-errors")
			w.WriteHeader(422)
			w.Write([]byte("faked rows"))
			return
		}

		wo, err = parseForm(wo, r.PostForm, s.activeSolutionWord)
		if err == ErrNotInWordList {
			w.Header().Add("HX-Retarget", "#any-errors")
			w.WriteHeader(422)
			w.Write([]byte("word not in word list"))
			return
		}

		s.lastEvaluatedAttempt = wo
		sessions.updateOrSet(s)

		fData := FormData{}.New()
		fData.Data = wo
		fData.IsSolved = wo.isSolved()
		fData.IsLoose = wo.isLoose()

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/lettr' route: %s", err)
		}
	})

	mux.HandleFunc("POST /new", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions)

		p := puzzle{}

		s.lastEvaluatedAttempt = p
		s.activeSolutionWord = generateSession().activeSolutionWord
		sessions.updateOrSet(s)

		p.Debug = s.activeSolutionWord.String()

		fData := FormData{}.New()
		fData.Data = p
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/new' route: %s", err)
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

		activeWord = word{'R', 'O', 'A', 'T', 'E'}
	}

	return session{id, expiresAt, SESSION_MAX_AGE_IN_SECONDS, activeWord, puzzle{}}
}

func generateSessionLifetime() time.Time {
	return time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
}

func pickRandomWord() (word, error) {
	filePath := "configs/en-en.words.v2.txt"

	f, err := fs.Open(filePath)
	if err != nil {
		return word{}, fmt.Errorf("pick random word failed when opening file: %s", err)
	}
	defer f.Close()

	lineCount, err := lineCounter(f)
	if err != nil {
		return word{}, fmt.Errorf("pick random word failed when reading line count: %s", err)
	}

	randsource := rand.NewSource(time.Now().UnixNano())
	randgenerator := rand.New(randsource)
	rolledLine := randgenerator.Intn(lineCount-1) + 1

	log.Printf("linecount: %d, roll: %d", lineCount, rolledLine)

	fc, err := fs.Open(filePath)
	if err != nil {
		return word{}, fmt.Errorf("pick random word failed when opening file: %s", err)
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
		return word{}, fmt.Errorf("pick random word failed with: %s", err)
	}

	rollWordRuneSlice := []rune(rollWord)

	pickedWord := word{}                   //initialized an empty array
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

func countFilledFormRows(postPuzzleForm url.Values) uint8 {
	isfilled := func(row []string) bool {
		emptyButWithLen := make([]string, len(row)) // we need empty slice but with right elem length
		return !(slices.Compare(row, emptyButWithLen) == 0)
	}

	var count uint8 = 0
	l := len(postPuzzleForm)
	for i := 0; i < l; i++ {
		guessedWord, ok := postPuzzleForm[fmt.Sprintf("r%d", i)]
		if ok && isfilled(guessedWord) {
			count++
		}
	}

	return count
}

func parseForm(p puzzle, form url.Values, solutionWord word) (puzzle, error) {

	// log.Println("")
	// log.Printf("parseForm() var solutionWord:'%v'\n", solutionWord)
	// log.Println("")

	for ri := range p.Guesses {
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

		if slices.Compare(Map(guessedWord, strings.ToLower), []string{"x", "x", "x", "x", "x"}) == 0 {
			return p, ErrNotInWordList
		}

		wg := evaluateGuessedWord(guessedWord, solutionWord)

		p.Guesses[ri] = wg
	}

	return p, nil
}

func evaluateGuessedWord(guessedWord []string, solutionWord word) wordGuess {
	solutionWord = solutionWord.ToLower()
	guessedLetterCountMap := make(map[rune]int)

	resultWordGuess := wordGuess{}

	for i := range guessedWord {
		gr, _ := utf8.DecodeRuneInString(strings.ToLower(guessedWord[i]))

		exact := solutionWord[i] == gr

		if exact {
			guessedLetterCountMap[gr]++
		}

		resultWordGuess[i] = letterGuess{gr, letterHitOrMiss{exact, exact}}
	}

	for i := range guessedWord {
		gr, _ := utf8.DecodeRuneInString(strings.ToLower(guessedWord[i]))

		some := solutionWord.contains(gr)

		if !resultWordGuess[i].HitOrMiss.Some || some {
			guessedLetterCountMap[gr]++
		}

		s := some && (guessedLetterCountMap[gr] <= solutionWord.count(gr))
		resultWordGuess[i].HitOrMiss.Some = s || resultWordGuess[i].HitOrMiss.Exact
	}

	return resultWordGuess
}
