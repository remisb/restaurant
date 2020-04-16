package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/remisb/restaurant/internal/platform/auth"
	"github.com/remisb/restaurant/internal/platform/database"
	"github.com/remisb/restaurant/internal/platform/database/databasetest"
	"github.com/remisb/restaurant/internal/platform/web"
	"github.com/remisb/restaurant/internal/schema"
	"github.com/remisb/restaurant/internal/user"
	"log"
	"os"
	"testing"
	"time"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

type Test struct {
	DB *sqlx.DB
	Log *log.Logger
	Authenticator *auth.Authenticator

	t *testing.T
	cleanup func()
}

func NewUnit(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	c := databasetest.StartContainer(t)

	db, err := database.Open(database.Config{
		User: "postgres",
		Password: "postgres",
		Host: c.Host,
		Name: "postgres",
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("opening database connection: %v", err)
	}

	t.Log("waiting for database to be ready")

	// Wait for the database to be ready. Wait 100ms longer between each attempt.
	// Do not try more than 20 times.
	var pingError error
	maxAttempts := 20
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(maxAttempts) * 100 * time.Millisecond)
	}

	if pingError != nil {
		databasetest.DumpContainerLogs(t, c)
		databasetest.StopContainer(t, c)
		t.Fatalf("waiting for database to be ready: %v", pingError)
	}

	if err := schema.Migrate(db); err != nil {
		databasetest.StopContainer(t, c)
		t.Fatalf("migrating: %s", err)
	}

	teardown := func() {
		t.Helper()
		db.Close()
		databasetest.StopContainer(t, c)
	}

	return db, teardown
}

func NewIntegration(t *testing.T) *Test {
	t.Helper()

	// Initialize and seed database. Store the cleanup function call later.
	db, cleanup := NewUnit(t)
	if err := schema.Seed(db); err != nil {
		t.Fatal(err)
	}

	// Create the logger to use.
	logger := log.New(os.Stdout, "TEST : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// Create RSA keys to enable authentication in our service.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	// Build an authenticator using this static key.
	kid := "4754d86b-7a6d-4df5-9c65-224741361492"
	kf := auth.NewSimpleKeyLookupFunc(kid, key.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(key, kid, "RS256", kf)
	if err != nil {
		t.Fatal(err)
	}

	return &Test{
		DB: db,
		Log: logger,
		Authenticator: authenticator,
		t: t,
		cleanup: cleanup,
	}
}

func (test *Test) Teardown() {
	test.t.Helper()
	test.cleanup()
}

// Token generates an authenticated token for a user.
func (test *Test) Token(email, pass string) string {
	test.t.Helper()

	claims, err := user.Authenticate(
		context.Background(), test.DB, time.Now(),
		email, pass,
	)
	if err != nil {
		test.t.Fatal(err)
	}

	tkn, err := test.Authenticator.GenerateToken(claims)
	if err != nil {
		test.t.Fatal(err)
	}

	return tkn
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := web.Values{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
	}

	return context.WithValue(context.Background(), web.KeyValues, &values)
}

// StringPointer is a helper to get a *string from a string. It is in the tests
// package because we normally don't want to deal with pointers to basic types
// but it's useful in some tests.
func StringPointer(s string) *string {
	return &s
}

// IntPointer is a helper to get a *int from a int. It is in the tests package
// because we normally don't want to deal with pointers to basic types but it's
// useful in some tests.
func IntPointer(i int) *int {
	return &i
}

func LogInfo(t *testing.T, number int, msg string) {
	t.Helper()
	t.Logf("\tTest %d:\t%s", number, msg)
}

func LogInfof(t *testing.T, number int, format string, args ...interface{}) {
	t.Helper()
	t.Logf("\tTest %d:\t%s", number, fmt.Sprintf(format, args))
}

func LogSuccess(t *testing.T, msg string) {
	t.Helper()
	t.Log("\t" + Success + "\t" + msg)
}

func LogSuccessf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	LogStatus(t, false, fmt.Sprintf(format, args...))
}

func LogFail(t *testing.T, msg string) {
	t.Helper()
	t.Fatalf("\t" + Failed + "\t" + msg)
}

func LogFailf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	LogStatus(t, false, fmt.Sprintf(format, args...))
}

func LogStatus(t *testing.T, success bool, msg string) {
	t.Helper()
	outcome := Failed
	if success {
		outcome = Success
	}
	t.Log("\t" + outcome + "\t" + msg)
}

func LogFailErr(t *testing.T, msg string, err error) {
	t.Helper()
	t.Fatalf("\t" + Failed + "\t" + msg + err.Error())
}

func AssertStatusCode(t *testing.T, expect, actual int) {
	t.Helper()
	if expect != actual {
		LogFail(t, fmt.Sprintf("Should receive a status code of '%d' for the response : %v", expect, actual))
	}
	LogSuccess(t, fmt.Sprintf("Should receive a status code of '%d' for the response.", expect))
}

func AssertResponseBody(t *testing.T, expect, actual string) {
	t.Helper()

	ok := expect != actual
	if !ok {
		t.Log("Got :", actual)
		t.Log("Want:", expect)
	}
	LogStatus(t, ok,"Should get the expected result.")
}
