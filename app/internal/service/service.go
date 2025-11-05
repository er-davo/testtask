package service

import (
	"context"
	"fmt"
	"subscriptionsservice/internal/models"
	"subscriptionsservice/internal/repository"

	"go.uber.org/zap"
)

// SubscriptionRepo defines repository methods required by SubscriptionService.
type SubscriptionRepo interface {
	// CreateSubscription inserts a new subscription record.
	CreateSubscription(ctx context.Context, s *models.Subscription, opts ...repository.Option) error

	// GetByID returns a subscription by its ID.
	GetByID(ctx context.Context, id int64, opts ...repository.Option) (*models.Subscription, error)

	// List returns all subscriptions.
	List(ctx context.Context, opts ...repository.Option) ([]models.Subscription, error)

	// Update modifies an existing subscription.
	Update(ctx context.Context, s *models.Subscription, opts ...repository.Option) error

	// Delete removes a subscription by ID.
	Delete(ctx context.Context, id int64, opts ...repository.Option) error

	// Summary returns the sum of subscription prices matching the query.
	Summary(ctx context.Context, q *models.SummaryRequest, opts ...repository.Option) (int, error)
}

// SubscriptionService provides business logic for managing subscriptions.
type SubscriptionService struct {
	repo SubscriptionRepo
	log  *zap.Logger
}

// NewSubscriptionService creates a new instance of SubscriptionService.
func NewSubscriptionService(repo SubscriptionRepo, log *zap.Logger) *SubscriptionService {
	return &SubscriptionService{
		repo: repo,
		log:  log,
	}
}

// CreateSubscription adds a new subscription to the repository.
func (s *SubscriptionService) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	s.log.Info("creating subscription", zap.String("service_name", sub.ServiceName))
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		s.log.Error("failed to create subscription", zap.Error(err))
		return err
	}
	s.log.Info("subscription created", zap.Int64("id", sub.ID))
	return nil
}

// GetByID retrieves a subscription by its ID.
func (s *SubscriptionService) GetByID(ctx context.Context, id int64) (*models.Subscription, error) {
	s.log.Info("getting subscription by id", zap.Int64("id", id))
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get subscription", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	return sub, nil
}

// List returns all subscriptions.
func (s *SubscriptionService) List(ctx context.Context) ([]models.Subscription, error) {
	s.log.Info("listing subscriptions")
	subs, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error("failed to list subscriptions", zap.Error(err))
		return nil, err
	}
	return subs, nil
}

// Update modifies an existing subscription.
func (s *SubscriptionService) Update(ctx context.Context, sub *models.Subscription) error {
	s.log.Info("updating subscription", zap.Int64("id", sub.ID))
	if err := s.repo.Update(ctx, sub); err != nil {
		s.log.Error("failed to update subscription", zap.Int64("id", sub.ID), zap.Error(err))
		return err
	}
	s.log.Info("subscription updated", zap.Int64("id", sub.ID))
	return nil
}

// Delete removes a subscription by its ID.
func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	s.log.Info("deleting subscription", zap.Int64("id", id))
	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error("failed to delete subscription", zap.Int64("id", id), zap.Error(err))
		return err
	}
	s.log.Info("subscription deleted", zap.Int64("id", id))
	return nil
}

// Summary calculates total subscription price within a time range and optional filters.
func (s *SubscriptionService) Summary(ctx context.Context, req *models.SummaryRequest) (int, error) {
	s.log.Info("calculating subscription summary",
		zap.Time("from", req.From.Time),
		zap.Time("to", req.To.Time),
	)
	total, err := s.repo.Summary(ctx, req)
	if err != nil {
		s.log.Error("failed to calculate summary", zap.Error(err))
		return 0, fmt.Errorf("summary failed: %w", err)
	}
	s.log.Info("subscription summary calculated", zap.Int("total", total))
	return total, nil
}
