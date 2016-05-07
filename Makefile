PROJECT = github.com/chromium/hstspreload.appspot.com/...

# .PHONY: test
# test: lint
# 	go test ${PROJECT}

.PHONY: build
build:
	go build ${PROJECT}

.PHONY: format
format:
	go fmt
	# Need to specify non-default clang-format: https://crbug.com/558447
	/usr/local/bin/clang-format -i -style=Google files/static/js/*.js

.PHONY: lint
lint:
	go vet ${PROJECT}
	golint -set_exit_status ${PROJECT}

.PHONY: pre-commit
pre-commit: lint build

.PHONY: travis
travis: pre-commit

.PHONY: serve
serve: check
	go run *.go

.PHONY: deploy
deploy: check
	aedeploy gcloud preview app deploy app.yaml --promote

CURRENT_DIR = "$(shell pwd)"
EXPECTED_DIR = "${GOPATH}/src/github.com/chromium/hstspreload.appspot.com"

.PHONY: check
check:
ifeq (${CURRENT_DIR}, ${EXPECTED_DIR})
	@echo "PASS: Current directory is in \$$GOPATH."
else
	@echo "FAIL"
	@echo "Expected: ${EXPECTED_DIR}"
	@echo "Actual:   ${CURRENT_DIR}"
endif
