package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

func (executionEngine *TestInstructionExecutionEngineStruct) timeoutHandlerForTestInstructionsUnderExecutionCommitOrRoleBackParallellSave(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference

	if doCommitNotRoleBack == true {

		dbTransaction.Commit(context.Background())

		// Trigger TestInstructionEngine to update status on TestCaseExecutions due to 'Timed Out' TestInstructionExecutions
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
			ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReference <- channelCommandMessage

	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare to update 'timed out' TestInstructionExecutions in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) timeoutHandlerForTestInstructionsUnderExecution(testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id":                                 "637852af-7a8d-4492-b15a-bb96425f3541",
		"testInstructionExecutionsToProcess": testInstructionExecutionsToProcess,
	}).Debug("Incoming 'timeoutHandlerForTestInstructionsUnderExecution'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "e70a9393-7f74-4d3d-8b79-b62b2dc7e4aa",
	}).Debug("Outgoing 'timeoutHandlerForTestInstructionsUnderExecution'")

	// Just exit if nothing is sent in
	if testInstructionExecutionsToProcess == nil {

		defer executionEngine.logger.WithFields(logrus.Fields{
			"id": "8cc70386-9a5b-45b4-9055-fde24c44868a",
		}).Error("Incoming 'testInstructionExecutionsToProcess' is empty")

		return
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "90fb40cd-a836-44a3-a5d5-30b800bac77f",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'timeoutHandlerForTestInstructionsUnderExecution'")

		return
	}

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer executionEngine.timeoutHandlerForTestInstructionsUnderExecutionCommitOrRoleBackParallellSave(
		&txn,
		&doCommitNotRoleBack,
		&testCaseExecutionsToProcess)

	// Loop 'Timed Out' TestInstructionExecutions and update Executions status in DB
	for _, tempTestInstructionExecutionToProcess := range testInstructionExecutionsToProcess {
		err = executionEngine.updateStatusOnTimedOutTestInstructionsExecutionInCloudDB(txn, &tempTestInstructionExecutionToProcess)
		if err != nil {
			return
		}
	}

	// Extract TestCaseExecutions to be able to update their Executions statuses, due to timed out TestInstructionExecutions
	err = executionEngine.getTestCasesExecutionsFromTestInstructionExecutions(&testInstructionExecutionsToProcess, &testCaseExecutionsToProcess)
	if err != nil {
		return
	}

	// Commit every database change
	doCommitNotRoleBack = true

	return
}

// Update status on 'timed out' TestInstructionExecutions
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTimedOutTestInstructionsExecutionInCloudDB(dbTransaction pgx.Tx, timedOutTestInstructionExecutionToUpdate *ChannelCommandTestInstructionExecutionStruct) (err error) {

	// If there are nothing to update then just exit
	if timedOutTestInstructionExecutionToUpdate == nil {
		return nil
	}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	// Extract variables to use in SQL
	// TestInstructionExecutionUuid
	testInstructionExecutionUuid := timedOutTestInstructionExecutionToUpdate.TestInstructionExecutionUuid

	// Extract variables to use in SQL
	// TestInstructionExecutionVersion
	var testInstructionExecutionVersion string
	testInstructionExecutionVersion = strconv.Itoa(int(timedOutTestInstructionExecutionToUpdate.TestInstructionExecutionVersion))

	// Extract variables to use in SQL
	// TestInstructionExecutionStatus
	var testInstructionExecutionTimeOutStatus string
	if timedOutTestInstructionExecutionToUpdate.TestInstructionExecutionCanBeReExecuted == true {
		testInstructionExecutionTimeOutStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_TIMEOUT_INTERRUPTION_CAN_BE_RERUN))
	} else {
		testInstructionExecutionTimeOutStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_TIMEOUT_INTERRUPTION))
	}

	// Create Update Statement  TestInstructionExecution
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
	sqlToExecute = sqlToExecute + "\"TestInstructionExecutionStatus\" = " + testInstructionExecutionTimeOutStatus + ", "
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s', ", currentDataTimeStamp)
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionHasFinished\" = '%s', ", "true")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionEndTimeStamp\" = '%s' ", currentDataTimeStamp)
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestInstructionExecutionUuid\" = '%s' ", testInstructionExecutionUuid)
	sqlToExecute = sqlToExecute + fmt.Sprintf("AND ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("(\"TestInstructionExecutionStatus\" <> %s ",
		strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_TIMEOUT_INTERRUPTION_CAN_BE_RERUN)))
	sqlToExecute = sqlToExecute + fmt.Sprintf("OR ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionStatus\" <> %s) ",
		strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_TIMEOUT_INTERRUPTION)))
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
		"Id":                       "3497a29b-852d-4477-9e07-aed1ddd13a9e",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
	if comandTag.RowsAffected() != 1 {
		errorId := "189d8c7f-6d35-42e4-8f89-f41a8a3f2128"
		err = errors.New(fmt.Sprintf("TestInstructionExecutionUuid '%s' with TestInstructionExecutionVersion '%s' is missing in Table: 'TestInstructionsUnderExecution' [ErroId: %s]", testInstructionExecutionUuid, testInstructionExecutionVersion, errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                           "9b4fc6ed-8083-443a-a523-660df7b94ae5",
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"sqlToExecute":                 sqlToExecute,
			"comandTag.RowsAffected()":     comandTag.RowsAffected(),
		}).Error("TestInstructionExecutionUuid 'might' be missing in Table. It can be so that another instance of ExecutionEngine changed the status before this one could do that.")

		return err
	}

	// No errors occurred
	return err

}

// Extract TestCaseExecutions to update their Executions statues, due to timed out TestInstructionExecutions
func (executionEngine *TestInstructionExecutionEngineStruct) getTestCasesExecutionsFromTestInstructionExecutions(
	testInstructionExecutionsToProcess *[]ChannelCommandTestInstructionExecutionStruct,
	testCaseExecutionsToProcess *[]ChannelCommandTestCaseExecutionStruct) (err error) {

	var testCaseExecutionsMapKey string
	var testCaseExecutionsMap map[string]string //map['testCaseExecutionsMapKey']'testCaseExecutionsMapKey'
	var testCaseExecutionVersionAsString string
	var existInMap bool

	// Initiate Map, map is used to keep track so TestCaseExecutions only are added once to slice
	testCaseExecutionsMap = make(map[string]string)

	// Loop TestInstructionExecutions and extract the TestCaseExecutions
	for _, tempTestInstructionExecution := range *testInstructionExecutionsToProcess {

		// Convert TestCaseVersion into string to be used in 'testCaseExecutionsMapKey'
		testCaseExecutionVersionAsString = strconv.Itoa(int(tempTestInstructionExecution.TestCaseExecutionVersion))

		// Create key for 'testCaseExecutionsMap'
		testCaseExecutionsMapKey = tempTestInstructionExecution.TestCaseExecutionUuid + testCaseExecutionVersionAsString

		// Only add TestCaseInstruction if it not exists in map
		_, existInMap = testCaseExecutionsMap[testCaseExecutionsMapKey]
		if existInMap == false {

			// Add to Map
			testCaseExecutionsMap[testCaseExecutionsMapKey] = testCaseExecutionsMapKey

			// Create object to store in slice
			var testCaseExecution ChannelCommandTestCaseExecutionStruct
			testCaseExecution = ChannelCommandTestCaseExecutionStruct{
				TestCaseExecutionUuid:    tempTestInstructionExecution.TestCaseExecutionUuid,
				TestCaseExecutionVersion: tempTestInstructionExecution.TestCaseExecutionVersion,
			}

			// Add to slice
			*testCaseExecutionsToProcess = append(*testCaseExecutionsToProcess, testCaseExecution)
		}

	}

	return err
}
