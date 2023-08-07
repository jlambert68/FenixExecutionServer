package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

// Commit or Rollback changes 'TestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB' and send over Broadcast-channel
func (executionEngine *TestInstructionExecutionEngineStruct) commitOrRoleBackTestInstructionIsNotHandledByThisExecutionInstance(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	broadcastingMessageForExecutionsReference *common_config.BroadcastingMessageForTestInstructionExecutionsStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	broadcastingMessageForExecutions := *broadcastingMessageForExecutionsReference

	// Should transaction be committed and be broadcast
	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Send message to BroadcastEngine over channel
		common_config.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

		common_config.Logger.WithFields(logrus.Fields{
			"id":                               "f88f7282-be22-43eb-bed0-a6e600d3db99",
			"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
		}).Debug("Sent message for broadcasting (broadcastingEngine_TestInstructionNotHandledByThisInstance)")

	} else {

		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) prepareTestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB(
	executionTrackNumber int,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) {

	// Calculate Execution Track
	var executionTrack int
	executionTrack = common_config.CalculateExecutionTrackNumber(
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	common_config.Logger.WithFields(logrus.Fields{
		"id": "34a32e60-6aa8-4400-ba2c-ff07a8f830e3",
		"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		"executionTrack": executionTrack,
	}).Debug("Incoming 'prepareTestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "5fc72e58-f1fd-4c64-ab9d-28166bf02743",
	}).Debug("Outgoing 'prepareTestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB'")

	// Begin SQL Transaction
	var txn pgx.Tx
	var err error
	txn, err = fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "67830336-4592-40bf-b86e-e9f1720ebd85",
			"error": err,
			"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB', dropping 'finalTestInstructionExecutionResultMessage'")

		return
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Message to be sent over Broadcast-system that this ExecutionInstance is not responsible for this TestInstructionExecution
	var broadcastingMessageForExecutions common_config.
		BroadcastingMessageForTestInstructionExecutionsStruct

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer executionEngine.commitOrRoleBackTestInstructionIsNotHandledByThisExecutionInstance(
		&txn,
		&doCommitNotRoleBack,
		&broadcastingMessageForExecutions)

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	err = executionEngine.updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB(txn, finalTestInstructionExecutionResultMessage)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "ee92c4fa-999a-47a8-aa64-00a6e00212c9",
			"error": err,
			"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		}).Error("Problem when saving 'finalTestInstructionExecutionResultMessage' to database in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB', will dropp 'finalTestInstructionExecutionResultMessage'")

		return
	}

	// Create the BroadCastMessage for the TestInstructionExecution
	var testInstructionExecutionBroadcastMessages []common_config.
		TestInstructionExecutionBroadcastMessageStruct
	var testInstructionExecutionBroadcastMessage common_config.
		TestInstructionExecutionBroadcastMessageStruct

	testInstructionExecutionBroadcastMessage = common_config.
		TestInstructionExecutionBroadcastMessageStruct{
		TestInstructionExecutionUuid:    finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion: "1",
	}
	testInstructionExecutionBroadcastMessages = append(testInstructionExecutionBroadcastMessages,
		testInstructionExecutionBroadcastMessage)

	broadcastingMessageForExecutions = common_config.
		BroadcastingMessageForTestInstructionExecutionsStruct{
		OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
		TestInstructionExecutions:        testInstructionExecutionBroadcastMessages,
	}

	// Do the commit and send over Broadcast-system
	doCommitNotRoleBack = true
}

// Insert row in table that tells that the TestInstructionExecution is not handled and needs to be picked up by correct ExecutionInstance
func (executionEngine *TestInstructionExecutionEngineStruct) updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB(
	dbTransaction pgx.Tx,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (
	err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"Id": "e13a24b8-816a-47ef-8235-ab0faf547510",
		"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
	}).Debug("Entering: updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB()")

	defer func() {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id": "be4fe03a-7c09-4041-95b1-46a1c823fda1",
		}).Debug("Exiting: updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB()")
	}()

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	var dataRowToBeInsertedMultiType []interface{}
	var dataRowsToBeInsertedMultiType [][]interface{}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Insert Statement for Ongoing TestInstructionExecution
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	dataRowToBeInsertedMultiType = nil

	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, common_config.ApplicationRuntimeUuid)
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType,
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, 1)
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType,
		int(finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus))
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType,
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionEndTimeStamp.String())
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp)

	dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionExecutionsReceivedByWrongExecutionInstance\" "
	sqlToExecute = sqlToExecute + "(\"ApplicationExecutionRuntimeUuid\", \"TestInstructionExecutionUuid\", " +
		"\"TestInstructionExecutionVersion\", \"TestInstructionExecutionStatus\", \"TestInstructionExecutionEndTimeStamp\", " +
		" \"TimeStamp\") "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "d6e90243-2194-4805-ba81-bbcd80498d3d",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "7cb0a424-dd00-4785-90b5-493cc4f38e5b",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                       "899baa48-9253-4e9e-997a-45ae5200e3b8",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}
