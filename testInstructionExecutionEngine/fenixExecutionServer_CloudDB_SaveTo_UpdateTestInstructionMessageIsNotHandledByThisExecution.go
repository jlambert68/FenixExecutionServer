package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
	"strings"
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

// Insert row in table that tells that the TestInstructionExecution is not handled and needs to be picked up by correct ExecutionInstance
func (executionEngine *TestInstructionExecutionEngineStruct) updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB(
	dbTransaction pgx.Tx,
	testInstructionExecutionMessageReceivedByWrongExecution common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct) (
	err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"Id": "e13a24b8-816a-47ef-8235-ab0faf547510",
		"testInstructionExecutionMessageReceivedByWrongExecution": testInstructionExecutionMessageReceivedByWrongExecution,
	}).Debug("Entering: updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB()")

	defer func() {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id": "be4fe03a-7c09-4041-95b1-46a1c823fda1",
		}).Debug("Exiting: updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB()")
	}()

	// Secure that there are no "'" in the json
	var cleanedMessageAsJsonString string
	cleanedMessageAsJsonString = strings.ReplaceAll(testInstructionExecutionMessageReceivedByWrongExecution.MessageAsJsonString, "'", "\"")

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
		testInstructionExecutionMessageReceivedByWrongExecution.TestInstructionExecutionUuid)
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType,
		testInstructionExecutionMessageReceivedByWrongExecution.TestInstructionExecutionVersion)
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType,
		common_config.GenerateDatetimeTimeStampForDB())
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType,
		int(testInstructionExecutionMessageReceivedByWrongExecution.MessageType))
	dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, cleanedMessageAsJsonString)

	dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	/*
		create table "FenixExecution"."TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc"
		(
		    "ApplicationExecutionRuntimeUuid"  uuid,
		    "TestInstructionExecutionUuid"    uuid      not null,
		    "TestInstructionExecutionVersion" integer   not null,
		    "TimeStamp"                       timestamp not null,
		    "MessageType"                      integer   not null
		        constraint "TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc"
		            references "FenixExecution"."TestInstructionExecutionExecutionMessageType",
		    "MessageAsJsonb"                  jsonb     not null
		);

		alter table "FenixExecution"."TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc"
		    owner to postgres;


	*/

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc\" "
	sqlToExecute = sqlToExecute + "(\"ApplicationExecutionRuntimeUuid\", \"TestInstructionExecutionUuid\", " +
		"\"TestInstructionExecutionVersion\", \"TimeStamp\", \"MessageType\", \"MessageAsJsonb\") "
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
