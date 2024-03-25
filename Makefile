
.PHONY: watch
watch:
	./scripts/tools.sh watch

.PHONY: test
test:
	./scripts/tools.sh test

.PHONY: down
down:
	./scripts/tools.sh down

.PHONY: skocli
skocli:
	./scripts/tools.sh skocli
