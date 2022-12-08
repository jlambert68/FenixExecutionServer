package testInstructionExecutionEngine

import (
	"context"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"time"
)

// TestInstructions that, of some reason, are stuck in OnExecutionQueue will be reprocessed
func (executionEngine *TestInstructionExecutionEngineStruct) sendAllZombieTestInstructionsOnExecutionQueue() (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "4975d842-23cc-4714-872d-453a990bf609",
	}).Debug("Incoming 'sendAllZombieTestInstructionsOnExecutionQueue'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "ecbefc72-f244-4ee5-beb9-e5098842a589",
	}).Debug("Outgoing 'sendAllZombieTestInstructionsOnExecutionQueue'")

	// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct
	testCaseExecutionsToProcess, err = executionEngine.loadAllZombieTestInstructionExecutionsOnExecutionQueue()
	if err != nil {
		return err
	}

	// Loop all TestCaseExecutions, with Zombie-TestInstructionExecutions, and trigger resend for the TestInstructionExecutions to Workers
	for _, testCaseExecutionToProcess := range testCaseExecutionsToProcess {

		// Trigger TestInstructionEngine to check if there are TestInstructionExecutions, in under Execution, waiting to be sent to Worker
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
			ChannelCommandTestCaseExecutions: []ChannelCommandTestCaseExecutionStruct{testCaseExecutionToProcess},
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReference <- channelCommandMessage
	}

	return err

}

// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck OnExecutionsQueue
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestInstructionExecutionsOnExecutionQueue() (testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct, err error) {

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT " +
		"TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\", TIEQ.\"QueueTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "ORDER BY TIEQ.\"QueueTimeStamp\" ASC "
	sqlToExecute = sqlToExecute + "; "

	SELECT DISTINCT TIUE."TestCaseExecutionUuid", TIUE."TestCaseExecutionVersion"
	INTO TABLE TEMP_TABLE
	FROM "FenixExecution"."TestInstructionsUnderExecution" TIUE
	GROUP BY TIUE."TestCaseExecutionUuid", TIUE."TestCaseExecutionVersion", TIUE."TestInstructionExecutionStatus"
	HAVING TIUE."TestInstructionExecutionStatus"  NOT IN (4);

	SELECT TIEQ."TestCaseExecutionUuid", TIEQ."TestCaseExecutionVersion", TIEQ."QueueTimeStamp"
	FROM "FenixExecution"."TestInstructionExecutionQueue" TIEQ
	LEFT JOIN TEMP_TABLE tmp
	ON TIEQ."TestCaseExecutionUuid" = tmp."TestCaseExecutionUuid" AND
	TIEQ."TestCaseExecutionVersion" = tmp."TestCaseExecutionVersion"
	WHERE tmp."TestCaseExecutionUuid"  IS NULL AND
	tmp."TestCaseExecutionVersion" IS NULL;

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "6641053f-8c30-4fd0-872c-ff33d7498588",
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
				"Id":           "58cb6f96-310d-4f32-9c86-483658f5aed6",
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
