package postgres

import (
	"context"
	"errors"

	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, userID int64, orderNumber string) (*model.Order, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO orders (user_id, order_number, status)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, order_number, status, accrual, uploaded_at`,
		userID, orderNumber, model.OrderStatusNew,
	)
	var order model.Order
	err := row.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, service.ErrDuplicateOrderNumber
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) GetByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, order_number, status, accrual, uploaded_at
		 FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.Order
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, err
		}
		list = append(list, &order)
	}
	return list, rows.Err()
}

func (r *OrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, order_number, status, accrual, uploaded_at
		 FROM orders WHERE order_number = $1`,
		orderNumber,
	)
	var order model.Order
	err := row.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FOR UPDATE SKIP LOCKED so multiple instances don't process the same order
func (r *OrderRepository) GetPending(ctx context.Context, limit int) ([]*model.Order, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`SELECT id, user_id, order_number, status, accrual, uploaded_at
		 FROM orders WHERE status IN ('NEW', 'PROCESSING')
		 ORDER BY uploaded_at ASC LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	var list []*model.Order
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, &order)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, tx.Commit(ctx)
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID int64, status string, accrual *int64) error {
	var accrualAmount int64
	if accrual != nil {
		accrualAmount = *accrual
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3`,
		status, accrualAmount, orderID,
	)
	return err
}

func (r *OrderRepository) MarkProcessedWithCredit(ctx context.Context, orderID, userID int64, accrual *int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var accrualAmount int64
	if accrual != nil {
		accrualAmount = *accrual
	}

	result, err := tx.Exec(ctx,
		`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3 AND status != $1`,
		model.OrderStatusProcessed, accrualAmount, orderID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return nil
	}
	if accrualAmount > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, $2, 0)
			 ON CONFLICT (user_id) DO UPDATE SET current = balance.current + EXCLUDED.current`,
			userID, accrualAmount,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
