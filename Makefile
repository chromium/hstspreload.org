.PHONY: serve
serve: check
	go run *.go

.PHONY: deploy
deploy: check
	aedeploy gcloud preview app deploy app.yaml --promote

CURRENT_DIR = "$(shell pwd)"
EXPECTED_DIR = "${GOPATH}/src/github.com/chromium/hstspreload/hstspreload.appspot.com"

.PHONY: check
check:
ifeq (${CURRENT_DIR}, ${EXPECTED_DIR})
	@echo "PASS: Current directory is in \$$GOPATH."
else
	@echo "FAIL"
	@echo "Expected: ${EXPECTED_DIR}"
	@echo "Actual:   ${CURRENT_DIR}"
endif
