package handlers

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/remisb/restaurant/internal/platform/auth"
	"github.com/remisb/restaurant/internal/platform/web"
	"github.com/remisb/restaurant/internal/restaurant"
	"go.opencensus.io/trace"
	"net/http"
)

type Restaurant struct {
	db *sqlx.DB
}

// List gets all existing restaurants in the system.
func (res *Restaurant) List(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Restaurant.List")
	defer span.End()

	restaurants, err := restaurant.List(ctx, res.db)
	if err != nil {
		return err
	}

	return web.Respond(ctx, w, restaurants, http.StatusOK)
}

func (res *Restaurant) Retrieve(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Restaurant.Retrieve")
	defer span.End()

	restRetrieved, err := restaurant.Retrieve(ctx, res.db, params["id"])
	if err != nil {
		switch err {
		case restaurant.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case restaurant.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return errors.Wrapf(err, "ID: %s", params["id"])
		}
	}

	return web.Respond(ctx, w, restRetrieved, http.StatusOK)
}

func (res *Restaurant) Create(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Restaurant.Create")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return web.NewShutdownError("claims missing from context")
	}

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	var nr restaurant.NewRestaurant
	if err := web.Decode(r, &nr); err != nil {
		return errors.Wrap(err, "decoding new restaurant")
	}

	restResult, err := restaurant.Create(ctx, res.db, claims, nr, v.Now)
	if err != nil {
		return errors.Wrapf(err, "creating new restaurant: %+v", nr)
	}

	return web.Respond(ctx, w, restResult, http.StatusCreated)
}

// Update decodes the body of a request to update an existing restaurant. The ID
// of the restaurant is part of the request URL.
func (res *Restaurant) Update(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Restaurant.Update")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return web.NewShutdownError("claims missing from context")
	}

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	var up restaurant.UpdateRestaurant
	if err := web.Decode(r, &up); err != nil {
		return errors.Wrap(err, "")
	}

	if err := restaurant.Update(ctx, res.db, claims, params["id"], up, v.Now); err != nil {
		switch err {
		case restaurant.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case restaurant.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case restaurant.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "updating restaurant %q: %+v", params["id"], up)
		}
	}

	return web.Respond(ctx, w, nil, http.StatusNoContent)
}

// Delete removes a single restaurant identified by an ID in the request URL.
func (res *Restaurant) Delete(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Restaurant.Delete")
	defer span.End()

	if err := restaurant.Delete(ctx, res.db, params["id"]); err != nil {
		switch err {
		case restaurant.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return errors.Wrapf(err, "Id: %s", params["id"])
		}
	}

	return web.Respond(ctx, w, nil, http.StatusNoContent)
}
