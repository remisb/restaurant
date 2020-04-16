package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/remisb/restaurant/cmd/restaurant-api/internal/handlers"
	"github.com/remisb/restaurant/internal/platform/web"
	"github.com/remisb/restaurant/internal/restaurant"
	"github.com/remisb/restaurant/internal/tests"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const (
	GET    = "GET"
	PUT    = "PUT"
	POST   = "POST"
	DELETE = "DELETE"
)

type RestaurantTests struct {
	app        http.Handler
	userToken  string
	adminToken string
}

// TestRestaurants runs a series of tests to exercise Restaurant behavior from the
// API level. The subtests all share the same database and application for
// speed and convenience. The downside is the order the tests are ran matters
// and one test may break if other tests are not ran before it. If a particular
// subtest needs a fresh instance of the application it can make it or it
// should be its own Test* function.
func TestRestaurants(t *testing.T) {
	test := tests.NewIntegration(t)
	defer test.Teardown()

	shutdown := make(chan os.Signal, 1)
	restaurantTests := RestaurantTests{
		app:        handlers.API("develop", shutdown, test.Log, test.DB, test.Authenticator),
		userToken:  test.Token("user@example.com", "gophers"),
		adminToken: test.Token("admin@example.com", "gophers"),
	}

	t.Run("postRestaurant400", restaurantTests.postRestaurant400)
	t.Run("postRestaurant401", restaurantTests.postRestaurant401)
	t.Run("getRestaurant404", restaurantTests.getRestaurant404)
	t.Run("getRestaurant400", restaurantTests.getRestaurant400)
	t.Run("deleteRestaurantNotFound", restaurantTests.deleteRestaurantNotFound)
	t.Run("deleteRestaurant400", restaurantTests.deleteRestaurant400)
	t.Run("putRestaurant404", restaurantTests.putRestaurant404)
	t.Run("putRestaurant400", restaurantTests.putRestaurant400)
	t.Run("crudRestaurants", restaurantTests.crudRestaurant)

	t.Run("postMenu400", restaurantTests.postMenu400)
	t.Run("postMenu201", restaurantTests.postMenu201)
	t.Run("crudMenu", restaurantTests.crudMenu)

}

// postRestaurant400 validates a restaurant can't be created with the endpoint
// unless a valid restaurant is submitted.
func (rt *RestaurantTests) postRestaurant400(t *testing.T) {
	r := createRequestBody(POST, "/v1/restaurant", rt.userToken, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new restaurant can't be created with and invalid document.")
	{
		t.Log("\tTest 0:\tWhen using and incomplete restaurant value.")
		{
			tests.AssertStatusCode(t, http.StatusBadRequest, w.Code)

			// Inspect the response
			var got web.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response to an error type: %v", tests.Failed, err)
			}
			t.Logf("\t%s\tShould be able to unmarshal the response to an error type.", tests.Success)

			want := web.ErrorResponse{
				Error: "field validation error",
				Fields: []web.FieldError{
					{Field: "name", Error: "name is a required field"},
					{Field: "address", Error: "address is a required field"},
				},
			}

			sorter := cmpopts.SortSlices(func(a, b web.FieldError) bool {
				return a.Field < b.Field
			})

			if diff := cmp.Diff(want, got, sorter); diff != "" {
				tests.LogFail(t, fmt.Sprintf("Should get the expected result. Diff:\n%s", diff))
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}
}

// postRestaurant401 validates a restaurant can't be created with the endpoint
// unless the user is authenticated
func (rt *RestaurantTests) postRestaurant401(t *testing.T) {
	nr := restaurant.NewRestaurant{
		Name:    "Restaurant No1",
		Address: "Address of restaurant no1",
	}

	body, err := json.Marshal(&nr)
	if err != nil {
		t.Fatal(err)
	}

	r := createRequestBody(POST, "/v1/restaurant", rt.userToken, bytes.NewBuffer(body))
	r.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	rt.app.ServeHTTP(w, r)
	t.Log("Given the need to validate a new restaurant can't be created with an invalid document.")
	{
		t.Log("\tTest 0:\tWhen using an incomplete restaurant value.")
		{
			tests.AssertStatusCode(t, http.StatusUnauthorized, w.Code)
		}
	}
}

// getRestaurant400 validates a restaurant request for a malformed id.
func (rt *RestaurantTests) getRestaurant400(t *testing.T) {
	id := "12345"

	r := createRequest(GET, "/v1/restaurant/"+id, rt.userToken)
	w := httptest.NewRecorder()

	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a restaurant with a malformed id.")
	{
		t.Logf("\tTest 0:\tWhen using the new restaurant %s.", id)
		{
			tests.AssertStatusCode(t, http.StatusBadRequest, w.Code)

			recv := w.Body.String()
			resp := `{"error":"ID is not in its proper form"}`
			if resp != recv {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}
}

func (rt *RestaurantTests) getRestaurant404(t *testing.T) {
	id := "a224a8d6-3f9e-4b11-9900-e81a25d80702"

	r := createRequest(GET, "/v1/restaurant/"+id, rt.userToken)
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a restaurant with an unknown id")
	{
		tests.LogInfof(t, 0, "When using the new restaurant %s.", id)
		{
			tests.AssertStatusCode(t, http.StatusNotFound, w.Code)

			recv := w.Body.String()
			resp := "Restaurant not found"
			if !strings.Contains(recv, resp) {
				t.Log("Got ", recv)
				t.Log("Want:", resp)
				tests.LogFail(t, "Should get the expected result.")
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}
}

func (rt *RestaurantTests) deleteRestaurant400(t *testing.T) {
	id := "12345"

	r := createRequest(DELETE, "/v1/restaurant/"+id, rt.userToken)
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate deleting a restaurant with a malformed id.")
	{
		tests.LogInfof(t, 0, "When using malformed id %s", id)
		{
			tests.AssertStatusCode(t, http.StatusBadRequest, w.Code)

			recv := w.Body.String()
			resp := `{"error":"ID is not in its proper form"}`
			if resp != recv {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}

}

func (rt *RestaurantTests) deleteRestaurantNotFound(t *testing.T) {
	id := "a224a8d6-3f9e-4b11-9900-e81a25d80702"

	r := createRequest(DELETE, "/v1/restaurant/"+id, rt.userToken)
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate deleting a restaurant that does not exist.")
	{
		t.Logf("\tTest 0:\tWhen using the new restaurant %s.", id)
		{
			tests.AssertStatusCode(t, http.StatusNoContent, w.Code)
		}
	}
}

// putRestaurant404 validates updating a restaurant that does not exist.
func (rt *RestaurantTests) putRestaurant400(t *testing.T) {
	up := restaurant.UpdateRestaurant{
		Name: tests.StringPointer("Nonexistent"),
	}

	id := "12345"

	body, err := json.Marshal(&up)
	if err != nil {
		t.Fatal(err)
	}

	r := createRequestBody(PUT, "/v1/restaurant/"+id, rt.userToken, bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate updating a restaurant with malformed id.")
	{
		tests.LogInfof(t, 0, "When using the new restaurant %s.", id)
		{
			tests.AssertStatusCode(t, http.StatusBadRequest, w.Code)

			recv := w.Body.String()
			resp := `{"error":"ID is not in its proper form"}`
			if resp != recv {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}

}

// putRestaurant404 validates updating a restaurant that does not exist.
func (rt *RestaurantTests) putRestaurant404(t *testing.T) {
	up := restaurant.UpdateRestaurant{
		Name: tests.StringPointer("Nonexistent"),
	}

	id := "9b468f90-1cf1-4377-b3fa-68b450d632a0"

	body, err := json.Marshal(&up)
	if err != nil {
		t.Fatal(err)
	}

	r := createRequestBody(PUT, "/v1/restaurant/"+id, rt.userToken, bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate updating a restaurant that does not exist.")
	{
		tests.LogInfof(t, 0, "When using the new restaurant %s.", id)
		{
			tests.AssertStatusCode(t, http.StatusNotFound, w.Code)

			recv := w.Body.String()
			resp := "Restaurant not found"
			if !strings.Contains(recv, resp) {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}
}

// crudRestaurant performs a complete test of CRUD against the api.
func (rt *RestaurantTests) crudRestaurant(t *testing.T) {
	r := rt.postRestaurant201(t)
	defer rt.deleteRestaurant204(t, r.ID)

	rt.getRestaurant200(t, r.ID)
	rt.putRestaurant204(t, r.ID)
	rt.getRestaurants200(t)
}

// postRestaurant201 validates a restaurant can be created with the endpoint.
func (rt *RestaurantTests) postRestaurant201(t *testing.T) restaurant.Restaurant {
	nr := restaurant.NewRestaurant{
		Name:    "Restaurant name",
		Address: "Restaurant address",
	}

	body, err := json.Marshal(&nr)
	if err != nil {
		t.Fatal(err)
	}

	r := createRequestBody(POST, "/v1/restaurant", rt.userToken, bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	//r.Header.Set("Authorization", "Bearer "+rt.userToken)

	rt.app.ServeHTTP(w, r)

	// p is the value we will return.
	var rest restaurant.Restaurant

	t.Log("Given the need to create a new restaurant with the restaurant endpoint.")
	{
		tests.LogInfo(t, 0, "When using the declared restaurant value.")
		{
			tests.AssertStatusCode(t, http.StatusCreated, w.Code)

			if err := json.NewDecoder(w.Body).Decode(&rest); err != nil {
				tests.LogFailErr(t, "Should be able to unmarshal the response : ", err)
			}

			// Define what we wanted to receive. We will just trust the generated
			// fields like ID and Dates so we copy p.
			want := rest
			want.Name = "Restaurant name"
			want.Address = "Restaurant address"

			if diff := cmp.Diff(want, rest); diff != "" {
				tests.LogFail(t, fmt.Sprintf("Should get the expected result. Diff:\n%s", diff))
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}

	return rest
}

// deleteRestaurant200 validates deleting a restaurant that does exist.
func (rt *RestaurantTests) deleteRestaurant204(t *testing.T, id string) {
	r := createRequest(DELETE, "/v1/restaurant/"+id, rt.userToken)
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate deleting a restaurant that does exist.")
	//{
	tests.LogInfo(t, 0, "When using the new restaurant "+id+".")
	//{
	tests.AssertStatusCode(t, http.StatusNoContent, w.Code)
	//}
	//}
}

// getRestaurants200 validates a restaurant list request for an existing restaurants.
func (rt *RestaurantTests) getRestaurants200(t *testing.T) {
	r := createRequest(GET, "/v1/restaurant", rt.userToken)
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting list of existing restaurants.")
	{
		tests.LogInfo(t, 0, "When retrieving all restaurants.")
		{
			tests.AssertStatusCode(t, http.StatusOK, w.Code)

			var r []restaurant.Restaurant
			if err := json.NewDecoder(w.Body).Decode(&r); err != nil {
				tests.LogFail(t, fmt.Sprintf("Should be able to unmarshal the response : %v", err))
			}

			tests.LogStatus(t, len(r) == 2, "List should containt 2 restaurants.")
			// Define what we wanted to receive. We will just trust the generated
			// fields like Dates so we copy p.

			//want := r
			//want.ID = id
			//want.Name = "Restaurant name"
			//want.Address = "Restaurant address"
			//
			//if diff := cmp.Diff(want, r); diff != "" {
			//	tests.LogFail(t, fmt.Sprintf("Should get the expected result. Diff:\n%s", diff))
			//}
			//tests.LogSuccess(t, "Should get the expected result.")
		}
	}
}

// getRestaurant200 validates a restaurant request for an existing id.
func (rt *RestaurantTests) getRestaurant200(t *testing.T, id string) {
	r := createRequest(GET, "/v1/restaurant/"+id, rt.userToken)
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a restaurant that exists.")
	{
		tests.LogInfo(t, 0, fmt.Sprintf("When using the new restaurant %s.", id))
		{
			tests.AssertStatusCode(t, http.StatusOK, w.Code)

			var r restaurant.Restaurant
			if err := json.NewDecoder(w.Body).Decode(&r); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response : %v", tests.Failed, err)
			}

			// Define what we wanted to receive. We will just trust the generated
			// fields like Dates so we copy p.
			want := r
			want.ID = id
			want.Name = "Restaurant name"
			want.Address = "Restaurant address"

			if diff := cmp.Diff(want, r); diff != "" {
				tests.LogFail(t, fmt.Sprintf("Should get the expected result. Diff:\n%s", diff))
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}
}

// putRestaurant204 validates updating a restaurant that does exist.
func (rt *RestaurantTests) putRestaurant204(t *testing.T, id string) {
	body := `{"name": "Test restaurant", "Address": "test address"}`

	r := createRequestBody(PUT, "/v1/restaurant/"+id, rt.userToken, strings.NewReader(body))
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to update a restaurant with the restaurant endpoint.")
	{
		tests.LogInfo(t, 0, "When using the modified restaurant value.")
		{
			tests.AssertStatusCode(t, http.StatusNoContent, w.Code)

			r = createRequest(GET, "/v1/restaurant/"+id, rt.userToken)
			w = httptest.NewRecorder()
			rt.app.ServeHTTP(w, r)

			tests.AssertStatusCode(t, http.StatusOK, w.Code)

			var ru restaurant.Restaurant
			if err := json.NewDecoder(w.Body).Decode(&ru); err != nil {
				tests.LogFailf(t, "Should be able to unmarshal the response : %v", err)
			}

			if ru.Name != "Test restaurant" {
				tests.LogFailf(t, "Should see an updated Name : got %q want %q", ru.Name, "Test restaurant")
			}
			tests.LogSuccess(t, "Should see an updated Name.")
		}
	}
}

func createRequest(method, url, token string) *http.Request {
	return createRequestBody(method, url, token, nil)
}

func createRequestBody(method, url, token string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, url, body)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}
