package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
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

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	testInstructionExecutionUuid := timedOutTestInstructionExecutionToUpdate.TestInstructionExecutionUuid

	var testInstructionExecutionStatus string
	testInstructionExecutionStatus = strconv.Itoa(int(timedOutTestInstructionExecutionToUpdate.TestInstructionExecutionStatus))
	testInstructionExecutionEndTimeStamp := common_config.ConvertGrpcTimeStampToStringForDB(timedOutTestInstructionExecutionToUpdate.TestInstructionExecutionEndTimeStamp)

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
		errorId := "6f87dc32-61aa-4d29-a812-8b85d32d8cc1"
		err = errors.New(fmt.Sprintf("TestInstructionExecutionUuid '%s' is missing in Table [ErroId: %s]", testInstructionExecutionUuid, errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                           "4737bf84-4444-4e0d-8653-cd2d1ad86b12",
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"sqlToExecute":                 sqlToExecute,
		}).Error("TestInstructionExecutionUuid is missing in Table")

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
