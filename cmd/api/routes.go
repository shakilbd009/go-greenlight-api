package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Initialize a new httprouter router instance.
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	// Register the relevant methods, URL patterns and handler functions for our
	// endpoints using the HandlerFunc() method. Note that http.MethodGet and
	// http.MethodPost are constants which equate to the strings "GET" and "POST"
	// respectively.
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.requireActivatedUser(app.healthcheckHandler))

	// Use the requirePermission() middleware on each of the /v1/movies** endpoints,
	// passing in the required permission code as the first parameter.
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.requirePermission("movies:read", app.listMoviesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.requirePermission("movies:write", app.createMovieHandler))
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.requirePermission("movies:read", app.showMovieHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.requirePermission("movies:write", app.updateMovieHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.requirePermission("movies:write", app.deleteMovieHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	// Add the PUT /v1/users/password endpoint.
	router.HandlerFunc(http.MethodPut, "/v1/users/password", app.updateUserPasswordHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	// Add the POST /v1/tokens/password-reset endpoint.
	router.HandlerFunc(http.MethodPost, "/v1/tokens/password-reset", app.createPasswordResetTokenHandler)
	// Add the POST /v1/tokens/activation endpoint.
	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)

	router.Handler(http.MethodGet, "/v1/metrics", expvar.Handler())

	// Return the httprouter instance.”

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
