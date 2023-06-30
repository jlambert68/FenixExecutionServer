package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

// TestInstructions that, of some reason, are stuck in OnExecutionQueue will be reprocessed
func (executionEngine *TestInstructionExecutionEngineStruct) sendAllZombieTestInstructionsOnExecutionQueue(
	executionTrackNumber int) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "4975d842-23cc-4714-872d-453a990bf609",
	}).Debug("Incoming 'sendAllZombieTestInstructionsOnExecutionQueue'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "ecbefc72-f244-4ee5-beb9-e5098842a589",
	}).Debug("Outgoing 'sendAllZombieTestInstructionsOnExecutionQueue'")

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "5e76978d-431c-4d7f-a119-9bdee673ae47",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB'")

		return err
	}

	defer txn.Commit(context.Background())

	// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck in UnderExecutions
	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct
	testCaseExecutionsToProcess, err = executionEngine.loadAllZombieTestInstructionExecutionsOnExecutionQueue(txn)
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
			ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
			ChannelCommandTestCaseExecutions: []ChannelCommandTestCaseExecutionStruct{testCaseExecutionToProcess},
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
	}

	return err

}

// Load all TestCasesExecutions for Zombie-TestInstructions that are stuck OnExecutionsQueue
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllZombieTestInstructionExecutionsOnExecutionQueue(dbTransaction pgx.Tx) (testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct, err error) {

	/*
		sqlToExecute := ""
		sqlToExecute = sqlToExecute + "SELECT DISTINCT " +
			"TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\", TIEQ.\"QueueTimeStamp\" "
		sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionExecutionQueue\" TIEQ "
		sqlToExecute = sqlToExecute + "ORDER BY TIEQ.\"QueueTimeStamp\" ASC "
		sqlToExecute = sqlToExecute + "; "
	*/

	// Generate unique number for temporary table namne
	rand.Seed(time.Now().UnixNano())
	min := 1
	max := 100000
	randomNumber := rand.Intn(max-min+1) + min
	var randomNumberAsString string
	randomNumberAsString = strconv.Itoa(randomNumber)
	var tempraryTableName = "TEMP_TABLE_" + randomNumberAsString

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "CREATE TEMPORARY TABLE " + tempraryTableName + " ON COMMIT DROP AS "
	sqlToExecute = sqlToExecute + "SELECT DISTINCT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "GROUP BY TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", TIUE.\"TestInstructionExecutionStatus\" "
	sqlToExecute = sqlToExecute + "HAVING TIUE.\"TestInstructionExecutionStatus\"  NOT IN (4) " // 4=TestInstructionExecution ended OK
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "36989958-b28e-4225-8cef-7888393d1b8f",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadAllZombieTestInstructionExecutionsOnExecutionQueue'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "09356aee-fb7a-4731-8741-d3cfd61844ca",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "dff4d8fd-3c9b-456f-95b4-fe75a0103a5d",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	sqlToExecute = ""
	sqlToExecute = sqlToExecute + "SELECT TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\", " +
		"TIEQ.\"QueueTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "LEFT JOIN  " + tempraryTableName + " tmp "
	sqlToExecute = sqlToExecute + "ON TIEQ.\"TestCaseExecutionUuid\" = tmp.\"TestCaseExecutionUuid\" AND "
	sqlToExecute = sqlToExecute + "TIEQ.\"TestCaseExecutionVersion\" = tmp.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "WHERE tmp.\"TestCaseExecutionUuid\"  IS NULL AND "
	sqlToExecute = sqlToExecute + "tmp.\"TestCaseExecutionVersion\" IS NULL "
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "70214c17-2712-4a25-ab30-e4f784fc65c9",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadAllZombieTestInstructionExecutionsOnExecutionQueue'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "934fbad2-5197-4881-82fd-3edc47ce9c50",
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
