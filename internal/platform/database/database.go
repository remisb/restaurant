package database

import (
	"context"
	"github.com/jmoiron/sqlx"
	"go.opencensus.io/trace"
	"net/url"
)

// Config is used to hold the required properties to use database.
type Config struct {
	User string
	Password string
	Host string
	Name string
	DisableTLS bool
}
func Open(cfg Config) (*sqlx.DB, error) {

	sslMode := "rquire"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:     "postgres",
		User:       url.UserPassword(cfg.User, cfg.Password),
		Host:       cfg.Host,
		Path:       cfg.Name,
		RawQuery:   q.Encode(),
	}

	return sqlx.Open("postgres", u.String())
}

// StatusCheck returns nil if it can successfully talk to the database. It
// returns a non-nil error otherwise.
func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.StatusCheck")
	defer span.End()

	// Run a simple query to determine connectivity. The db has a "Ping" method
	// but it can false-positive when it was previously able to talk to the
	// database but the database has since gone away. Running this query forces a
	// round trip to the database.
	const q = `SELECT true`
	var tmp bool
	return db.QueryRowContext(ctx, q).Scan(&tmp)
}
