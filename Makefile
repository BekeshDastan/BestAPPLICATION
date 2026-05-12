.PHONY: help up down logs migrate migrate-user migrate-post migrate-chat proto test-unit test-int test lint build keygen

SERVICES   := user post chat story notification
COMPOSE    := docker compose -f deploy/docker-compose.yml
PG_DSN_user := host=localhost port=5432 user=social password=social dbname=user_db sslmode=disable
PG_DSN_post := host=localhost port=5432 user=social password=social dbname=post_db sslmode=disable
PG_DSN_chat := host=localhost port=5432 user=social password=social dbname=chat_db sslmode=disable

help:
	@echo ""
	@echo "  up            start infrastructure (postgres, redis, nats, minio, mailhog)"
	@echo "  down          stop and remove containers"
	@echo "  logs          tail container logs"
	@echo "  proto         regenerate gRPC Go code from all .proto files"
	@echo "  keygen        generate ECDSA key pair → keys/"
	@echo "  migrate-user  run goose migrations for user service"
	@echo "  migrate-post  run goose migrations for post service"
	@echo "  migrate-chat  run goose migrations for chat service"
	@echo "  test-unit     run unit tests for all services"
	@echo "  test-int      run integration tests (requires Docker)"
	@echo "  lint          run golangci-lint"
	@echo "  build         build all service binaries to bin/"
	@echo ""

up:
	$(COMPOSE) up -d
	@echo "✓ Infrastructure up. Postgres healthcheck may take ~10s."

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f

proto:
	@mkdir -p gen/go
	@for proto in api/proto/**/**/*.proto; do \
		protoc \
			--proto_path=api/proto \
			--go_out=gen/go --go_opt=paths=source_relative \
			--go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
			"$$proto"; \
	done
	@echo "✓ proto generation done"

keygen:
	@mkdir -p keys
	openssl ecparam -name prime256v1 -genkey -noout -out keys/private.pem
	openssl ec -in keys/private.pem -pubout -out keys/public.pem
	@echo "✓ keys/private.pem and keys/public.pem generated"

migrate-user:
	goose -dir backend/user/migrations postgres "$(PG_DSN_user)" up

migrate-post:
	goose -dir backend/post/migrations postgres "$(PG_DSN_post)" up

migrate-chat:
	goose -dir backend/chat/migrations postgres "$(PG_DSN_chat)" up

test-unit:
	@echo ">> unit tests: user"
	cd backend/user && go test ./tests/unit/... -v -count=1
	@echo ">> unit tests: post"
	cd backend/post && go test ./tests/unit/... -v -count=1
	@echo ">> unit tests: chat"
	cd backend/chat && go test ./tests/unit/... -v -count=1

test-int:
	@echo ">> integration tests: user (requires Docker)"
	cd backend/user && go test ./tests/integration/... -v -count=1 -timeout 120s
	@echo ">> integration tests: post (requires Docker)"
	cd backend/post && go test ./tests/integration/... -v -count=1 -timeout 120s
	@echo ">> integration tests: chat (requires Docker)"
	cd backend/chat && go test ./tests/integration/... -v -count=1 -timeout 120s

test: test-unit test-int

lint:
	cd backend/user && golangci-lint run ./...

build:
	@mkdir -p bin
	@for svc in $(SERVICES); do \
		echo ">> building $$svc"; \
		cd backend/$$svc && go build -o ../../bin/$$svc ./cmd && cd ../..; \
	done
