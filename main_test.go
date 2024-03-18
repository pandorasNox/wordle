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
			args{session{fixedUuid, expireDate, SESSION_MAX_AGE_IN_SECONDS, wordleWord{}}},
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
				id:            "12345678-abcd-1234-abcd-ab1234567890",
				expiresAt:     time.Unix(1615256178, 0).Add(SESSION_MAX_AGE_IN_SECONDS * time.Second),
				maxAgeSeconds: 120,
				activeWord:    wordleWord{'R', 'O', 'A', 'T', 'E'},
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
		wo   wordle
		form url.Values
		w    wordleWord
	}
	tests := []struct {
		name string
		args args
		want wordle
	}{
		// TODO: Add test cases.
		{
			name: "no hits, neither same or exact",
			args: args{wordle{}, url.Values{}, wordleWord{'M', 'I', 'S', 'S', 'S'}},
			want: wordle{},
		},
		{
			name: "full exact match",
			args: args{
				wordle{},
				url.Values{"r0c0": []string{"M"}, "r0c1": []string{"A"}, "r0c2": []string{"T"}, "r0c3": []string{"C"}, "r0c4": []string{"H"}},
				wordleWord{'M', 'A', 'T', 'C', 'H'},
			},
			want: wordle{"", [6][5]wordleLetter{
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
			if got := parseForm(tt.args.wo, tt.args.form, tt.args.w); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseForm() = %v, want %v", got, tt.want)
			}
		})
	}
}
