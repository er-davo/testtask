package repository

import (
	"context"
	"strconv"
	"strings"
	"subscriptionsservice/internal/models"
	"subscriptionsservice/internal/retry"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executer defines a common interface for executing SQL queries.
// It is implemented by pgxpool.Pool and pgx.Tx to allow transactional reuse.
type Executer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// SubscriptionsRepo provides CRUD operations and summary queries
// for the subscriptions table.
type SubscriptionsRepo struct {
	db    *pgxpool.Pool
	retry retry.Retrier
}

// NewSubscriptionsRepo returns a new instance of SubscriptionsRepo.
func NewSubscriptionsRepo(db *pgxpool.Pool, r retry.Retrier) *SubscriptionsRepo {
	return &SubscriptionsRepo{
		db:    db,
		retry: r,
	}
}

// CreateSubscription inserts a new subscription record into the database.
func (r *SubscriptionsRepo) CreateSubscription(ctx context.Context, subs *models.Subscription) error {
	return r.retry.Do(ctx, func() error {
		return createSubscription(ctx, r.db, subs)
	})
}

// GetByID retrieves a single subscription by its ID.
func (r *SubscriptionsRepo) GetByID(ctx context.Context, id int64) (*models.Subscription, error) {
	sub := &models.Subscription{}
	var retryErr error

	if err := r.retry.Do(ctx, func() error {
		sub, retryErr = getByID(ctx, r.db, id)
		return retryErr
	}); err != nil {
		return nil, err
	}

	return sub, nil
}

// List returns all subscriptions ordered by ID.
func (r *SubscriptionsRepo) List(ctx context.Context) ([]models.Subscription, error) {
	subs := []models.Subscription{}
	var retryErr error

	if err := r.retry.Do(ctx, func() error {
		subs, retryErr = list(ctx, r.db)
		return retryErr
	}); err != nil {
		return nil, err
	}

	return subs, nil
}

// Update modifies an existing subscription record.
func (r *SubscriptionsRepo) Update(ctx context.Context, subs *models.Subscription) error {
	return r.retry.Do(ctx, func() error {
		return update(ctx, r.db, subs)
	})
}

// Delete removes a subscription by ID.
func (r *SubscriptionsRepo) Delete(ctx context.Context, id int64) error {
	return r.retry.Do(ctx, func() error {
		return delete(ctx, r.db, id)
	})
}

// Summary calculates the total subscription cost for a given period.
// Optional filters by user or service name can be applied.
func (r *SubscriptionsRepo) Summary(ctx context.Context, q *models.SummaryRequest) (int, error) {
	var (
		sum      int
		retryErr error
	)

	if err := r.retry.Do(ctx, func() error {
		sum, retryErr = summary(ctx, r.db, q)
		return retryErr
	}); err != nil {
		return 0, err
	}

	return sum, nil
}

// createSubscription performs a single INSERT query for a subscription.
func createSubscription(ctx context.Context, exec Executer, subs *models.Subscription) error {
	if subs == nil {
		return ErrNilValue
	}

	const query = `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	var endDate interface{}
	if subs.EndDate != nil {
		endDate = subs.EndDate.Time
	}

	return wrapDBError(exec.QueryRow(ctx, query,
		subs.ServiceName, subs.Price, subs.UserID,
		subs.StartDate.Time, endDate,
	).Scan(&subs.ID))
}

// getByID performs a SELECT query for a single subscription by ID.
func getByID(ctx context.Context, exec Executer, id int64) (*models.Subscription, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}

	const query = `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions WHERE id = $1;
	`

	var (
		s         models.Subscription
		startDate time.Time
		endDate   *time.Time
	)

	if err := exec.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.ServiceName, &s.Price, &s.UserID, &startDate, &endDate,
	); err != nil {
		return nil, wrapDBError(err)
	}

	s.StartDate = models.MonthDate{Time: startDate}

	if endDate != nil {
		e := models.MonthDate{Time: *endDate}
		s.EndDate = &e
	} else {
		s.EndDate = nil
	}

	return &s, nil
}

// list returns all subscription rows.
func list(ctx context.Context, exec Executer) ([]models.Subscription, error) {
	const query = `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions ORDER BY id;
	`

	rows, err := exec.Query(ctx, query)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var (
		res       []models.Subscription
		startDate time.Time
		endDate   *time.Time
	)

	for rows.Next() {
		var s models.Subscription
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &startDate, &endDate); err != nil {
			return nil, wrapDBError(err)
		}
		s.StartDate = models.MonthDate{Time: startDate}

		if endDate != nil {
			e := models.MonthDate{Time: *endDate}
			s.EndDate = &e
		} else {
			s.EndDate = nil
		}

		res = append(res, s)
	}

	return res, wrapDBError(rows.Err())
}

// update modifies an existing subscription.
func update(ctx context.Context, exec Executer, subs *models.Subscription) error {
	if subs == nil {
		return ErrNilValue
	}

	const query = `
		UPDATE subscriptions
		SET service_name = $1, price = $2, user_id = $3,
		    start_date = $4, end_date = $5
		WHERE id = $6;
	`

	var endDate interface{}
	if subs.EndDate != nil {
		endDate = subs.EndDate.Time
	}

	cmd, err := exec.Exec(ctx, query,
		subs.ServiceName, subs.Price, subs.UserID,
		subs.StartDate.Time, endDate, subs.ID,
	)
	if err != nil {
		return wrapDBError(err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// delete removes a subscription record by ID.
func delete(ctx context.Context, exec Executer, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}

	const query = `DELETE FROM subscriptions WHERE id = $1;`

	cmd, err := exec.Exec(ctx, query, id)
	if err != nil {
		return wrapDBError(err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// summary calculates the total price of subscriptions
// active between the given MonthDate range.
func summary(ctx context.Context, exec Executer, q *models.SummaryRequest) (int, error) {
	var args []any
	var cond []string

	cond = append(cond, "start_date >= $1")
	args = append(args, q.From.Time)

	cond = append(cond, "(end_date <= $2 OR end_date IS NULL)")
	args = append(args, q.To.Time)

	idx := 3
	if q.UserID != nil {
		cond = append(cond, "user_id = $"+strconv.Itoa(idx))
		args = append(args, *q.UserID)
		idx++
	}
	if q.ServiceName != nil {
		cond = append(cond, "service_name = $"+strconv.Itoa(idx))
		args = append(args, *q.ServiceName)
	}

	query := `
		SELECT COALESCE(SUM(price), 0)
		FROM subscriptions
		WHERE ` + strings.Join(cond, " AND ")

	var total int
	if err := exec.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, wrapDBError(err)
	}

	return total, nil
}
