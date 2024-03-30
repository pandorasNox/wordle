package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/uuid"
)

func Test_constructCookie(t *testing.T) {
	fixedUuid := "9566c74d-1003-4c4d-bbbb-0407d1e2c649"
	expireDate := time.Date(2024, 02, 27, 0, 0, 0, 0, time.Now().Location())

	type args struct {
		s session
	}
	tests := []struct {
		name string
		args args
		want http.Cookie
	}{
		// add test cases here
		{
			"test_name",
			args{session{fixedUuid, expireDate, SESSION_MAX_AGE_IN_SECONDS, wordleWord{}, wordle{}}},
			http.Cookie{
				Name:     SESSION_COOKIE_NAME,
				Value:    fixedUuid,
				Path:     "/",
				MaxAge:   SESSION_MAX_AGE_IN_SECONDS,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructCookie(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleSession(t *testing.T) {
	type args struct {
		w        http.ResponseWriter
		req      *http.Request
		sessions *sessions
	}

	// monkey patch time.Now
	patchFnTime := func() time.Time {
		return time.Unix(1615256178, 0)
	}
	patchesTime := gomonkey.ApplyFunc(time.Now, patchFnTime)
	defer patchesTime.Reset()
	// monkey patch uuid.NewString
	patches := gomonkey.ApplyFuncReturn(uuid.NewString, "12345678-abcd-1234-abcd-ab1234567890")
	defer patches.Reset()

	patchFnPickRandomWord := func() (wordleWord, error) {
		// return wordleWord{'P', 'A', 'T', 'C', 'H'}, nil
		return wordleWord{'R', 'O', 'A', 'T', 'E'}, nil
	}
	patchesPickRandomWord := gomonkey.ApplyFunc(pickRandomWord, patchFnPickRandomWord)
	defer patchesPickRandomWord.Reset()

	// recorder := httptest.NewRecorder()
	// sess := sessions{}

	tests := []struct {
		name string
		args args
		want session
	}{
		// add test cases here
		{
			"test handleSession is generating new session if no cookie is set",
			args{
				httptest.NewRecorder(),
				httptest.NewRequest("get", "/", strings.NewReader("Hello, Reader!")),
				&sessions{},
			},
			session{
				id:                 "12345678-abcd-1234-abcd-ab1234567890",
				expiresAt:          time.Unix(1615256178, 0).Add(SESSION_MAX_AGE_IN_SECONDS * time.Second),
				maxAgeSeconds:      120,
				activeSolutionWord: wordleWord{'R', 'O', 'A', 'T', 'E'},
			},
		},
		// {
		// 	// todo // check out https://gist.github.com/jonnyreeves/17f91155a0d4a5d296d6 for inspiration
		// 	"test got cookie but no session corresponding session on server",
		// 	args{},
		// 	session{
		// 		id:            "12345678-abcd-1234-abcd-ab1234567890",
		// 		expiresAt:     time.Unix(1615256178, 0).Add(SESSION_MAX_AGE_IN_SECONDS * time.Second),
		// 		maxAgeSeconds: 120,
		// 		activeWord:    wordleWord{'R','O','A','T','E'},
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleSession(tt.args.w, tt.args.req, tt.args.sessions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleSession() = %v, want %v", got, tt.want)
			}
		})
	}

	// fmt.Println("")
	// fmt.Println("foooooooooooooooo")
	// fmt.Println("")

	// t.Run("test", func(t *testing.T) {
	// 	// t.Errorf("fail %v", session{})
	// 	t.Errorf("fail %v", handleSession(httptest.NewRecorder(), httptest.NewRequest("get", "/", strings.NewReader("Hello, Reader!")), &sessions{}))
	// })
}

func Test_parseForm(t *testing.T) {
	type args struct {
		wo           wordle
		form         url.Values
		solutionWord wordleWord
	}
	tests := []struct {
		name string
		args args
		want wordle
	}{
		// TODO: Add test cases.
		{
			name: "no hits, neither same or exact",
			// args: args{wordle{}, url.Values{}, wordleWord{'M', 'I', 'S', 'S', 'S'}},
			args: args{wordle{}, url.Values{"r0": []string{}}, wordleWord{'M', 'I', 'S', 'S', 'S'}},
			want: wordle{},
		},
		{
			name: "full exact match",
			args: args{
				wordle{},
				url.Values{"r0": []string{"M", "A", "T", "C", "H"}},
				wordleWord{'M', 'A', 'T', 'C', 'H'},
			},
			want: wordle{"", [6]wordGuess{
				{
					{'M', letterHitOrMiss{true, true}},
					{'A', letterHitOrMiss{true, true}},
					{'T', letterHitOrMiss{true, true}},
					{'C', letterHitOrMiss{true, true}},
					{'H', letterHitOrMiss{true, true}},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseForm(tt.args.wo, tt.args.form, tt.args.solutionWord); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseForm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_evaluateGuessedWord(t *testing.T) {
	type args struct {
		guessedWord  []string
		solutionWord wordleWord
	}
	tests := []struct {
		name string
		args args
		want wordGuess
	}{
		// test cases
		{
			name: "no hits, neither same or exact",
			args: args{[]string{}, wordleWord{'M', 'I', 'S', 'S', 'S'}},
			want: wordGuess{},
		},
		{
			name: "full exact match",
			args: args{
				[]string{"M", "A", "T", "C", "H"},
				wordleWord{'M', 'A', 'T', 'C', 'H'},
			},
			want: wordGuess{
				{'M', letterHitOrMiss{true, true}},
				{'A', letterHitOrMiss{true, true}},
				{'T', letterHitOrMiss{true, true}},
				{'C', letterHitOrMiss{true, true}},
				{'H', letterHitOrMiss{true, true}},
			},
		},
		{
			name: "partial exact and partial some match",
			args: args{
				[]string{"R", "A", "U", "L", "O"},
				wordleWord{'R', 'O', 'A', 'T', 'E'},
			},
			want: wordGuess{
				{'R', letterHitOrMiss{Some: true, Exact: true}},
				{'A', letterHitOrMiss{Some: true, Exact: false}},
				{'U', letterHitOrMiss{Some: false, Exact: false}},
				{'L', letterHitOrMiss{Some: false, Exact: false}},
				{'O', letterHitOrMiss{Some: true, Exact: false}},
			},
		},
		{
			name: "guessed word contains duplicats",
			args: args{
				[]string{"R", "O", "T", "O", "R"},
				wordleWord{'R', 'O', 'A', 'T', 'E'},
			},
			want: wordGuess{
				{'R', letterHitOrMiss{Some: true, Exact: true}},
				{'O', letterHitOrMiss{Some: true, Exact: true}},
				{'T', letterHitOrMiss{Some: true, Exact: false}},
				{'O', letterHitOrMiss{Some: false, Exact: false}}, // both false bec we already found it or even already guesst the exact match
				{'R', letterHitOrMiss{Some: false, Exact: false}}, // both false bec we already found it or even already guesst the exact match
			},
		},
		// {
		// 	name: "target word contains duplicats / guessed word contains duplicats",
		// 	args: args{
		// 		wordle{},
		// 		url.Values{"r0c0": []string{"M"}, "r0c1": []string{"A"}, "r0c2": []string{"T"}, "r0c3": []string{"C"}, "r0c4": []string{"H"}},
		// 		wordleWord{'M', 'A', 'T', 'C', 'H'},
		// 	},
		// 	want: wordle{"", [6][5]wordleLetter{
		// 		{
		// 			{'R', letterHitOrMiss{true, true}},
		// 			{'O', letterHitOrMiss{true, true}},
		// 			{'T', letterHitOrMiss{true, true}},
		// 			{'O', letterHitOrMiss{true, true}},
		// 			{'R', letterHitOrMiss{true, true}},
		// 		},
		// 	}},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluateGuessedWord(tt.args.guessedWord, tt.args.solutionWord); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evaluateGuessedWord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sessions_updateOrSet(t *testing.T) {
	type args struct {
		sess session
	}
	tests := []struct {
		name string
		ss   *sessions
		args args
		want sessions
	}{
		{
			"set new session",
			&sessions{},
			args{session{id: "foo"}},
			sessions{session{id: "foo"}},
		},
		{
			"update session",
			&sessions{session{id: "foo", maxAgeSeconds: 1}},
			args{session{id: "foo", maxAgeSeconds: 2}},
			sessions{session{id: "foo", maxAgeSeconds: 2}},
		},
		{
			"update session changes only correct session",
			&sessions{session{id: "foo"}, session{id: "bar"}, session{id: "baz", maxAgeSeconds: 1}, session{id: "foobar"}},
			args{session{id: "baz", maxAgeSeconds: 2}},
			sessions{session{id: "foo"}, session{id: "bar"}, session{id: "baz", maxAgeSeconds: 2}, session{id: "foobar"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ss.updateOrSet(tt.args.sess)
			if !reflect.DeepEqual((*tt.ss), tt.want) {
				t.Errorf("evaluateGuessedWord() = %v, want %v", tt.ss, tt.want)
			}
		})
	}
}
