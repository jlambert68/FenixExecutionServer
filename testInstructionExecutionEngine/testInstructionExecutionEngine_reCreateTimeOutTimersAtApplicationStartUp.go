package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
)

// At start up this function will load all TestInstructionExecutions that is still ongoing and recreate their TimeOut-timers in TimeOut-Engine
func (executionEngine *TestInstructionExecutionEngineStruct) reCreateTimeOutTimersAtApplicationStartUp(
	executionTrackNumber int) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "d7b706ce-14e2-449a-828c-3720c8844fcf",
	}).Debug("Incoming 'reCreateTimeOutTimersAtApplicationStartUp'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "d6cf998d-c1a9-4ad1-80f0-08b5e52a1b68",
	}).Debug("Outgoing 'reCreateTimeOutTimersAtApplicationStartUp'")

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "d9c2acad-99a3-45ef-b0be-b1aec798cf8b",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'reCreateTimeOutTimersAtApplicationStartUp'")

		return err
	}

	// Close db-transaction when leaving this function
	defer txn.Commit(context.Background())

	// Load all TestInstructionsExecutions for Zombie-TestInstructions that should be classified to have ongoing TimeOut-timer
	var testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct
	testInstructionExecutionsToProcess, err = executionEngine.loadAllZombieTestInstructionExecutionsIWithOngoingTimeoutTimers(txn)
	if err != nil {
		return err
	}

	// If there are no TestInstructionExecutions then just end
	if testInstructionExecutionsToProcess == nil {
		return nil
	}

	// Loop all TestInstructionExecutions, with Zombie-TestInstructionExecutions, and trigger Create TimeOutTimer for the TestInstructionExecutions
	for _, testInstructionExecutionToProcess := range testInstructionExecutionsToProcess {

		// Allocate TimeOut-timer
		// Create a message with TestInstructionExecution to be sent to TimeOutEngine ta be able to allocate a TimeOutTimer
		var tempTimeOutChannelTestInstructionExecutionsAllocate common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutionsAllocate = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestCaseExecutionUuid:                   testInstructionExecutionToProcess.TestCaseExecutionUuid,
			TestCaseExecutionVersion:                testInstructionExecutionToProcess.TestInstructionExecutionVersion,
			TestInstructionExecutionUuid:            testInstructionExecutionToProcess.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion:         1,
			TimeOutTime:                             testInstructionExecutionToProcess.TestInstructionTimeOutTime,
			TestInstructionExecutionCanBeReExecuted: testInstructionExecutionToProcess.TestInstructionExecutionCanBeReExecuted,
		}

		var tempTimeOutChannelCommandAllocate common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommandAllocate = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAllocateTestInstructionExecutionToTimeOutTimer,
			TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutionsAllocate,
			SendID:                                  "8a70b3d7-bbf6-458a-90bb-553e7d181834",
		}

		// Calculate Execution Track
		var executionTrack int
		executionTrack = common_config.CalculateExecutionTrackNumber(
			testInstructionExecutionToProcess.TestInstructionExecutionUuid)

		// Send message on TimeOutEngineChannel to Allocate the TestInstructionExecution to Timer-queue
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommandAllocate

		// Add a TimeOut-timer
		// Create a message with TestInstructionExecution to be sent to TimeOutEngine ta be able to Add a TimeOutTimer
		var tempTimeOutChannelTestInstructionExecutionsAdd common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutionsAdd = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestCaseExecutionUuid:                   testInstructionExecutionToProcess.TestCaseExecutionUuid,
			TestCaseExecutionVersion:                testInstructionExecutionToProcess.TestInstructionExecutionVersion,
			TestInstructionExecutionUuid:            testInstructionExecutionToProcess.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion:         1,
			TimeOutTime:                             testInstructionExecutionToProcess.TestInstructionTimeOutTime,
			TestInstructionExecutionCanBeReExecuted: testInstructionExecutionToProcess.TestInstructionExecutionCanBeReExecuted,
		}

		var tempTimeOutChannelCommandAdd common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommandAdd = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer,
			TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutionsAdd,
			SendID:                                  "1ccab0e2-ee16-4d7b-9ba0-f1344dc491a5",
		}

		// Send message on TimeOutEngineChannel to Add the TestInstructionExecution to Timer-queue
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommandAdd

	}

	return err

}

// Load all TestInstructionsExecutions for Zombie-TestInstructions that should be classified to have ongoing TimeOut-timer
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestInstructionExecutionsIWithOngoingTimeoutTimers(
	dbTransaction pgx.Tx) (
	testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct,
	err error) {

	var currentTimeStampForDB string
	currentTimeStampForDB = common_config.GenerateDatetimeTimeStampForDB()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT  TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", " +
		"TIUE.\"TestInstructionCanBeReExecuted\", TIUE.\"ExpectedExecutionEndTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionEndTimeStamp\" IS NULL AND "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionHasFinished\" = false AND "
	sqlToExecute = sqlToExecute + "TIUE.\"ExpectedExecutionEndTimeStamp\" >= '" + currentTimeStampForDB + "' "
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"ExpectedExecutionEndTimeStamp\" ASC "
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "358764d5-40ab-4d7e-a4e3-fe75d68a2f36",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadAllZombieTestInstructionExecutionsIWithOngoingTimeoutTimers'")
	}

	// Query DB
	// Execute Query CloudDB
	rows, err := dbTransaction.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "3acd678c-5c1b-40b8-9154-847197a621d6",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionExecutionToProcess ChannelCommandTestInstructionExecutionStruct

		err := rows.Scan(

			&tempTestInstructionExecutionToProcess.TestCaseExecutionUuid,
			&tempTestInstructionExecutionToProcess.TestCaseExecutionVersion,
			&tempTestInstructionExecutionToProcess.TestInstructionExecutionUuid,
			&tempTestInstructionExecutionToProcess.TestInstructionExecutionVersion,
			&tempTestInstructionExecutionToProcess.TestInstructionExecutionCanBeReExecuted,
			&tempTestInstructionExecutionToProcess.TestInstructionTimeOutTime,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "ef5e5123-68c6-4384-974a-a3006da73d72",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add Queue-message to slice of messages
		testInstructionExecutionsToProcess = append(testInstructionExecutionsToProcess, tempTestInstructionExecutionToProcess)
	}

	return testInstructionExecutionsToProcess, err

}
