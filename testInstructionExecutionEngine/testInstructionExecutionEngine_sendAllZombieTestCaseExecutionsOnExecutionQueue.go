package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"sort"
	"time"
)

// TestCaseExecutions that, of some reason, are stuck on ExecutionQueue will be resent
func (executionEngine *TestInstructionExecutionEngineStruct) lookForZombieTestCaseExecutionsOnExecutionQueue(
	executionTrackNumber int) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "ef2d1fdf-d584-48df-909d-f4c7fd9a2ea9",
	}).Debug("Incoming 'sendAllZombieTestCaseExecutionsOnExecutionQueue'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "3df39aec-5c0b-43d0-9317-a673ccca66c6",
	}).Debug("Outgoing 'sendAllZombieTestCaseExecutionsOnExecutionQueue'")

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "e172cf8e-ecbe-4e47-936a-28fa12bb24e6",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'lookForZombieTestCaseExecutionsOnExecutionQueue'")

		return err
	}

	// Close db-transaction when leaving this function
	defer txn.Commit(context.Background())

	// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct
	testCaseExecutionsToProcess, err = executionEngine.loadAllZombieTestCaseExecutionsOnExecutionQueue(txn)
	if err != nil {
		return err
	}

	// Extract "lowest" TestCaseExecutionUuid
	if len(testCaseExecutionsToProcess) > 0 {
		var uuidSlice []string
		for _, uuid := range testCaseExecutionsToProcess {
			uuidSlice = append(uuidSlice, uuid.TestCaseExecutionUuid)
		}
		sort.Strings(uuidSlice)

		// Define Execution Track based on "lowest "TestCaseExecutionUuid
		executionTrackNumber = common_config.CalculateExecutionTrackNumber(uuidSlice[0])
	}

	//  Load all TestCasesExecutions for Zombie-TestCaseExecutions that are stuck on ExecutionQueue
	for _, testCaseExecutionToProcess := range testCaseExecutionsToProcess {

		// Trigger TestInstructionEngine to check if there are TestCaseExecutions on ExecutionsQueue
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandProcessTestCaseExecutionsOnExecutionQueue,
			ChannelCommandTestCaseExecutions: []ChannelCommandTestCaseExecutionStruct{testCaseExecutionToProcess},
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
	}

	return err

}

// Load all TestCasesExecutions for Zombie-TestCaseExecutions that are stuck on ExecutionQueue
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestCaseExecutionsOnExecutionQueue(
	dbTransaction pgx.Tx) (
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct,
	err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT " +
		"TCEQ.\"TestCaseExecutionUuid\", TCEQ.\"TestCaseExecutionVersion\", TCEQ.\"QueueTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestCaseExecutionQueue\" TCEQ "
	sqlToExecute = sqlToExecute + "ORDER BY TCEQ.\"QueueTimeStamp\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "104bc872-b6b5-404f-b955-30263e7e79e6",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadAllZombieTestCaseExecutionsOnExecutionQueue'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "1c19fa8c-41a8-4196-9e5b-e11c5790117f",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Temp variables
	var tempQueueTimeStamp time.Time

	// Extract data from DB result set
	for rows.Next() {

		var tempTestCaseExecutionToProcess ChannelCommandTestCaseExecutionStruct

		err := rows.Scan(

			&tempTestCaseExecutionToProcess.TestCaseExecutionUuid,
			&tempTestCaseExecutionToProcess.TestCaseExecutionVersion,
			&tempQueueTimeStamp,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "4110b994-1a89-45d4-a1b2-9cf985e4ffc1",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add Queue-message to slice of messages
		testCaseExecutionsToProcess = append(testCaseExecutionsToProcess, tempTestCaseExecutionToProcess)
	}

	return testCaseExecutionsToProcess, err

}
