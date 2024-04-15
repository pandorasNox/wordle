# lettr

## known issues
* If server restarts while playing, the next guess will end in a "fake rows" error. Resolved by reloading the page.

## plan of action
* [x] generate session (cookie) when none is present 
* [x] keep user data in server memory
* [ ] optional: session memory management based on cookie lifetime

## quiz
### what happens on server side
* [ ] word generation – requires: allowed word list, words to exclude (previous taken quizes)
* [80%] input validation (check matches) – requires: allowed word list, generated/current word, input of last word, number of tries
* [ ] trigger like quiz win/fail – requires: number of tries
* [ ] refactoring
    * [ ] rename wordle to `lettr` bec of trademark
    * [ ] session handling
    * [ ] use packages instead of everythiung in one file
* [ ] check out https://github.com/torenware/vite-go
    * https://vitejs.dev/guide/backend-integration VS https://www.npmjs.com/package/webpack-assets-manifest
