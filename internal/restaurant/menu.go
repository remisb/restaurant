package restaurant

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/remisb/restaurant/internal/platform/auth"
	"go.opencensus.io/trace"
	"time"
)

func CreateMenu(ctx context.Context, db *sqlx.DB, user auth.Claims, nm NewMenu, now time.Time) (*Menu, error) {
	ctx, span := trace.StartSpan(ctx, "internal.Restaurant.CreateMenu")
	defer span.End()


	currentTime := now.UTC()
	m := Menu{
		ID: uuid.New().String(),
		RestaurantID: nm.RestaurantID,
		Date: currentTime,
		Menu: nm.Menu,
	}

	const q = `INSERT INTO menu 
	  (menu_id, restaurant_id, date, menu, votes)
	  VALUES ($1, $2, $3, $4, $5)`

	_, err := db.ExecContext(ctx, q, m.ID, m.RestaurantID, m.Date, m.Menu, 0)
	if err != nil {
		return nil, errors.Wrap(err, "inserting menu")
	}
	return &m, nil
}

// Retrieve finds the restaurant identified by a given ID.
func MenuRetrieve(ctx context.Context, db *sqlx.DB, id string) (*Menu, error) {
	ctx, span := trace.StartSpan(ctx, "internal.restaurant.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidID
	}

	var m Menu

	const q = `SELECT * FROM menu AS r WHERE menu_id =  $1`

	if err := db.GetContext(ctx, &m, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "selecting single menu")
	}

	return &m, nil
}

func MenuUpdate(ctx context.Context, db *sqlx.DB, user auth.Claims, restaurantId string, update UpdateMenu, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.Restaurant.MenuUpdate")
	defer span.End()

	r, err := Retrieve(ctx, db, restaurantId)
	if err != nil {
		return err
	}

	if r.OwnerUserID != user.Subject {
		return ErrForbidden
	}

	m, err := MenuRetrieve(ctx, db, update.ID)
	if err != nil {
		return err
	}

	if update.Menu != "" {
		m.Menu = update.Menu
		m.Date = update.Date
	}

	const q = `UPDATE menu SET
		"menu" = $2,
		"date" = $3
		WHERE menu_id =  $1`

	_, err = db.ExecContext(ctx, q, update.ID, update.Menu, update.Date)
	if err != nil {
		return errors.Wrap(err, "updating menu")
	}

	return nil
}
