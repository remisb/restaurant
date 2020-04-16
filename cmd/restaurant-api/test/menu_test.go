package test

import (
	"bytes"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"github.com/remisb/restaurant/internal/restaurant"
	"github.com/remisb/restaurant/internal/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// crudRestaurant performs a complete test of CRUD against the api.
func (rt *RestaurantTests) crudMenu(t *testing.T) {
	r := rt.postRestaurant201(t)
	defer rt.deleteRestaurant204(t, r.ID)

	rt.getRestaurant200(t, r.ID)
	rt.putRestaurant204(t, r.ID)
	rt.getRestaurants200(t)
}

// postMenu201 validates a restaurant menu can be created with the endpoint.
func (mt *RestaurantTests) postMenu201(t *testing.T) {

	location := time.FixedZone("UTC+03", 3*60*60)
	newMenu := restaurant.NewMenu{
		RestaurantID: "a224a8d6-3f9e-4b11-9900-e81a25d80702",
		Date:         time.Date(2020, 03, 10, 0, 0, 0, 0, location),
		Menu:         "Test menu content",
	}

	body, err := json.Marshal(&newMenu)
	if err != nil {
		t.Fatal(err)
	}

	r := createRequestBody(POST, "/v1/restaurant/"+newMenu.RestaurantID+"/menu", mt.adminToken, bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	mt.app.ServeHTTP(w, r)

	var m restaurant.Menu

	t.Log("Given the need to create a new menu for specified restaurant for current day with the endpoint.")
	{
		tests.LogInfo(t, 0, "When using the declared menu value.")
		{
			tests.AssertStatusCode(t, http.StatusCreated, w.Code)

			if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
				tests.LogFailf(t, "Should be able to unmarshal the response : %v", err)
			}

			want := m
			want.Votes = 0
			want.RestaurantID = newMenu.RestaurantID
			want.Date = newMenu.Date
			want.Menu = newMenu.Menu

			if diff := cmp.Diff(want, m); diff != "" {
				tests.LogFailf(t, "Should get the expected results. Diff:\n%s", diff)
			}
			tests.LogSuccess(t, "Should get the expected result.")
		}
	}

}

func (rt *RestaurantTests) postMenu403(t *testing.T) {
	restaurantId := "a224a8d6-3f9e-4b11-9900-e81a25d80702"

	r := createRequestBody(POST, "/v1/restaurant/"+restaurantId+"/menu", rt.userToken, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting today's restaurant menu with an unknown restaurantId")
	tests.LogInfof(t, 0, "When using the new restaurant %s.", restaurantId)
	tests.AssertStatusCode(t, http.StatusForbidden, w.Code)
}

func (rt *RestaurantTests) postMenu400(t *testing.T) {
	restaurantId := "a224a8d6-3f9e-4b11-9900-e81a25d80702"

	r := createRequestBody(POST, "/v1/restaurant/"+restaurantId+"/menu", rt.adminToken, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	rt.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting today's restaurant menu with an unknown restaurantId")
	tests.LogInfof(t, 0, "When using the new restaurant %s.", restaurantId)
	tests.AssertStatusCode(t, http.StatusNotFound, w.Code)

	recv := w.Body.String()
	resp := "Restaurant not found"
	success := false
	if !strings.Contains(recv, resp) {
		t.Log("Got ", recv)
		t.Log("Want:", resp)
		success = true
	}
	tests.LogStatus(t, success, "Should get the expected result.")
}
