configure: ## Register the skill on the Atomist platform
	docker run --init -it --rm --pull=always \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $${PWD}:/skill \
		atomist/local-skill --pwd /skill --workspace $${ATOMIST_WORKSPACE} --apikey $${ATOMIST_API_KEY} --host-dir $${PWD} --watch

run: ## Starts the skill: an HTTP server listening on http://localhost:8080
	go run "."