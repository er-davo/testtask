//go:build integration
// +build integration

package repository_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"subscriptionsservice/internal/database"
	"subscriptionsservice/internal/models"
	"subscriptionsservice/internal/repository"
	"subscriptionsservice/internal/retry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var db *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		log.Fatal(err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	db, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := database.Migrate("../../../migrations", dsn); err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	db.Close()
	_ = pgContainer.Terminate(ctx)

	os.Exit(code)
}

func TestSubscriptionsRepo_CRUD(t *testing.T) {
	repo := repository.NewSubscriptionsRepo(db, retry.NoRetry())

	subs := &models.Subscription{
		ServiceName: "Netflix",
		Price:       15,
		UserID:      uuid.New(),
		StartDate:   models.MonthDate{Time: time.Now()},
		EndDate:     nil,
	}

	tx, err := db.Begin(t.Context())
	assert.NoError(t, err)
	defer tx.Rollback(t.Context())

	t.Run("Create", func(t *testing.T) {
		err := repo.CreateSubscription(t.Context(), subs, repository.WithTx(tx))
		assert.NoError(t, err)
		assert.NotZero(t, subs.ID)
	})

	t.Run("GetByID", func(t *testing.T) {
		got, err := repo.GetByID(t.Context(), subs.ID, repository.WithTx(tx))
		assert.NoError(t, err)
		assert.Equal(t, subs.ServiceName, got.ServiceName)
	})

	t.Run("Update", func(t *testing.T) {
		subs.Price = 20
		err := repo.Update(t.Context(), subs, repository.WithTx(tx))
		assert.NoError(t, err)

		got, err := repo.GetByID(t.Context(), subs.ID, repository.WithTx(tx))
		assert.NoError(t, err)
		assert.Equal(t, subs.Price, got.Price)
	})

	t.Run("List with pagination", func(t *testing.T) {
		// создаём еще одну подписку
		another := &models.Subscription{
			ServiceName: "Spotify",
			Price:       10,
			UserID:      subs.UserID,
			StartDate:   models.MonthDate{Time: time.Now()},
		}
		assert.NoError(t, repo.CreateSubscription(t.Context(), another, repository.WithTx(tx)))

		all, err := repo.List(t.Context(), 10, 0, repository.WithTx(tx))
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(all), 2)
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(t.Context(), subs.ID, repository.WithTx(tx))
		assert.NoError(t, err)

		_, err = repo.GetByID(t.Context(), subs.ID, repository.WithTx(tx))
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestSubscriptionsRepo_Summary(t *testing.T) {
	repo := repository.NewSubscriptionsRepo(db, retry.NoRetry())

	tx, err := db.Begin(t.Context())
	assert.NoError(t, err)
	defer tx.Rollback(t.Context())

	user1 := uuid.New()
	user2 := uuid.New()

	// Подписки:
	// Netflix (user1): с 1 января по 31 марта — 2? мес × 20 = 40
	// Spotify (user1): с 15 февраля по 15 апреля — 2 мес × 10 = 20
	// Hulu (user2): с 1 февраля без конца — в диапазоне тоже 2 мес × 15 = 30
	layout := "2006-01-02"
	parse := func(s string) time.Time {
		tm, _ := time.Parse(layout, s)
		return tm
	}

	subs := []*models.Subscription{
		{
			ServiceName: "Netflix",
			Price:       20,
			UserID:      user1,
			StartDate:   models.MonthDate{Time: parse("2025-01-01")},
			EndDate:     &models.MonthDate{Time: parse("2025-03-31")},
		},
		{
			ServiceName: "Spotify",
			Price:       10,
			UserID:      user1,
			StartDate:   models.MonthDate{Time: parse("2025-02-15")},
			EndDate:     &models.MonthDate{Time: parse("2025-04-15")},
		},
		{
			ServiceName: "Hulu",
			Price:       15,
			UserID:      user2,
			StartDate:   models.MonthDate{Time: parse("2025-02-01")},
			EndDate:     nil,
		},
	}

	for _, s := range subs {
		assert.NoError(t, repo.CreateSubscription(t.Context(), s, repository.WithTx(tx)))
	}

	from := models.MonthDate{Time: parse("2025-02-01")}
	to := models.MonthDate{Time: parse("2025-03-31")}

	tests := []struct {
		name        string
		userID      *uuid.UUID
		serviceName *string
		expectedSum int
	}{
		{
			name:        "All subscriptions",
			userID:      nil,
			serviceName: nil,
			expectedSum: 90, // 40 + 20 + 30
		},
		{
			name:        "Filter by UserID=user1",
			userID:      &user1,
			serviceName: nil,
			expectedSum: 60, // Netflix 40 + Spotify 20
		},
		{
			name:        "Filter by ServiceName='Hulu'",
			userID:      nil,
			serviceName: ptrString("Hulu"),
			expectedSum: 30,
		},
		{
			name:        "Filter by UserID=user1 and ServiceName='Spotify'",
			userID:      &user1,
			serviceName: ptrString("Spotify"),
			expectedSum: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &models.SummaryRequest{
				From:        from,
				To:          to,
				UserID:      ptrUUIDToString(tt.userID),
				ServiceName: tt.serviceName,
			}

			sum, err := repo.Summary(t.Context(), req, repository.WithTx(tx))
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSum, sum)
		})
	}
}

func ptrString(s string) *string { return &s }
func ptrUUIDToString(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}
