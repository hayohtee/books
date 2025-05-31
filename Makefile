include .env

MIGRATIONS_PATH = ./migrations

.PHONY: migrate-create
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=<filename>" \
		exit 1; \
	fi
	@echo "Creating migration files for $(name)"
	@migrate create -seq -ext .sql -dir ${MIGRATIONS_PATH} $(name)

.PHONY: migrate-up
migrate-up:
	@echo "Running up migrations"
	@migrate -path ${MIGRATIONS_PATH} -database ${DATABASE_DSN} up

.PHONY: migrate-down
migrate-down:
	@echo "Running down migrations"
	@migrate -path ${MIGRATIONS_PATH} -database ${DATABASE_DSN} down

.PHONY: migrate-drop
migrate-drop:
	@echo "Dropping all tables"
	@migrate -path ${MIGRATIONS_PATH} -database ${DATABASE_DSN} drop

.PHONY: compose-up
compose-up:
	@echo "Running docker compose up"
	@docker compose up --build -d

.PHONY: compose-stop
compose-stop:
	@echo "Stopping docker compose"
	@docker compose stop

.PHONY: compose-rm
compose-rm:
	@echo "Removing docker compose services"
	@docker compose rm

.PHONY: sqlc-generate
sqlc-generate:
	@echo "Generating sql queries"
	@sqlc generate

.PHONY: oapi-generate
oapi-generate:
	@echo "generating server from OpenAPI specification"
	@oapi-codegen --config oapi-codegen.yaml ./api/openapi.yaml
