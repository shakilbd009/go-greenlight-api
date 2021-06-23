
include:	.envrc
# ==================================================================================== #
# HELPERS
# ==================================================================================== #
# help: print this help message
#.PHONY	help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'



# run/api: run the cmd/api application
#.PHONY	run/api
run/api:
	go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN} -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASSWORD}

# db/psql: connect to the database using psql
#.PHONY	db/sql
db/psql:
	psql ${GREENLIGHT_DB_DSN}
# Create the new confirm target.
#.PHONY	confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# db/migrations/up: apply all up database migrations
#.PHONY	db/migrations/up
db/migrations/up:	confirm
	@echo "Running up Migrations..."
	migrate -path ./migrations -database ${GREENLIGHT_DB_DSN} up

# db/migrations/new name=$1: create a new database migration
#.PHONY	db/migrations/new
db/migrations/new:
	@echo "creating mi files for ${name}..."
	migrate create -seq -ext=.sql -dir=./migrations ${name}

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #


## audit: tidy dependencies and format, vet and test all code
audit: vendor
	@echo 'Formating code...'
	go fmt ./...
	@echo 'vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...


## vendor: tidy and vendor dependencies
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor
# ==================================================================================== #
# BUILD
# ==================================================================================== #

current_time = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.version=${git_description}'
## build/api: build the cmd/api application
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/api ./cmd/api