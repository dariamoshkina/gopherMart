package worker

import (
	"context"
	"errors"
	"math"
	"sync"
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
	orders      OrderRepo
	client      AccrualClient
	interval    time.Duration
	workerCount int
	logger      *zap.Logger

	mu          sync.Mutex
	pausedUntil time.Time
}

func New(orders OrderRepo, client AccrualClient, interval time.Duration, workerCount int, logger *zap.Logger) *Poller {
	return &Poller{
		orders:      orders,
		client:      client,
		interval:    interval,
		workerCount: workerCount,
		logger:      logger,
	}
}

func (p *Poller) pauseFor(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if until := time.Now().Add(d); until.After(p.pausedUntil) {
		p.pausedUntil = until
	}
}

func (p *Poller) waitIfPaused(ctx context.Context) {
	p.mu.Lock()
	until := p.pausedUntil
	p.mu.Unlock()

	d := time.Until(until)
	if d <= 0 {
		return
	}
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

func (p *Poller) Run(ctx context.Context) {
	jobs := make(chan *model.Order, p.workerCount)

	var wg sync.WaitGroup
	for range p.workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for order := range jobs {
				if ctx.Err() != nil {
					return
				}
				p.waitIfPaused(ctx)
				if err := p.processOrder(ctx, order); err != nil {
					if rl, ok := errors.AsType[*accrual.RateLimitError](err); ok {
						p.logger.Info("rate limited, backing off",
							zap.Duration("retry_after", rl.RetryAfter))
						p.pauseFor(rl.RetryAfter)
						continue
					}
					p.logger.Warn("process order",
						zap.String("order", order.OrderNumber),
						zap.Error(err))
				}
			}
		}()
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case <-ticker.C:
			p.poll(ctx, jobs)
		}
	}
}

func (p *Poller) poll(ctx context.Context, jobs chan<- *model.Order) {
	pending, err := p.orders.GetPending(ctx, 100)
	if err != nil {
		p.logger.Error("fetch pending orders", zap.Error(err))
		return
	}

	for _, order := range pending {
		select {
		case jobs <- order:
		case <-ctx.Done():
			return
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
			accrualKopecks = new(int64(math.Round(*result.Accrual * 100)))
		}
		if err = p.orders.MarkProcessedWithCredit(ctx, order.ID, order.UserID, accrualKopecks); err != nil {
			return err
		}
		p.logger.Info("order processed",
			zap.String("order", order.OrderNumber),
			zap.Int64p("accrual_kopecks", accrualKopecks))
	}
	return nil
}
