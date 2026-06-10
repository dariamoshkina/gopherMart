package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

type BalanceRepository struct {
	pool *pgxpool.Pool
}

func NewBalanceRepository(pool *pgxpool.Pool) *BalanceRepository {
	return &BalanceRepository{pool: pool}
}

func (r *BalanceRepository) GetByUserID(ctx context.Context, userID int64) (*model.Balance, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT user_id, current, withdrawn FROM balance WHERE user_id = $1`,
		userID,
	)
	var balance model.Balance
	err := row.Scan(&balance.UserID, &balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

func (r *BalanceRepository) GetOrCreate(ctx context.Context, userID int64) (*model.Balance, error) {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, 0, 0) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return r.GetByUserID(ctx, userID)
}

func (r *BalanceRepository) Credit(ctx context.Context, userID int64, amount int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, $2, 0)
		 ON CONFLICT (user_id) DO UPDATE SET current = balance.current + EXCLUDED.current`,
		userID, amount,
	)
	return err
}

func (r *BalanceRepository) Withdraw(ctx context.Context, userID int64, orderNumber string, sum int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE balance SET current = current - $1, withdrawn = withdrawn + $1
		 WHERE user_id = $2 AND current >= $1`,
		sum, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return service.ErrInsufficientBalance
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`,
		userID, orderNumber, sum,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *BalanceRepository) ListWithdrawalsByUserID(ctx context.Context, userID int64) ([]*model.Withdrawal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, order_number, sum, processed_at
		 FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.Withdrawal
	for rows.Next() {
		var withdrawal model.Withdrawal
		if err := rows.Scan(&withdrawal.ID, &withdrawal.UserID, &withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return nil, err
		}
		list = append(list, &withdrawal)
	}
	return list, rows.Err()
}
