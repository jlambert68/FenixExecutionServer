package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecutionInCloudDBCommitOrRoleBack(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
		go func() {
			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                   ChannelCommandCheckTestInstructionExecutionQueue,
				ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
			}

			*executionEngine.executionEngineChannelRef <- channelCommandMessage

		}()
	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Update status, which came from Connector/Worker, on ongoing TestCaseExecution
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecutionInCloudDB(testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id":                          "c11ebfee-0ca3-4288-af46-7fa9bbe26faa",
		"testCaseExecutionsToProcess": testCaseExecutionsToProcess,
	}).Debug("Incoming 'updateStatusOnTestCaseExecutionInCloudDB'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "c813ca2d-f4f2-4148-b800-662622999a5f",
	}).Debug("Outgoing 'updateStatusOnTestCaseExecutionInCloudDB'")

	// If there are nothing to update then just exit
	if len(testCaseExecutionsToProcess) == 0 {
		return nil
	}

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "01615e5a-2410-4233-b234-0c083ca561ab",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'updateStatusOnTestCaseExecutionInCloudDB'")

		return err
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer executionEngine.updateStatusOnTestCaseExecutionInCloudDBCommitOrRoleBack(
		&txn,
		&doCommitNotRoleBack,
		&testCaseExecutionsToProcess) //txn.Commit(context.Background())

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecution
	var loadTestInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct
	loadTestInstructionExecutionStatusMessages , err = executionEngine.loadTestInstructionExecutionStatusMessages(testCaseExecutionsToProcess)

	// Exit when there was a problem reading database
	if err != nil {
		return err
	}

	 // For each combination 'TestCaseExecutionUuid && TestCaseExecutionVersion' create correct TestCaseExecutionStatus
fortsätt här....

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

// Used as type for respons object when calling 'loadTestInstructionExecutionStatusMessages'
type loadTestInstructionExecutionStatusMessagesStruct struct {
	TestCaseExecutionUuid          string
	TestCaseExecutionVersion       int
	TestInstructionExecutionOrder  int
	TestInstructionName            string
	TestInstructionExecutionStatus int
}

// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecution
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestInstructionExecutionStatusMessages(testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (loadTestInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct, err error) {

	//usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate WHERE-values to only target correct 'TestCaseExecutionUuid' together with 'TestCaseExecutionVersion'
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar string
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars string
	for testCaseExecutionCounter, testCaseExecution := range testCaseExecutionsToProcess {
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar =
			"(TIUE.\"TestCaseExecutionUuid\" = '" + testCaseExecution.TestCaseExecution + "' AND " +
				"TIUE.\"TestCaseExecutionVersion\" = " + strconv.Itoa(int(testCaseExecution.TestCaseExecutionVersion)) + ") "

		switch testCaseExecutionCounter {
		case 0:
			// When this is the first then we need to add 'AND before'
			// *NOT NEEDED* in this Query
			//correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars = "AND "

		default:
			// When this is not the first then we need to add 'OR' after previous
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
				correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + "OR "
		}

		// Add the WHERE-values
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar

	}

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\",  TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionOrder\", TIUE.\"TestInstructionName\", TIUE.\"TestInstructionExecutionStatus\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"TestCaseExecutionUuid\" ASC, TIUE.\"TestInstructionExecutionOrder\" ASC, " +
		"TIUE.\"TestInstructionName\" ASC "
	sqlToExecute = sqlToExecute + ";"

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "77d9657d-f784-4adf-a66a-9173f8dc7f08",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*loadTestInstructionExecutionStatusMessagesStruct{}, err
	}

	var loadTestInstructionExecutionStatusMessage loadTestInstructionExecutionStatusMessagesStruct

	// Extract data from DB result set
	for rows.Next() {


		err := rows.Scan(
			&loadTestInstructionExecutionStatusMessage.TestCaseExecutionUuid,
			&loadTestInstructionExecutionStatusMessage.TestCaseExecutionVersion,
			&loadTestInstructionExecutionStatusMessage.TestInstructionExecutionOrder,
			&loadTestInstructionExecutionStatusMessage.TestInstructionName,
			&loadTestInstructionExecutionStatusMessage.TestInstructionExecutionStatus,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "fb586bd2-8133-4d90-bb02-f5a36cde32d9",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*loadTestInstructionExecutionStatusMessagesStruct{}, err
		}

		// Add TestInstructionExecutionStatus-message to slice of messages
		loadTestInstructionExecutionStatusMessages = append(loadTestInstructionExecutionStatusMessages,
			&loadTestInstructionExecutionStatusMessage)

	}

	return loadTestInstructionExecutionStatusMessages, err

}
