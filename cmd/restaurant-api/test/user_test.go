package test

import (
	"bytes"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/remisb/restaurant/cmd/restaurant-api/internal/handlers"
	"github.com/remisb/restaurant/internal/platform/web"
	"github.com/remisb/restaurant/internal/tests"
	"github.com/remisb/restaurant/internal/user"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUsers(t *testing.T) {
	test := tests.NewIntegration(t)
	defer test.Teardown()

	shutdown := make(chan os.Signal, 1)
	tests := UserTests{
		app:        handlers.API("develop", shutdown, test.Log, test.DB, test.Authenticator),
		userToken:  test.Token("user@example.com", "gophers"),
		adminToken: test.Token("admin@example.com", "gophers"),
	}

	t.Run("getToken401", tests.getToken401)
	t.Run("getToken200", tests.getToken200)
	t.Run("postUser400", tests.postUser400)
	t.Run("postUser401", tests.postUser401)
	t.Run("postUser403", tests.postUser403)
}

// UserTests holds methods for each user subtest. This type allows passing
// dependencies for tests while still providing a convenient syntax when
// subtests are registered.
type UserTests struct {
	app        http.Handler
	userToken  string
	adminToken string
}

func (ut *UserTests) getToken401(t *testing.T) {
	r := httptest.NewRequest(GET, "/v1/users/token", nil)
	r.SetBasicAuth("unknown@example.com", "some-password")
	w := httptest.NewRecorder()
	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to deny tokens to unknown users.")
	{
		tests.LogInfof(t, 0, "When fetching a token with an unrecognized email.")
		tests.AssertStatusCode(t, http.StatusUnauthorized, w.Code)
	}
}

// getToken200
func (ut *UserTests) getToken200(t *testing.T) {

	r := httptest.NewRequest(GET, "/v1/users/token", nil)
	r.SetBasicAuth("admin@example.com", "gophers")
	w := httptest.NewRecorder()

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to issues tokens to known users.")
	{
		tests.LogInfo(t, 0, "When fetching a token with valid credentials.")
		{
			tests.AssertStatusCode(t, http.StatusOK, w.Code)

			var got struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				tests.LogFailf(t, "Should be able to unmarshal the response : %v", err)
			}
			tests.LogSuccess(t, "Should be able to unmarshal the response.")

			// TODO(jlw) Should we ensure the token is valid?
		}
	}
}

// postUser400 validates a user can't be created with the endpoint
// unless a valid user document is submitted.
func (ut *UserTests) postUser400(t *testing.T) {
	body, err := json.Marshal(&user.NewUser{})
	if err != nil {
		t.Fatal(err)
	}
	r := createRequestBody(POST, "/v1/users", ut.adminToken, bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new user can't be created with an invalid document.")
	{
		tests.LogInfo(t, 0, "When using an incomplete user value.")
		{
			tests.AssertStatusCode(t, http.StatusBadRequest, w.Code)

			// Inspect the response.
			var got web.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				tests.LogFailf(t, "Should be able to unmarshal the response to an error type : %v", err)
			}
			tests.LogSuccess(t, "Should be able to unmarshal the response to an error type.")

			// Define what we want to see.
			want := web.ErrorResponse{
				Error: "field validation error",
				Fields: []web.FieldError{
					{Field: "name", Error: "name is a required field"},
					{Field: "email", Error: "email is a required field"},
					{Field: "roles", Error: "roles is a required field"},
					{Field: "password", Error: "password is a required field"},
				},
			}

			// We can't rely on the order of the field errors so they have to be
			// sorted. Tell the cmp package how to sort them.
			sorter := cmpopts.SortSlices(func(a, b web.FieldError) bool {
				return a.Field < b.Field
			})

			if diff := cmp.Diff(want, got, sorter); diff != "" {
				tests.LogFailf(t, "Should get the expected result. Diff:\n%s", diff)
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}
}

// postUser401 validates a user can't be created unless the calling user is
// authenticated.
func (ut *UserTests) postUser401(t *testing.T) {
	body, err := json.Marshal(&user.User{})
	if err != nil {
		t.Fatal(err)
	}

	r := createRequestBody(POST, "/v1/users", ut.userToken, bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new user can't be created with an invalid document.")
	tests.LogInfo(t, 0, "When using an incomplete user value.")
	tests.AssertStatusCode(t, http.StatusForbidden, w.Code)
}

// postUser403 validates a user can't be created unless the calling user is
// an admin user. Regular users can't do this.
func (ut *UserTests) postUser403(t *testing.T) {
	body, err := json.Marshal(&user.User{})
	if err != nil {
		t.Fatal(err)
	}

	// Not setting the Authorization header
	r := createRequestBody(POST, "/v1/users", "", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new user can't be created with an invalid document.")
	tests.LogInfo(t, 0, "When using an incomplete user value.")
	tests.AssertStatusCode(t, http.StatusUnauthorized, w.Code)
}

// getUser400 validates a user request for a malformed userid.
func (ut *UserTests) getUser400(t *testing.T) {
	id := "12345"

	r := createRequestBody(GET, "/v1/users/"+id, ut.userToken, nil)
	w := httptest.NewRecorder()
	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a user with a malformed userid.")
	{
		tests.LogInfof(t, 0, "When using the new user %s.", id)
		{
			tests.AssertStatusCode(t, http.StatusBadRequest, w.Code)
			expect := `{"error":"ID is not in its proper form"}`
			tests.AssertResponseBody(t, expect, w.Body.String())
		}
	}
}
