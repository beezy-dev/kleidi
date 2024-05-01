default: help

.PHONY: help


.PHONY: commit
commit: ## quick commit
		git add --all . && git commit -m "makefile commit"

.PHONY: build
build: ## container build
		./containerBuild.sh