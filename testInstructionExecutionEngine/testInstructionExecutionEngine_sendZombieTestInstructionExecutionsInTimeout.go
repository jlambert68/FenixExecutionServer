package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"time"
)

// TestInstructions  in UnderExecution that, of some reason, should be put into timeout-status
func (executionEngine *TestInstructionExecutionEngineStruct) findAllZombieTestInstructionExecutionsInTimeout() (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "3a8e228e-a7e6-470e-927f-a961d1caa015",
	}).Debug("Incoming 'findAllZombieTestInstructionExecutionsInTimeout'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "dfe4934f-56f0-4444-b703-15d803376a54",
	}).Debug("Outgoing 'findAllZombieTestInstructionExecutionsInTimeout'")

	// Load all TestInstructionsExecutions for Zombie-TestInstructions that should be classified to have timeout-status
	var testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct
	testInstructionExecutionsToProcess, err = executionEngine.loadAllZombieTestInstructionExecutionsInTimeout()
	if err != nil {
		return err
	}

	// Loop all TestInstructionExecutions, with Zombie-TestInstructionExecutions, and trigger resend for the TestInstructionExecutions to Workers
	for _, testInstructionExecutionToProcess := range testInstructionExecutionsToProcess {

		// Trigger TestInstructionEngine to check if there are TestInstructionExecutions, in under Execution, waiting to be sent to Worker
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                          ChannelCommandProcessTestInstructionExecutionsThatHaveTimedOut,
			ChannelCommandTestInstructionExecutions: []ChannelCommandTestInstructionExecutionStruct{testInstructionExecutionToProcess},
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReference <- channelCommandMessage
	}

	return err

}

// Load all TestInstructionsExecution for Zombie-TestInstructions that should be put into timeout-status
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestInstructionExecutionsInTimeout() (testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct, err error) {

	var currentTimeStampForDB string
	currentTimeStampForDB = common_config.GenerateDatetimeTimeStampForDB()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT  TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", " +
		"TIUE.\"TestInstructionCanBeReExecuted\", TIUE.\"ExpectedExecutionEndTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionEndTimeStamp\" IS NULL AND "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionHasFinished\" = false AND "
	sqlToExecute = sqlToExecute + "TIUE.\"ExpectedExecutionEndTimeStamp\" < '" + currentTimeStampForDB + "' "
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"ExpectedExecutionEndTimeStamp\" ASC "
	sqlToExecute = sqlToExecute + ";"

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "f29c952f-9771-43a2-ad39-a2e0e20f6cd6",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Temp variables
	var tempExpectedExecutionEndTimeStamp time.Time

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionExecutionToProcess ChannelCommandTestInstructionExecutionStruct

		err := rows.Scan(

			&tempTestInstructionExecutionToProcess.TestCaseExecutionUuid,
			&tempTestInstructionExecutionToProcess.TestCaseExecutionVersion,
			&tempTestInstructionExecutionToProcess.TestInstructionExecutionUuid,
			&tempTestInstructionExecutionToProcess.TestInstructionExecutionVersion,
			&tempTestInstructionExecutionToProcess.TestInstructionExecutionCanBeReExecuted,
			&tempExpectedExecutionEndTimeStamp,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "39d4d8a4-93cd-4ec5-8f7f-893dccf7e0b6",
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
