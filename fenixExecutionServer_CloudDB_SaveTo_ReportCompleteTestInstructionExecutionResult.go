package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	"errors"
	"fmt"
	uuidGenerator "github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) commitOrRoleBackReportCompleteTestInstructionExecutionResult(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testCaseExecutionsToProcessReference *[]testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct,
	thereExistsOnGoingTestInstructionExecutionsReference *bool,
	triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutionsReference *bool) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference
	thereExistsOnGoingTestInstructionExecutions := *thereExistsOnGoingTestInstructionExecutionsReference
	triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions := *triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutionsReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Create response channel to be able to get response when ChannelCommand has finished
		var returnChannelWithDBError testInstructionExecutionEngine.ReturnChannelWithDBErrorType
		returnChannelWithDBError = make(chan testInstructionExecutionEngine.ReturnChannelWithDBErrorStruct)

		// Update status for TestCaseExecution, based on incoming TestInstructionExecution
		if triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions == true {
			channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
				ChannelCommand:                    testInstructionExecutionEngine.ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
				ChannelCommandTestCaseExecutions:  testCaseExecutionsToProcess,
				ReturnChannelWithDBErrorReference: &returnChannelWithDBError,
			}

			*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage

			// Wait for errReturnMessage in return channel
			var returnChannelMessage testInstructionExecutionEngine.ReturnChannelWithDBErrorStruct
			returnChannelMessage = <-returnChannelWithDBError

			//Check if there was an error in previous ChannelCommand, if so then exit
			if returnChannelMessage.Err != nil {
				return
			}

			// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue, If we got an OK as respons from TestInstruction

			channelCommandMessage = testInstructionExecutionEngine.ChannelCommandStruct{
				ChannelCommand:                   testInstructionExecutionEngine.ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
				ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
			}

			*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage

		} else {

			// Update status for TestCaseExecution, based on incoming TestInstructionExecution
			channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
				ChannelCommand:                    testInstructionExecutionEngine.ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
				ChannelCommandTestCaseExecutions:  testCaseExecutionsToProcess,
				ReturnChannelWithDBErrorReference: &returnChannelWithDBError,
			}

			*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage

			// Wait for errReturnMessage in return channel
			var returnChannelMessage testInstructionExecutionEngine.ReturnChannelWithDBErrorStruct
			returnChannelMessage = <-returnChannelWithDBError

			//Check if there was an error in previous ChannelCommand, if so then exit
			if returnChannelMessage.Err != nil {
				return
			}

			// If there are Ongoing TestInstructionsExecutions then secure that they are triggered to be sent to Worker
			if thereExistsOnGoingTestInstructionExecutions == true {
				channelCommandMessage = testInstructionExecutionEngine.ChannelCommandStruct{
					ChannelCommand:                   testInstructionExecutionEngine.ChannelCommandCheckOngoingTestInstructionExecutions,
					ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
				}
				*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage

			}
		}
	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecution in the CloudDB
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB(finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

	// Verify that the ExecutionStatus is a final status
	// (0, 'TIE_INITIATED') -> NOT OK
	// (1, 'TIE_EXECUTING') -> NOT OK
	// (2, 'TIE_CONTROLLED_INTERRUPTION' -> OK
	// (3, 'TIE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (4, 'TIE_FINISHED_OK' -> OK
	// (5, 'TIE_FINISHED_OK_CAN_BE_RERUN' -> OK
	// (6, 'TIE_FINISHED_NOT_OK' -> OK
	// (7, 'TIE_FINISHED_NOT_OK_CAN_BE_RERUN' -> OK
	// (8, 'TIE_UNEXPECTED_INTERRUPTION' -> OK
	// (9, 'TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN' -> OK
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

	// TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage
	var testCaseExecutionsToProcess []testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct

	// TestInstructionExecution didn't end with an OK(4, 'TIE_FINISHED_OK' or 5, 'TIE_FINISHED_OK_CAN_BE_RERUN') then Stop further processing
	var thereExistsOnGoingTestInstructionExecutionsOnQueue bool

	// If this is the last TestInstructionExecution and any TestInstructionExecution failed, then trigger change in TestCaseExecution-status
	var triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions bool

	defer fenixExecutionServerObject.commitOrRoleBackReportCompleteTestInstructionExecutionResult(
		&txn,
		&doCommitNotRoleBack,
		&testCaseExecutionsToProcess,
		&thereExistsOnGoingTestInstructionExecutionsOnQueue,
		&triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions) //txn.Commit(context.Background())

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

	// Load TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage
	testCaseExecutionsToProcess, err = fenixExecutionServerObject.loadTestCaseExecutionAndTestCaseExecutionVersion(finalTestInstructionExecutionResultMessage)
	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when loading TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage, from database: " + err.Error(),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// If this is the last on TestInstructionExecution and any of them ended with a 'Non-OK-status' then stop pick new TestInstructionExecutions from Queue
	var testInstructionExecutionSiblingsStatus []*testInstructionExecutionSiblingsStatusStruct
	testInstructionExecutionSiblingsStatus, err = fenixExecutionServerObject.areAllOngoingTestInstructionExecutionsFinishedAndAreAnyTestInstructionExecutionEndedWithNonOkStatus(finalTestInstructionExecutionResultMessage)

	// When there are TestInstructionExecutions in result set then they can have been ended with a Non-OK-status or that they are ongoing in their executions
	if len(testInstructionExecutionSiblingsStatus) != 0 {

		for _, testInstructionExecution := range testInstructionExecutionSiblingsStatus {
			// Is this any ongoing TestInstructionExecutions?
			if testInstructionExecution.testInstructionExecutionStatus < 2 {
				thereExistsOnGoingTestInstructionExecutionsOnQueue = true
				break
			}
		}

	} else {
		// All TestInstructionsExecution ended with an OK-status so Update TestCaseExecutionStatus and Check for New TestInstructionExecutions on Queue
		triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions = true
	}

	// Update Status on TestCaseExecution

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

type testInstructionExecutionSiblingsStatusStruct struct {
	testCaseExecutionUuid                      string
	testCaseExecutionVersion                   int
	testInstructionExecutionUuid               string
	testInstructionInstructionExecutionVersion int
	testInstructionExecutionStatus             int
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

// Load TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) loadTestCaseExecutionAndTestCaseExecutionVersion(finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (testCaseExecutionsToProcess []testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionUuid\" = '" + finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid + "'; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "f0fbac73-b7e6-4eea-9932-4ce49d690fd8",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Variable to store TestCaseExecutionUUID and its TestCaseExecutionVersion
	var channelCommandTestCaseExecutions []testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct

	// Extract data from DB result set
	for rows.Next() {
		var channelCommandTestCaseExecution testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct

		err := rows.Scan(
			&channelCommandTestCaseExecution.TestCaseExecution,
			&channelCommandTestCaseExecution.TestCaseExecutionVersion,
		)

		if err != nil {

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
				"Id":           "ab5ec697-c33e-49d1-8f03-e297a05ffccc",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add TestCaseExecutionUUID and its TestCaseExecutionVersion to slice of messages
		channelCommandTestCaseExecutions = append(channelCommandTestCaseExecutions, channelCommandTestCaseExecution)

	}

	// Verify that we got exactly one row from database
	if len(channelCommandTestCaseExecutions) != 1 {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":                               "84270631-b49c-486a-9ea9-704979d6b387",
			"channelCommandTestCaseExecutions": channelCommandTestCaseExecutions,
			"Number of Rows":                   len(channelCommandTestCaseExecutions),
		}).Error("The result gave not exactly one row from database")

	}

	return channelCommandTestCaseExecutions, err

}

// Verify if siblings to current finsihed TestInstructionExecutions are all finished and if any of them ended with a Non-OK-status
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) areAllOngoingTestInstructionExecutionsFinishedAndAreAnyTestInstructionExecutionEndedWithNonOkStatus(finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (testInstructionExecutionSiblingsStatus []*testInstructionExecutionSiblingsStatusStruct, err error) {

	// Generate UUID as part of name for Temp-table AND
	tempTableUuid := uuidGenerator.New().String()
	tempTableName := "tempTable_" + tempTableUuid

	//usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Create SQL that only List TestInstructionExecutions that did not end with a OK-status
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "CREATE TEMP TABLE " + tempTableName + " AS "

	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\",  TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionUuid\" = '" + finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid + " AND "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionInstructionExecutionVersion\" = 1; "

	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionStatus\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE, tempTableName "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestCaseExecutionUuid\" = " + tempTableName + ".\"TestCaseExecutionUuid\" AND "
	sqlToExecute = sqlToExecute + "TIUE.\"TestCaseExecutionVersion\" = " + tempTableName + ".\"TestCaseExecutionVersion\" AND "
	sqlToExecute = sqlToExecute + "(TIUE.\"TestInstructionExecutionStatus\" < 4 OR "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionStatus\" > 5);"

	sqlToExecute = sqlToExecute + "DROP TABLE " + tempTableName + ";"

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "a414a9b3-bed8-49ed-9ec4-b2077725f7fd",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result
	for rows.Next() {
		var testInstructionExecutionSiblingStatus *testInstructionExecutionSiblingsStatusStruct

		err = rows.Scan(
			&testInstructionExecutionSiblingStatus.testCaseExecutionUuid,
			&testInstructionExecutionSiblingStatus.testCaseExecutionVersion,
			&testInstructionExecutionSiblingStatus.testInstructionExecutionUuid,
			&testInstructionExecutionSiblingStatus.testInstructionInstructionExecutionVersion,
			&testInstructionExecutionSiblingStatus.testInstructionExecutionStatus,
		)

		if err != nil {

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
				"Id":           "0c30827d-e9e1-4962-b28b-ea74b05e4dc7",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add status for TestInstructionExecution-sibling to slice
		testInstructionExecutionSiblingsStatus = append(testInstructionExecutionSiblingsStatus, testInstructionExecutionSiblingStatus)

	}

	return testInstructionExecutionSiblingsStatus, err

}
