package txn

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxBeginner = *pgxpool.Pool

func PgxExecute[D Doer[Txn, PgxBeginner]](
	ctx context.Context, db PgxBeginner, do D, fn DoFunc[Txn, PgxBeginner, D]) (D, error) {
	return do, ExecuteTxn(ctx, db, do, fn)
}

type PgxOptions = *pgx.TxOptions

type PgxDoerBase struct {
	DoerBase[PgxOptions]
}

func (d *PgxDoerBase) IsReadOnly() bool {
	return strings.Compare(string(pgx.ReadOnly), string(d.Options().AccessMode)) == 0
}
