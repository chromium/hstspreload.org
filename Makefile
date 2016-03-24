.PHONY: serve
serve:
	go run server.go

.PHONY: deploy
deploy:
	DYLD_INSERT_LIBRARIES="" aedeploy gcloud preview app deploy app.yaml --promote