package handlers

import (
	"github.com/jmoiron/sqlx"
	"github.com/remisb/restaurant/internal/mid"
	"github.com/remisb/restaurant/internal/platform/auth"
	"github.com/remisb/restaurant/internal/platform/web"
	"log"
	"net/http"
	"os"
)

const (
	GET = "GET"
	PUT = "PUT"
	POST = "POST"
	DELETE = "DELETE"
)

func API(build string, shutdown chan os.Signal, log *log.Logger, db *sqlx.DB, authenticator *auth.Authenticator) http.Handler {
	app := web.NewApp(shutdown, mid.Logger(log), mid.Errors(log), mid.Metrics(), mid.Panics(log))

	check := Check{
		build: build,
		db: db,
	}

	app.Handle(GET, "/v1/health", check.Health)

	u := User{
		db: db,
		authenticator: authenticator,
	}

	app.Handle(GET, "/v1/users", u.List, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))
	app.Handle(POST, "/v1/users", u.Create, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))

	app.Handle(GET, "/v1/users/token", u.Token)

	// Register restaurant and menu endpoints.
	r := Restaurant{
		db: db,
	}
	app.Handle(GET, "/v1/restaurant", r.List, mid.Authenticate(authenticator))
	app.Handle(POST, "/v1/restaurant", r.Create, mid.Authenticate(authenticator))
	app.Handle(GET, "/v1/restaurant/:id", r.Retrieve, mid.Authenticate(authenticator))
	app.Handle(PUT, "/v1/restaurant/:id", r.Update, mid.Authenticate(authenticator))
	app.Handle(DELETE, "/v1/restaurant/:id", r.Delete, mid.Authenticate(authenticator))

	// restaurant menu handlers

	// Register restaurant and menu endpoints.
	m := Menu{
		db: db,
	}
	app.Handle(GET, "/v1/restaurant/:restaurantId/menu", m.RetrieveMenu, mid.Authenticate(authenticator))
	app.Handle(GET, "/v1/restaurant/:restaurantId/votes", m.RetrieveVotes, mid.Authenticate(authenticator))
	app.Handle(POST, "/v1/restaurant/:restaurantId/menu", m.CreateMenu, mid.Authenticate(authenticator), mid.HasRole(auth.RoleAdmin))
	return app
}
