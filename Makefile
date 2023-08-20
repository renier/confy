
export VAULT_TOKEN = myroot
export VAULT_ADDR = http://127.0.0.1:8200

test:
	@if [ -z "$$(docker ps --filter 'name=vault-for-confy' --filter 'status=running' --quiet)" ]; then \
		docker run -p 8200:8200 --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=myroot' -d --name=vault-for-confy vault:0.11.6; \
		sleep 2; \
		vault secrets disable secret/; \
		vault secrets enable -path=secret -version=1 kv; \
	fi
	@while IFS=$$'\n' read -r line; do vault write secret/$$(echo "$$line" | cut -d ' ' -f 1) @fixtures/$$(echo "$$line" | cut -d ' ' -f 2); done < fixtures/paths.txt
	go test -race ./...

IMAGE := gcr.io/mission-e/confy-example

image:
	docker build --platform linux/amd64 -t $(IMAGE):latest .

push:
	docker push $(IMAGE):latest

.PHONY: test image push
