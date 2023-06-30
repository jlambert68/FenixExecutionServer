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

// SecondsWithOutAnswerFromWorker - Number of seconds without answer from Worker that a TestInstructionExecution will be counted as a Zombie
const SecondsWithOutAnswerFromWorker time.Duration = time.Second * 15

// TestInstructions that, of some reason, are stuck in UnderExecution will be resent
// If no response was received back from Worker, and 'some time' has passed
func (executionEngine *TestInstructionExecutionEngineStruct) sendAllZombieTestInstructionsUnderExecution(
	executionTrackNumber int) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "7dec5937-5436-494d-a66e-fb77b49bf6c4",
	}).Debug("Incoming 'sendAllZombieTestInstructionsUnderExecution'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "3633ae34-262f-4634-aaa6-d2603b9ad51d",
	}).Debug("Outgoing 'sendAllZombieTestInstructionsUnderExecution'")

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "271c0bba-fb4f-433d-a07b-56d5b86ed60a",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'sendAllZombieTestInstructionsUnderExecution'")

		return err
	}

	// Close db-transaction when leaving this function
	defer txn.Commit(context.Background())

	// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct
	testCaseExecutionsToProcess, err = executionEngine.loadAllZombieTestInstructionExecutionsUnderExecution(txn)
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

	// Loop all TestCaseExecutions, with Zombie-TestInstructionExecutions, and trigger resend for the TestInstructionExecutions to Workers
	for _, testCaseExecutionToProcess := range testCaseExecutionsToProcess {

		// Trigger TestInstructionEngine to check if there are TestInstructionExecutions, in under Execution, waiting to be sent to Worker
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
			ChannelCommandTestCaseExecutions: []ChannelCommandTestCaseExecutionStruct{testCaseExecutionToProcess},
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
	}

	return err

}

// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestInstructionExecutionsUnderExecution(
	dbTransaction pgx.Tx) (
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct,
	err error) {

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT " +
		"TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", TIUE.\"SentTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"ExpectedExecutionEndTimeStamp\" IS NULL AND " +
		"TIUE.\"TestInstructionCanBeReExecuted\" = true "
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"SentTimeStamp\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "18cf15d9-917f-4c39-9c0b-4138a1fd8a2f",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadAllZombieTestInstructionExecutionsUnderExecution'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "5be15e8b-fbb9-4912-a847-40bc4076006b",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Temp variables
	var tempSentTimeStamp time.Time

	// Extract data from DB result set
	for rows.Next() {

		var tempTestCaseExecutionToProcess ChannelCommandTestCaseExecutionStruct

		err := rows.Scan(

			&tempTestCaseExecutionToProcess.TestCaseExecutionUuid,
			&tempTestCaseExecutionToProcess.TestCaseExecutionVersion,
			&tempSentTimeStamp,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "e8212ea5-b12e-4e37-a8dc-502bf1012906",
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
