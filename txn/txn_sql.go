package txn

import (
	"context"
	"database/sql"
)

type SqlBeginner = *sql.DB

func SqlExecute[D Doer[Txn, SqlBeginner]](
	ctx context.Context, db SqlBeginner, do D, fn DoFunc[Txn, SqlBeginner, D]) (D, error) {
	return do, ExecuteTxn(ctx, db, do, fn)
}

type SqlOptions = *sql.TxOptions

type SqlDoerBase struct {
	DoerBase[SqlOptions]
}

func (d *SqlDoerBase) IsReadOnly() bool {
	return d.Options().ReadOnly
}

type SqlTx = *sql.Tx

type SqlWrapper struct {
	Raw SqlTx
}

func (w *SqlWrapper) Commit(context.Context) error {
	return w.Raw.Commit()
}

func (w *SqlWrapper) Rollback(context.Context) error {
	return w.Raw.Rollback()
}
