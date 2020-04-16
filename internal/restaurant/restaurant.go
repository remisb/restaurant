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

// Predefined errors identify expected failure conditions.
var (
	// ErrNotFound is used when a specific Restaurant is requested but does not exist.
	ErrNotFound = errors.New("Restaurant not found")

	// ErrInvalidID is used when an invalid UUID is provided.
	ErrInvalidID = errors.New("ID is not in its proper form")

	// ErrForbidden occurs when a user tries to do something that is forbidden to
	// them according to our access control policies.
	ErrForbidden = errors.New("Attempted action is not allowed")
)

func List(ctx context.Context, db *sqlx.DB) ([]Restaurant, error) {
	ctx, span := trace.StartSpan(ctx, "internal.restaurant.List")
	defer span.End()

	restaurants := []Restaurant{}
	const q = `SELECT * FROM restaurant`
	if err := db.SelectContext(ctx, &restaurants, q); err != nil {
		return nil, errors.Wrap(err, "selecting restaurants")
	}
	return restaurants, nil
}

func Create(ctx context.Context, db *sqlx.DB, user auth.Claims, nr NewRestaurant, now time.Time) (*Restaurant, error) {
	ctx, span := trace.StartSpan(ctx, "internal.restaurant.Create")
	defer span.End()

	currentTime := now.UTC()
	r := Restaurant{
		ID:          uuid.New().String(),
		Name:        nr.Name,
		Address:     nr.Address,
		OwnerUserID: user.Subject,
		DateCreated: currentTime,
		DateUpdated:  currentTime,
	}

	const q = `INSERT INTO restaurant
	    (restaurant_id, name, address, owner_user_id, date_created, date_updated)
	    VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := db.ExecContext(ctx, q, r.ID, r.Name, r.Address, r.OwnerUserID, r.DateCreated, r.DateUpdated)
	if err != nil {
		return nil, errors.Wrap(err, "inserting restaurant")
	}

	return &r, nil
}


// Retrieve finds the restaurant identified by a given ID.
func Retrieve(ctx context.Context, db *sqlx.DB, id string) (*Restaurant, error) {
	ctx, span := trace.StartSpan(ctx, "internal.restaurant.Retrieve")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidID
	}

	var r Restaurant

	const q = `SELECT r.* FROM restaurant AS r WHERE r.restaurant_id = $1`

	if err := db.GetContext(ctx, &r, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "selecting single restaurant")
	}

	return &r, nil
}

// Update modifies data about a Restaurant. It will error if the specified ID is
// invalid or does not reference an existing Restaurant.
func Update(ctx context.Context, db *sqlx.DB, user auth.Claims, id string, update UpdateRestaurant, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.restaurant.Update")
	defer span.End()

	r, err := Retrieve(ctx, db, id)
	if err != nil {
		return err
	}

	// If you do not have the admin role ...
	// and you are not the owner of this product ...
	// then get outta here!
	if !user.HasRole(auth.RoleAdmin) && r.OwnerUserID != user.Subject {
		return ErrForbidden
	}

	if update.Name != nil {
		r.Name = *update.Name
	}
	if update.Address != nil {
		r.Address = *update.Address
	}
	r.DateUpdated = now

	const q = `UPDATE restaurant SET
		"name" = $2,
		"address" = $3,
		"date_updated" = $4
		WHERE restaurant_id = $1`
	_, err = db.ExecContext(ctx, q, id,
		r.Name, r.Address, r.DateUpdated,
	)
	if err != nil {
		return errors.Wrap(err, "updating restaurant")
	}

	return nil
}

// Delete removes the product identified by a given ID.
func Delete(ctx context.Context, db *sqlx.DB, id string) error {
	ctx, span := trace.StartSpan(ctx, "internal.restaurant.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidID
	}

	const q = `DELETE FROM restaurant WHERE restaurant_id = $1`

	if _, err := db.ExecContext(ctx, q, id); err != nil {
		return errors.Wrapf(err, "deleting restaurant %s", id)
	}

	return nil
}
