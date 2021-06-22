
include: .envrc
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
	go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN}

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