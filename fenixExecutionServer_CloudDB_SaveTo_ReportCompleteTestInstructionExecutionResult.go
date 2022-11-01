package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) commitOrRoleBackReportCompleteTestInstructionExecutionResult(dbTransactionReference *pgx.Tx, doCommitNotRoleBackReference *bool) {
	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
		go func() {
			channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
				ChannelCommand: testInstructionExecutionEngine.ChannelCommandCheckTestInstructionExecutionQueue,
			}

			*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage

		}()

	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecution in the CloudDB
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB(finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

	// Verify that the ExecutionStatus is a final status
	// (0, 'TIE_INITIATED') -> NOT OK
	// (1, 'TIE_EXECUTING') -> NOT OK
	// (2, 'TIE_CONTROLLED_INTERRUPTION') -> OK
	// (3, 'TIE_FINISHED_OK') -> OK
	// (4, 'TIE_FINISHED_NOT_OK') -> OK
	// (5, 'TIE_UNEXPECTED_INTERRUPTION' -> OK
	if finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus < 2 {

		common_config.Logger.WithFields(logrus.Fields{
			"id": "d9ef51cf-1d36-4df2-a719-c1390823e252",
			"finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus": finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus,
		}).Error("'TestInstructionExecutionStatus' is not a final status for a TestInstructionExecution. Must be '> 1'")

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     fmt.Sprintf("'TestInstructionExecutionStatus' is not a final status for a TestInstructionExecution. Got '%s' but expected value '> 1'", finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "76a47577-da52-4cae-82fb-37f0947ad6a9",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB'")

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer fenixExecutionServerObject.commitOrRoleBackReportCompleteTestInstructionExecutionResult(&txn, &doCommitNotRoleBack) //txn.Commit(context.Background())

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	err = fenixExecutionServerObject.updateStatusOnTestInstructionsExecutionInCloudDB(txn, finalTestInstructionExecutionResultMessage)
	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when Updating TestInstructionExecutionStatus in database: " + err.Error(),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// Commit every database change
	doCommitNotRoleBack = true

	// Create Return message
	ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:                      true,
		Comments:                     "",
		ErrorCodes:                   nil,
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
	}

	return ackNackResponse
}

// Update status, which came from Connector/Worker, on ongoing TestInstructionExecution
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) updateStatusOnTestInstructionsExecutionInCloudDB(dbTransaction pgx.Tx, finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (err error) {

	// If there are nothing to update then just exit
	if finalTestInstructionExecutionResultMessage == nil {
		return nil
	}

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	testInstructionExecutionUuid := finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid

	var testInstructionExecutionStatus string
	testInstructionExecutionStatus = strconv.Itoa(int(finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus))
	testInstructionExecutionEndTimeStamp := common_config.ConvertGrpcTimeStampToStringForDB(finalTestInstructionExecutionResultMessage.TestInstructionExecutionEndTimeStamp)

	// Create Update Statement  TestInstructionExecution

	sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
	sqlToExecute = sqlToExecute + "\"TestInstructionExecutionStatus\" = " + testInstructionExecutionStatus + ", " // TIE_EXECUTING
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s', ", currentDataTimeStamp)
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionHasFinished\" = '%s', ", "true")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionEndTimeStamp\" = '%s' ", testInstructionExecutionEndTimeStamp)
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestInstructionExecutionUuid\" = '%s' ", testInstructionExecutionUuid)

	sqlToExecute = sqlToExecute + "; "

	// If no positive responses the just exit
	if len(sqlToExecute) == 0 {
		return nil
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "e2a88e5e-a3b0-47d4-b867-93324126fbe7",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "ffa5c358-ba6c-47bd-a828-d9e5d826f913",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
	if comandTag.RowsAffected() != 1 {
		errorId := "89e28340-64cd-40f3-921f-caa7729c5d0b"
		err = errors.New(fmt.Sprintf("TestInstructionExecutionUuid '%s' is missing in Table [ErroId: %s]", testInstructionExecutionUuid, errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                           "e2a88e5e-a3b0-47d4-b867-93324126fbe7",
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"sqlToExecute":                 sqlToExecute,
		}).Error("TestInstructionExecutionUuid is missing in Table")

		return err
	}

	// No errors occurred
	return err

}
