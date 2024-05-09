package main

import (
	"bufio"
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
//go:embed web/static/assets/*
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
	language             language
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

type language string

func NewLang(maybeLang string) (language, error) {
	switch language(maybeLang) {
	case LANG_EN, LANG_DE:
		return language(maybeLang), nil
	default:
		return LANG_EN, fmt.Errorf("couldn't create new language from given value: '%s'", maybeLang)
	}
}

const (
	LANG_EN language = "en"
	LANG_DE language = "de"
)

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

func toWord(wo string) (word, error) {
	out := word{}

	length := 0
	for i, l := range wo {
		length++
		if length > len(out) {
			return word{}, fmt.Errorf("string does not match allowed word length: length=%d, expectedLength=%d", length, len(out))
		}

		out[i] = l
	}

	if length < len(out) {
		return word{}, fmt.Errorf("string is to short: length=%d, expectedLength=%d", length, len(out))
	}

	return out, nil
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
	Language              language
}

func (fd FormData) New() FormData {
	return FormData{
		Data:                  puzzle{},
		Errors:                make(map[string]string),
		JSCachePurgeTimestamp: time.Now().Unix(),
		Language:              LANG_EN,
	}
}

type wordDatabase struct {
	db map[language]map[word]bool
}

func (wdb *wordDatabase) Init(fs iofs.FS, filePathsByLanguage map[language]string) error {
	wdb.db = make(map[language]map[word]bool)

	for l, path := range filePathsByLanguage {
		wdb.db[l] = make(map[word]bool)

		f, err := fs.Open(path)
		if err != nil {
			return fmt.Errorf("wordDatabase init failed when opening file: %s", err)
		}
		defer f.Close()

		fInfo, err := f.Stat()
		if err != nil {
			return fmt.Errorf("wordDatabase init failed when obtaining stat: %s", err)
		}

		var allowedSize int64 = 2 * 1024 * 1024 // 2 MB
		if fInfo.Size() > allowedSize {
			return fmt.Errorf("wordDatabase init failed with forbidden file size: path='%s', size='%d'", path, fInfo.Size())
		}

		scanner := bufio.NewScanner(f)
		var line int = 0
		for scanner.Scan() {
			if line == 0 { // skip first metadata line
				line++
				continue
			}

			candidate := scanner.Text()
			word, err := toWord(candidate)
			if err != nil {
				return fmt.Errorf("wordDatabase init, couldn't parse line to word: line='%s', err=%s", candidate, err)
			}

			wdb.db[l][word.ToLower()] = true

			line++
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("wordDatabase init failed scanning file with: path='%s', err=%s", path, err)
		}
	}

	return nil
}

func (wdb wordDatabase) Exists(l language, w word) bool {
	db, ok := wdb.db[l]
	if !ok {
		return false
	}

	_, ok = db[w.ToLower()]
	return ok
}

func (wdb wordDatabase) RandomPick(l language) (word, error) {
	db, ok := wdb.db[l]
	if !ok {
		return word{}, fmt.Errorf("RandomPick failed with unknown language: '%s'", l)
	}

	randsource := rand.NewSource(time.Now().UnixNano())
	randgenerator := rand.New(randsource)
	rolledLine := randgenerator.Intn(len(db))

	currentLine := 0
	for w := range db {
		if currentLine == rolledLine {
			return w, nil
		}

		currentLine++
	}

	return word{}, fmt.Errorf("RandomPick could not find random line aka this should never happen ^^")
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

func filePathsByLang() map[language]string {
	return map[language]string{
		LANG_EN: "configs/en-en.words.v2.txt",
		LANG_DE: "configs/de-de.words.v2.txt",
	}
}

func main() {
	log.Println("staring server...")

	envCfg := envConfig()
	sessions := sessions{}

	wordDb := wordDatabase{}
	err := wordDb.Init(fs, filePathsByLang())
	if err != nil {
		log.Fatalf("init wordDatabase failed: %s", err)
	}

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
		sess := handleSession(w, req, &sessions, wordDb)

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
		fData.Language = sess.language

		err := t.ExecuteTemplate(w, "index.html.tmpl", fData)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	})

	mux.HandleFunc("POST /lettr", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions, wordDb)

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

		wo, err = parseForm(wo, r.PostForm, s.activeSolutionWord, s.language, wordDb)
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
		fData.Language = s.language

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/lettr' route: %s", err)
		}
	})

	mux.HandleFunc("POST /new", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions, wordDb)

		// handle lang switch
		l := s.language
		maybeLang := r.FormValue("lang")
		if maybeLang != "" {
			l, _ = NewLang(maybeLang)
			s.language = l

			data := struct {
				Language language
			}{
				Language: l,
			}

			err := t.ExecuteTemplate(w, "oob-lang-switch", data)
			if err != nil {
				log.Printf("error t.ExecuteTemplate '/new' route: %s", err)
			}
		}

		p := puzzle{}

		s.lastEvaluatedAttempt = p
		s.activeSolutionWord = generateSession(l, wordDb).activeSolutionWord
		sessions.updateOrSet(s)

		p.Debug = s.activeSolutionWord.String()

		fData := FormData{}.New()
		fData.Data = p
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()
		fData.Language = s.language

		// w.Header().Add("HX-Refresh", "true")
		err := t.ExecuteTemplate(w, "lettr-form", fData)
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

func handleSession(w http.ResponseWriter, req *http.Request, sessions *sessions, wdb wordDatabase) session {
	var err error
	var sess session

	cookie, err := req.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		return newSession(w, sessions, wdb)
	}

	if cookie == nil {
		return newSession(w, sessions, wdb)
	}

	sid := cookie.Value
	i := slices.IndexFunc(*sessions, func(s session) bool {
		return s.id == sid
	})
	if i == -1 {
		return newSession(w, sessions, wdb)
	}

	sess = (*sessions)[i]

	c := constructCookie(sess)
	http.SetCookie(w, &c)

	sess.expiresAt = generateSessionLifetime()
	(*sessions)[i] = sess

	return sess
}

func newSession(w http.ResponseWriter, sessions *sessions, wdb wordDatabase) session {
	sess := generateSession(LANG_EN, wdb)
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

func generateSession(lang language, wdb wordDatabase) session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := generateSessionLifetime()
	activeWord, err := wdb.RandomPick(lang)
	if err != nil {
		log.Printf("pick random word failed: %s", err)

		activeWord = word{'R', 'O', 'A', 'T', 'E'}
	}

	return session{id, expiresAt, SESSION_MAX_AGE_IN_SECONDS, lang, activeWord, puzzle{}}
}

func generateSessionLifetime() time.Time {
	return time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
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

func parseForm(p puzzle, form url.Values, solutionWord word, l language, wdb wordDatabase) (puzzle, error) {
	for ri := range p.Guesses {
		maybeGuessedWord, ok := form[fmt.Sprintf("r%d", ri)]
		if !ok {
			continue
		}

		guessedWord, err := sliceToWord(maybeGuessedWord)
		if err != nil {
			return p, fmt.Errorf("parseForm could not create guessedWord from form input: %s", err.Error())
		}

		if !wdb.Exists(l, guessedWord) {
			return p, ErrNotInWordList
		}

		wg := evaluateGuessedWord(guessedWord, solutionWord)

		p.Guesses[ri] = wg
	}

	return p, nil
}

func sliceToWord(maybeGuessedWord []string) (word, error) {
	w := word{}

	if len(maybeGuessedWord) != len(w) {
		return word{}, fmt.Errorf("sliceToWord: provided slice does not match word length")
	}

	for i, l := range maybeGuessedWord {
		w[i], _ = utf8.DecodeRuneInString(strings.ToLower(l))
		if w[i] == 65533 {
			w[i] = 0
		}
	}

	return w, nil
}

func evaluateGuessedWord(guessedWord word, solutionWord word) wordGuess {
	solutionWord = solutionWord.ToLower()
	guessedLetterCountMap := make(map[rune]int)

	resultWordGuess := wordGuess{}

	for i, gr := range guessedWord {
		exact := solutionWord[i] == gr

		if exact {
			guessedLetterCountMap[gr]++
		}

		resultWordGuess[i] = letterGuess{gr, letterHitOrMiss{exact, exact}}
	}

	for i, gr := range guessedWord {
		some := solutionWord.contains(gr)

		if !resultWordGuess[i].HitOrMiss.Some || some {
			guessedLetterCountMap[gr]++
		}

		s := some && (guessedLetterCountMap[gr] <= solutionWord.count(gr))
		resultWordGuess[i].HitOrMiss.Some = s || resultWordGuess[i].HitOrMiss.Exact
	}

	return resultWordGuess
}
