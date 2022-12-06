package testInstructionExecutionEngine

import (
	"context"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"time"
)

// SecondsWithOutAnswerFromWorker - Number of seconds without answer from Worker that a TestInstructionExecution will be counted as a Zombie
const SecondsWithOutAnswerFromWorker time.Duration = time.Second * 15

// TestInstructions that, of some reason, are stuck in UnderExecution will be resent
// If no response was received back from Worker, and 'some time' has passed
func (executionEngine *TestInstructionExecutionEngineStruct) sendAllZombieTestInstructionsUnderExecution() (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "7dec5937-5436-494d-a66e-fb77b49bf6c4",
	}).Debug("Incoming 'sendAllZombieTestInstructionsUnderExecution'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "3633ae34-262f-4634-aaa6-d2603b9ad51d",
	}).Debug("Outgoing 'sendAllZombieTestInstructionsUnderExecution'")

	// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct
	testCaseExecutionsToProcess, err = executionEngine.loadAllZombieTestInstructionExecutionsUnderExecution()
	if err != nil {
		return err
	}

	// Loop all TestCaseExecutions, with Zombie-TestInstructionExecutions, and trigger resend for the TestInstructionExecutions to Workers
	for _, testCaseExecutionToProcess := range testCaseExecutionsToProcess {

		// Trigger TestInstructionEngine to check if there are TestInstructionExecutions, in under Execution, waiting to be sent to Worker
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
			ChannelCommandTestCaseExecutions: []ChannelCommandTestCaseExecutionStruct{testCaseExecutionToProcess},
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReference <- channelCommandMessage
	}

	return err

}

// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestInstructionExecutionsUnderExecution() (testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct, err error) {

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT " +
		"TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", TIUE.\"SentTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE"
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"ExpectedExecutionEndTimeStamp\" IS NULL AND " +
		"TIUE.\"TestInstructionCanBeReExecuted\" = true"
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"SentTimeStamp\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

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
