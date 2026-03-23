package worker

import (
	"context"
	"errors"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/dariamoshkina/gopherMart/internal/client/accrual"
	"github.com/dariamoshkina/gopherMart/internal/model"
)

type OrderRepo interface {
	GetPending(ctx context.Context, limit int) ([]*model.Order, error)
	UpdateStatus(ctx context.Context, orderID int64, status string, accrual *int64) error
	MarkProcessedWithCredit(ctx context.Context, orderID, userID int64, accrual *int64) error
}

type AccrualClient interface {
	GetOrder(ctx context.Context, orderNumber string) (*accrual.AccrualResult, error)
}

type Poller struct {
	orders   OrderRepo
	client   AccrualClient
	interval time.Duration
	logger   *zap.Logger
}

func New(orders OrderRepo, client AccrualClient, interval time.Duration, logger *zap.Logger) *Poller {
	return &Poller{
		orders:   orders,
		client:   client,
		interval: interval,
		logger:   logger,
	}
}

func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	pending, err := p.orders.GetPending(ctx, 100)
	if err != nil {
		p.logger.Error("fetch pending orders", zap.Error(err))
		return
	}

	for _, order := range pending {
		if ctx.Err() != nil {
			return
		}
		if err := p.processOrder(ctx, order); err != nil {
			var rl *accrual.RateLimitError
			if errors.As(err, &rl) {
				p.logger.Info("rate limited, backing off",
					zap.Duration("retry_after", rl.RetryAfter))
				select {
				case <-time.After(rl.RetryAfter):
				case <-ctx.Done():
				}
				return
			}
			p.logger.Warn("process order",
				zap.String("order", order.OrderNumber),
				zap.Error(err))
		}
	}
}

func (p *Poller) processOrder(ctx context.Context, order *model.Order) error {
	result, err := p.client.GetOrder(ctx, order.OrderNumber)
	if err != nil {
		if errors.Is(err, accrual.ErrNotRegistered) {
			return nil
		}
		return err
	}

	switch result.Status {
	case "REGISTERED":
		return nil
	case "PROCESSING":
		return p.orders.UpdateStatus(ctx, order.ID, model.OrderStatusProcessing, nil)
	case "INVALID":
		return p.orders.UpdateStatus(ctx, order.ID, model.OrderStatusInvalid, nil)
	case "PROCESSED":
		var accrualKopecks *int64
		if result.Accrual != nil {
			v := int64(math.Round(*result.Accrual * 100))
			accrualKopecks = &v
		}
		if err := p.orders.MarkProcessedWithCredit(ctx, order.ID, order.UserID, accrualKopecks); err != nil {
			return err
		}
		p.logger.Info("order processed",
			zap.String("order", order.OrderNumber),
			zap.Int64p("accrual_kopecks", accrualKopecks))
	}
	return nil
}
