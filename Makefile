
app_name := config_api

db_env :=
db_env_file := .env
ifneq ($(strip $(db_env)),)
	db_env_file = .env.$(db_env)
endif


pg_dump := /usr/bin/env pg_dump
ifneq (, $(shell which pg_dump16))
	pg_dump = $(shell which pg_dump16)
endif

remote_alias += config_api

aarch64_bin := bin/$(app_name)-linux-aarch64-static

build-local:
	@echo "Building local"
	@go build -o bin/$(app_name) .

build-linux-aarch64-static:
	@echo "Building linux aarch64 static"
	@CC=aarch64-linux-musl-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build --ldflags '-linkmode external -extldflags "-static"' -o $(aarch64_bin)

$(aarch64_bin): build-linux-aarch64-static

deploy-linux-aarch64-static: $(aarch64_bin)
	@echo "Deploying linux aarch64 static"
	@# @scp $(aarch64_bin) $(remote_alias):/opt/config_api/bin/$(app_name)
	@xz -c $(aarch64_bin) | ssh $(remote_alias) "xz -dc | sudo install -m 755 -o root -g config_api /dev/stdin /opt/config_api/bin/$(app_name)"

deploy-env:
	@echo "Deploying env"
	@cat .env.prod | ssh -C $(remote_alias) "sudo install -m 755 -o root -g config_api /dev/stdin /opt/config_api/.env"

copy-scripts:
	@echo "Copying scripts"
	@tar -czf - Makefile scripts | ssh $(remote_alias) "sudo mkdir -p /opt/config_api && sudo tar -C /opt/config_api -xzf -"

build: build-linux-aarch64-static
deploy: deploy-linux-aarch64-static deploy-env

debug-listen:
	@sudo dlv attach --headless --listen 127.0.0.1:39001 --only-same-user=false --log --log-output debugger `pidof config_api`

migrate-new:
	@mkdir -p migrations
	@~/go/bin/godotenv -f $(db_env_file) /bin/bash -c '~/go/bin/goose -dir migrations postgres $${POSTGRES_URL} create $(name) sql'

migrate-status:
	@~/go/bin/godotenv -f $(db_env_file) /bin/bash -c '~/go/bin/goose -dir migrations postgres $${POSTGRES_URL} status'

migrate-up:
	@~/go/bin/godotenv -f $(db_env_file) /bin/bash -c '~/go/bin/goose -dir migrations postgres $${POSTGRES_URL} up'

migrate-down-ssh:
	~/go/bin/godotenv -f $(db_env_file) /bin/bash -c '~/go/bin/goose -dir migrations postgres $${POSTGRES_URL_SSH} down'

migrate-up-ssh:
	~/go/bin/godotenv -f $(db_env_file) /bin/bash -c '~/go/bin/goose -dir migrations postgres $${POSTGRES_URL_SSH} up'

connect-sql:
	~/go/bin/godotenv -f $(db_env_file) /bin/bash -c 'psql $${POSTGRES_URL}'

redis-cli:
	@~/go/bin/godotenv -f $(db_env_file) /bin/bash -c 'redis-cli -h $${REDIS_URL%*:*} -p $${REDIS_URL#*:}'

redis-mon:
	@~/go/bin/godotenv -f $(db_env_file) /bin/bash -c 'redis-cli -h $${REDIS_URL%*:*} -p $${REDIS_URL#*:} monitor'

dump-schema:
	~/go/bin/godotenv -f $(db_env_file) /bin/bash -c '$(pg_dump) -s $${POSTGRES_URL} > "schema-$$(date +%s).sql"'
