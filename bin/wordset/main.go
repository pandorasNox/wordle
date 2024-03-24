package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// {
//     "largely": {
//         "word": "largely",
//         "wordset_id": "54bd55df7265742391cf0000",
//         "meanings": [{
//             "id": "54bd55df7265742391d10000",
//             "def": "in large part",
//             "speech_part": "adverb",
//             "synonyms": ["mostly", "for the most part"]
//         }, {
//             "id": "54bd55df7265742391d20000",
//             "def": "on a large scale",
//             "example": "the sketch was so largely drawn that you could see it from the back row",
//             "speech_part": "adverb"
//         }]
//     }
// }

type Entries map[string]Entry

type Entry struct {
	Word string
}

func main() {
	urlFormat := "https://raw.githubusercontent.com/wordset/wordset-dictionary/master/data/%s.json"
	dataSetNames := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "misc", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	// dataSetNames := []string{"z"}

	all5LetterWords := []string{}
	for _, dsn := range dataSetNames {
		url := fmt.Sprintf(urlFormat, dsn)

		b := fetch(url)
		es := parseBody(b)

		for _, e := range es {
			if len(e.Word) == 5 && (!strings.ContainsAny(e.Word, " '-")) {
				all5LetterWords = append(all5LetterWords, e.Word)
			}
		}

		// fmt.Printf("enties:\n%v\n", es)
	}

	fmt.Printf("len(all5LetterWords):\n%d\n", len(all5LetterWords))
	fmt.Println("")
	fmt.Printf("all5LetterWords:\n%v\n", all5LetterWords)
}

func fetch(url string) (body []byte) {
	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "golang-commandline-tool")

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	return body
}

func parseBody(body []byte) Entries {
	e := Entries{}
	jsonErr := json.Unmarshal(body, &e)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return e
}
