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

type Menu struct {
	db *sqlx.DB
}

// List gets all existing restaurants in the system.
func (m *Menu) List(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Menu.List")
	defer span.End()

	restaurants, err := restaurant.List(ctx, m.db)
	if err != nil {
		return err
	}

	return web.Respond(ctx, w, restaurants, http.StatusOK)
}

func (m *Menu) RetrieveMenu(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Menu.Retrieve")
	defer span.End()

	menuRetrieved, err := restaurant.MenuRetrieve(ctx, m.db, params["restaurantId"])
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

	return web.Respond(ctx, w, menuRetrieved, http.StatusOK)
}

func (m *Menu) RetrieveVotes(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Menu.Retrieve")
	defer span.End()

	menuRetrieved, err := restaurant.MenuRetrieve(ctx, m.db, params["restaurantId"])
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

	return web.Respond(ctx, w, menuRetrieved, http.StatusOK)
}

func (m *Menu) CreateMenu(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Menu.CreateMenu")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return web.NewShutdownError("claims missing from context")
	}

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	restaurantId := params["restaurantId"]

	var nm restaurant.NewMenu
	if err := web.Decode(r, &nm); err != nil {
		return errors.Wrap(err, "decoding new menu")
	}

	if nm.RestaurantID != restaurantId {
		return restaurant.ErrInvalidID
	}

	restaurantRes, err := restaurant.Retrieve(ctx, m.db, restaurantId)
	if err != nil {
		switch err {
		case restaurant.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case restaurant.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case restaurant.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "retrieving restaurant id: %s", restaurantId)
		}
	}

	if restaurantRes.OwnerUserID != claims.Subject {
		return web.NewRequestError(err, http.StatusForbidden)
	}

	restResult, err := restaurant.CreateMenu(ctx, m.db, claims, nm, v.Now)
	if err != nil {
		return errors.Wrapf(err, "creating new menu: %+v", nm)
	}

	if restaurantRes == nil {
		return restaurant.ErrNotFound
	}
	return web.Respond(ctx, w, restResult, http.StatusCreated)
}

func (m *Menu) Update(ctx context.Context, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Menu.Update")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return web.NewShutdownError("claims missing from context")
	}

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	var up restaurant.UpdateMenu
	if err := web.Decode(r, &up); err != nil {
		return errors.Wrap(err, "request decode")
	}

	if err := restaurant.MenuUpdate(ctx, m.db, claims, params["restaurantId"], up, v.Now); err != nil {
		switch err {
		case restaurant.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case restaurant.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case restaurant.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "updating menu %q: %+v", params["restaurantId"], up)
		}
	}
	return nil
}
