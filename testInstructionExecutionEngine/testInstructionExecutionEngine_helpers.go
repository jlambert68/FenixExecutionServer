package testInstructionExecutionEngine

import (
	"context"
	"github.com/jackc/pgx/v4"
)

func (executionEngine *TestInstructionExecutionEngineStruct) commitOrRoleBackParallellSave(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

	} else {
		dbTransaction.Rollback(context.Background())
	}
}
