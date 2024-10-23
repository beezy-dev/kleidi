default: help

.PHONY: help
		
clean:
	@echo "--> cleaning existing vault instance"
	@podman rm vault --force


vault: export VAULT_ADDR=http://127.0.0.1:8200
vault: export VAULT_TOKEN="kleidi-demo"
vault: export VAULT_SKIP_VERITY="true"
vault: 
	@echo "--> starting fresh vault instance"
	@podman run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=kleidi-demo' -e 'VAULT_LOG_LEVEL=debug' -d --name=vault --network kind --ip 10.89.0.10 -p 8200:8200 hashicorp/vault
	vault secrets enable transit 


hvault:
	@echo "--> running kleidi from code"
	@go run cmd/kleidi/main.go -provider hvault

run-vault: clean vault hvault



