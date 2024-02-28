package main

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

// func Test_generateSession(t *testing.T) {
// 	uuid.SetRand(rand.New(rand.NewSource(1)))

// 	tests := []struct {
// 		name string
// 		want session
// 	}{
// 		// TODO: Add test cases.
// 		{
// 			"name of test",
// 			session{
// 				id:        uuid.NewString(),
// 				expiresAt: time.Now(),
// 			},
// 		},
// 		{
// 			"name of test",
// 			session{
// 				id:        uuid.NewString(),
// 				expiresAt: time.Now(),
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := generateSession(); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("generateSession() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

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
		// TODO: Add test cases.
		{
			"test_name",
			args{session{fixedUuid, expireDate, SESSION_MAX_AGE_IN_SECONDS}},
			// http.Cookie{},
			http.Cookie{
				Name:     SESSION_COOKIE_NAME,
				Value:    fixedUuid,
				Path:     "/",
				MaxAge:   SESSION_MAX_AGE_IN_SECONDS,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				// Expires:  expireDate,
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
	tests := []struct {
		name string
		args args
		want session
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleSession(tt.args.w, tt.args.req, tt.args.sessions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleSession() = %v, want %v", got, tt.want)
			}
		})
	}
}
