PROJECT = github.com/chromium/hstspreload.org/...

.PHONY: build
build:
	go build ${PROJECT}

.PHONY: lint
lint:
	go vet ${PROJECT}
	go install honnef.co/go/tools/cmd/staticcheck@v0.5.1
	staticcheck ${PROJECT}

.PHONY: pre-commit
pre-commit: lint build test

.PHONY: deploy
deploy: version
	date
	time gcloud app deploy app.yaml cron.yaml
	date

# Version file.

.PHONY: version
version:
	git rev-parse HEAD > ./frontend/version

# Google Cloud Datastore Emulator

GCD_NAME = gcd-grpc-1.0.0
XDG_CACHE_HOME ?= $(HOME)/.cache
DATABASE_TESTING_FOLDER = ${XDG_CACHE_HOME}/datastore-emulator

.PHONY: get-datastore-emulator
get-datastore-emulator: ${DATABASE_TESTING_FOLDER}/gcd/gcd.sh
${DATABASE_TESTING_FOLDER}/gcd/gcd.sh:
	mkdir -p "${DATABASE_TESTING_FOLDER}"
	curl "https://storage.googleapis.com/gcd/tools/${GCD_NAME}.zip" -o "${DATABASE_TESTING_FOLDER}/${GCD_NAME}.zip"
	unzip "${DATABASE_TESTING_FOLDER}/${GCD_NAME}.zip" -d "${DATABASE_TESTING_FOLDER}"
	rm "${DATABASE_TESTING_FOLDER}/${GCD_NAME}.zip"

# Testing

.PHONY: serve
serve: get-datastore-emulator version
	go run *.go -local

.PHONY: test
test: get-datastore-emulator
	go test -v -cover "${PROJECT}"
