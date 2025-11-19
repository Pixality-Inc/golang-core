package postgres

import "context"

type CarryFunc = func(ctx context.Context, queryRunner QueryRunner) error

type (
	DaoFunc0                                 = func(ctx context.Context, queryRunner QueryRunner) error
	DaoFunc1[P1 any]                         = func(ctx context.Context, queryRunner QueryRunner, p1 P1) error
	DaoFunc2[P1 any, P2 any]                 = func(ctx context.Context, queryRunner QueryRunner, p1 P1, p2 P2) error
	DaoFunc3[P1 any, P2 any, P3 any]         = func(ctx context.Context, queryRunner QueryRunner, p1 P1, p2 P2, p3 P3) error
	DaoFunc4[P1 any, P2 any, P3 any, P4 any] = func(ctx context.Context, queryRunner QueryRunner, p1 P1, p2 P2, p3 P3, p4 P4) error
)

type (
	TxFunc0                                         = func(ctx context.Context) func(queryRunner QueryRunner) error
	TxFunc1[P1 any]                                 = func(ctx context.Context, p1 P1) func(queryRunner QueryRunner) error
	TxFunc2[P1 any, P2 any]                         = func(ctx context.Context, p1 P1, p2 P2) func(queryRunner QueryRunner) error
	TxFunc3[P1 any, P2 any, P3 any]                 = func(ctx context.Context, p1 P1, p2 P2, p3 P3) func(queryRunner QueryRunner) error
	TxFunc4[P1 any, P2 any, P3 any, P4 any]         = func(ctx context.Context, p1 P1, p2 P2, p3 P3, p4 P4) func(queryRunner QueryRunner) error
	TxFunc5[P1 any, P2 any, P3 any, P4 any, P5 any] = func(ctx context.Context, p1 P1, p2 P2, p3 P3, p4 P4, p5 P5) func(queryRunner QueryRunner) error
)

func CarryTxFunc(ctx context.Context, f CarryFunc) TransactionFunc {
	return func(queryRunner QueryRunner) error {
		return f(ctx, queryRunner)
	}
}

func CarryDaoFunc0(f DaoFunc0) CarryFunc {
	return f
}

func CarryDaoFunc1[P1 any](f DaoFunc1[P1], p1 P1) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, queryRunner, p1)
	}
}

func CarryDaoFunc2[P1 any, P2 any](f DaoFunc2[P1, P2], p1 P1, p2 P2) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, queryRunner, p1, p2)
	}
}

func CarryDaoFunc3[P1 any, P2 any, P3 any](f DaoFunc3[P1, P2, P3], p1 P1, p2 P2, p3 P3) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, queryRunner, p1, p2, p3)
	}
}

func CarryDaoFunc4[P1 any, P2 any, P3 any, P4 any](f DaoFunc4[P1, P2, P3, P4], p1 P1, p2 P2, p3 P3, p4 P4) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, queryRunner, p1, p2, p3, p4)
	}
}

func CarryTxFunc0(f TxFunc0) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx)(queryRunner)
	}
}

func CarryTxFunc1[P1 any](f TxFunc1[P1], p1 P1) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, p1)(queryRunner)
	}
}

func CarryTxFunc2[P1 any, P2 any](f TxFunc2[P1, P2], p1 P1, p2 P2) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, p1, p2)(queryRunner)
	}
}

func CarryTxFunc3[P1 any, P2 any, P3 any](f TxFunc3[P1, P2, P3], p1 P1, p2 P2, p3 P3) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, p1, p2, p3)(queryRunner)
	}
}

func CarryTxFunc4[P1 any, P2 any, P3 any, P4 any](f TxFunc4[P1, P2, P3, P4], p1 P1, p2 P2, p3 P3, p4 P4) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, p1, p2, p3, p4)(queryRunner)
	}
}

func CarryTxFunc5[P1 any, P2 any, P3 any, P4 any, P5 any](f TxFunc5[P1, P2, P3, P4, P5], p1 P1, p2 P2, p3 P3, p4 P4, p5 P5) CarryFunc {
	return func(ctx context.Context, queryRunner QueryRunner) error {
		return f(ctx, p1, p2, p3, p4, p5)(queryRunner)
	}
}
