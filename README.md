# lettr

## known issues
* If server restarts while playing, the next guess will end in a "fake rows" error. Resolved by reloading the page.

## plan of action
* [x] generate session (cookie) when none is present 
* [x] keep user data in server memory
* [ ] optional: session memory management based on cookie lifetime

## quiz
### what happens on server side
* [80%] word generation – requires: allowed word list, words to exclude (previous taken quizes)
* [x] input validation (check matches) – requires: allowed word list, generated/current word, input of last word, number of tries
* [x] trigger like quiz win/fail – requires: number of tries
* [ ] refactoring
    * [x] rename wordle to `lettr` bec of trademark
    * [ ] session handling
    * [ ] use packages instead of everythiung in one file
* [ ] check out https://github.com/torenware/vite-go
    * https://vitejs.dev/guide/backend-integration VS https://www.npmjs.com/package/webpack-assets-manifest

## open todo
- must
    * [x] word not InWordDB
    * [x] website app version via ??? (assets? or on webpage?)
    * [x] (mobile) keyboard (use + indication for used letters)
    * [x] protect against request size + correct http 413 error code throw
    * [ ] fix scripts/tools.sh func_exec_cli passing parameter issue
    * [ ] editorial work: e.g. words like games or gamer are missing + maybe we introduce a common vs uncommen word list
    * [ ] bugfix: full page get form submit request on random occations when it should just be a htmx post
- nice-to-have
    * [ ] get definition (e.g. wikitionary)
    * [ ] hint feature / give me one letter
    * [ ] ui languge should also change
    * [ ] ESLint
    * [ ] http error codes:
        * [ ] 414 URI Too Long
        * [ ] 431 Request Header Fields Too Large (RFC 6585)
