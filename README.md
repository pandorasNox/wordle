# wordle

## known issues
* If server restarts while playing, the next guess will end in a "fake rows" error. Resolved by reloading the page.

## plan of action
* [x] generate session (cookie) when none is present 
* [ ] keep user data in server memory
* [ ] optional: session memory management based on cookie lifetime

## quiz
### what happens on server side
* [ ] word generation – requires: allowed word list, words to exclude (previous taken quizes)
* [ ] input validation (check matches) – requires: allowed word list, generated/current word, input of last word, number of tries
* [ ] trigger like quiz win/fail – requires: number of tries
