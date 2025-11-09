package repository

import (
	"context"
	"time"

	"subscriptionsservice/internal/models"
	"subscriptionsservice/internal/retry"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executer allows query execution by both Pool and Tx.
type Executer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// RepositoryOptions contains options for repository. (Ececuter)
type RepositoryOptions struct {
	exec Executer
}

// Option is a function that configures RepositoryOptions.
type Option func(*RepositoryOptions)

// WithTx configures RepositoryOptions with a transaction.
func WithTx(tx pgx.Tx) Option {
	return func(o *RepositoryOptions) {
		o.exec = tx
	}
}

// defaultOptions returns default options (pool).
func defaultOptions(repo *SubscriptionsRepo) RepositoryOptions {
	return RepositoryOptions{
		exec: repo.db,
	}
}

// SubscriptionsRepo provides CRUD and summary operations.
type SubscriptionsRepo struct {
	db    *pgxpool.Pool
	retry retry.Retrier
	psql  sq.StatementBuilderType
}

// NewSubscriptionsRepo initializes SubscriptionsRepo with Squirrel.
func NewSubscriptionsRepo(db *pgxpool.Pool, r retry.Retrier) *SubscriptionsRepo {
	return &SubscriptionsRepo{
		db:    db,
		retry: r,
		psql:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// CreateSubscription inserts a new record.
func (r *SubscriptionsRepo) CreateSubscription(ctx context.Context, subs *models.Subscription, opts ...Option) error {
	opt := r.applyOptions(opts...)

	return r.retry.Do(ctx, func() error {
		var endDate interface{}
		if subs.EndDate != nil {
			// передаём только дату без времени
			endDate = subs.EndDate.Time.Format("2006-01-02")
		} else {
			endDate = nil
		}

		query := r.psql.Insert("subscriptions").
			Columns(
				"service_name", "price", "user_id",
				"start_date", "end_date",
			).Values(
			subs.ServiceName, subs.Price, subs.UserID,
			subs.StartDate.Time.Format("2006-01-02"),
			endDate,
		).Suffix("RETURNING id")

		sql, args, err := query.ToSql()
		if err != nil {
			return err
		}

		return wrapDBError(opt.exec.QueryRow(ctx, sql, args...).Scan(&subs.ID))
	})
}

// GetByID retrieves a subscription by ID.
func (r *SubscriptionsRepo) GetByID(ctx context.Context, id int64, opts ...Option) (*models.Subscription, error) {
	opt := r.applyOptions(opts...)

	var sub models.Subscription
	var retryErr error

	if err := r.retry.Do(ctx, func() error {
		query := r.psql.Select(
			"id", "service_name", "price",
			"user_id", "start_date", "end_date",
		).From("subscriptions").
			Where(sq.Eq{"id": id})

		sql, args, err := query.ToSql()
		if err != nil {
			return err
		}

		var startDate time.Time
		var endDate *time.Time
		err = opt.exec.QueryRow(ctx, sql, args...).Scan(
			&sub.ID, &sub.ServiceName, &sub.Price,
			&sub.UserID, &startDate, &endDate,
		)
		if err != nil {
			return wrapDBError(err)
		}

		sub.StartDate = models.MonthDate{Time: startDate}
		if endDate != nil {
			e := models.MonthDate{Time: *endDate}
			sub.EndDate = &e
		}
		retryErr = nil
		return nil
	}); err != nil {
		return nil, err
	}

	return &sub, retryErr
}

// List returns subscriptions ordered by id with optional pagination.
// If limit == 0 -> no LIMIT applied.
func (r *SubscriptionsRepo) List(ctx context.Context, limit, offset int, opts ...Option) ([]models.Subscription, error) {
	opt := r.applyOptions(opts...)

	var subs []models.Subscription

	if err := r.retry.Do(ctx, func() error {
		builder := r.psql.Select(
			"id", "service_name", "price",
			"user_id", "start_date", "end_date",
		).From("subscriptions").OrderBy("id ASC")

		if limit > 0 {
			builder = builder.Limit(uint64(limit)).Offset(uint64(offset))
		}

		sqlStr, args, err := builder.ToSql()
		if err != nil {
			return err
		}

		rows, err := opt.exec.Query(ctx, sqlStr, args...)
		if err != nil {
			return wrapDBError(err)
		}
		defer rows.Close()

		for rows.Next() {
			var s models.Subscription
			var startDate time.Time
			var endDate *time.Time
			if err := rows.Scan(
				&s.ID, &s.ServiceName, &s.Price,
				&s.UserID, &startDate, &endDate,
			); err != nil {
				return wrapDBError(err)
			}
			s.StartDate = models.MonthDate{Time: startDate}
			if endDate != nil {
				e := models.MonthDate{Time: *endDate}
				s.EndDate = &e
			}
			subs = append(subs, s)
		}
		return wrapDBError(rows.Err())
	}); err != nil {
		return nil, err
	}

	return subs, nil
}

// Update modifies an existing record.
func (r *SubscriptionsRepo) Update(ctx context.Context, subs *models.Subscription, opts ...Option) error {
	opt := r.applyOptions(opts...)

	return r.retry.Do(ctx, func() error {
		var endDate interface{}
		if subs.EndDate != nil {
			endDate = subs.EndDate.Time.Format("2006-01-02")
		} else {
			endDate = nil
		}

		query := r.psql.Update("subscriptions").
			Set("service_name", subs.ServiceName).
			Set("price", subs.Price).
			Set("user_id", subs.UserID).
			Set("start_date", subs.StartDate.Time.Format("2006-01-02")).
			Set("end_date", endDate).
			Where(sq.Eq{"id": subs.ID})

		sql, args, err := query.ToSql()
		if err != nil {
			return err
		}

		cmd, err := opt.exec.Exec(ctx, sql, args...)
		if err != nil {
			return wrapDBError(err)
		}
		if cmd.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Delete removes a record by ID.
func (r *SubscriptionsRepo) Delete(ctx context.Context, id int64, opts ...Option) error {
	opt := r.applyOptions(opts...)

	return r.retry.Do(ctx, func() error {
		query := r.psql.Delete("subscriptions").Where(sq.Eq{"id": id})
		sql, args, err := query.ToSql()
		if err != nil {
			return err
		}

		cmd, err := opt.exec.Exec(ctx, sql, args...)
		if err != nil {
			return wrapDBError(err)
		}
		if cmd.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Summary calculates total price taking into account months of overlap between
// subscription period and the requested [From, To] range.
// For each subscription we compute number of months in the intersection (inclusive),
// then add price * months to total.
func (r *SubscriptionsRepo) Summary(ctx context.Context, q *models.SummaryRequest, opts ...Option) (int, error) {
	opt := r.applyOptions(opts...)

	var total int

	if err := r.retry.Do(ctx, func() error {
		// select fields needed to compute overlap: price, start_date, end_date
		builder := r.psql.Select("price", "start_date", "end_date").
			From("subscriptions").
			Where(sq.LtOrEq{"start_date": q.To.Time}). // start_date <= to
			Where(sq.Or{
				sq.GtOrEq{"end_date": q.From.Time}, // end_date >= from
				sq.Expr("end_date IS NULL"),
			})

		if q.UserID != nil {
			builder = builder.Where(sq.Eq{"user_id": *q.UserID})
		}
		if q.ServiceName != nil {
			builder = builder.Where(sq.Eq{"service_name": *q.ServiceName})
		}

		sqlStr, args, err := builder.ToSql()
		if err != nil {
			return err
		}

		rows, err := opt.exec.Query(ctx, sqlStr, args...)
		if err != nil {
			return wrapDBError(err)
		}
		defer rows.Close()

		var (
			price     int
			startDate time.Time
			endDate   *time.Time
		)

		for rows.Next() {
			if err := rows.Scan(&price, &startDate, &endDate); err != nil {
				return wrapDBError(err)
			}

			// compute overlap interval [ovStart, ovEnd]
			ovStart := startDate
			if q.From.Time.After(ovStart) {
				ovStart = q.From.Time
			}

			ovEnd := q.To.Time
			if endDate != nil && endDate.Before(ovEnd) {
				ovEnd = *endDate
			}

			// if no overlap (ovEnd < ovStart) skip
			if ovEnd.Before(ovStart) {
				continue
			}

			months := monthsInclusive(ovStart, ovEnd)
			total += price * months
		}

		if err := rows.Err(); err != nil {
			return wrapDBError(err)
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return total, nil
}

func (r *SubscriptionsRepo) applyOptions(opts ...Option) *RepositoryOptions {
	opt := defaultOptions(r)
	for _, o := range opts {
		if o != nil {
			o(&opt)
		}
	}
	return &opt
}
