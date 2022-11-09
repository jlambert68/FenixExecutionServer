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
	loadTestInstructionExecutionStatusMessages, err = executionEngine.loadTestInstructionExecutionStatusMessages(testCaseExecutionsToProcess)

	// Exit when there was a problem reading database
	if err != nil {
		return err
	}

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

// Used as type when deciding what End-status a TestCaseExecution should have depending on its TestInstructionExecutions End-status
type testCaseExecutionStatusStruct struct {
	TestCaseExecutionUuid    string
	TestCaseExecutionVersion int
	TestCaseExecutionStatus  int
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

func (executionEngine *TestInstructionExecutionEngineStruct) transformTestInstructionExecutionStatusIntoTestCaseExecutionStatus(testInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct) (testCaseExecutionStatusMessages []*testCaseExecutionStatusStruct, err error) {
	// For each combination 'TestCaseExecutionUuid && TestCaseExecutionVersion' create correct TestCaseExecutionStatus
	// Generate Map that decides what Status that 'overwrite' other status
	// (0, 'TIE_INITIATED') -> NOT OK
	// (1, 'TIE_EXECUTING') -> NOT OK
	// (4, 'TIE_FINISHED_OK' -> OK
	// (5, 'TIE_FINISHED_OK_CAN_BE_RERUN' -> OK
	// (7, 'TIE_FINISHED_NOT_OK_CAN_BE_RERUN' -> OK
	// (3, 'TIE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (9, 'TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (6, 'TIE_FINISHED_NOT_OK' -> OK
	// (2, 'TIE_CONTROLLED_INTERRUPTION' -> OK
	// (8, 'TIE_UNEXPECTED_INTERRUPTION' -> OK

	// Inparameter for Map is 'Current Status' and the value for the Map prioritized Order for that Status
	var statusOrderDecisionMap = map[int]int{
		0: 0,
		1: 1,
		4: 2,
		5: 3,
		7: 4,
		3: 5,
		9: 6,
		6: 7,
		2: 8,
		8: 9}

	var currentTestCaseExecutionUuid string
	var previousTestCaseExecutionUuid string
	var currentTestCaseExecutionVersion int
	var previousTestCaseExecutionVersion int
	var currentStatus int
	var currentStatusOrder int
	var savedStatusOrder int
	var existInMap bool
	type ruleType int
	var ruleResult ruleType
	var foundRule bool
	const (
		ruleFirstTestCaseExecution ruleType = iota
		ruleSameTestCaseExecution
		ruleNewTestCaseExecutionButNotTheFirstOne
	)

	// Loop all TestInstructionExecutions and extract the highest end-status based on rules in 'statusOrderDecisionMap', for each TestCaseExecution
	for testInstructionExecutionCounter, testInstructionExecution := range testInstructionExecutionStatusMessages {

		currentTestCaseExecutionUuid = testInstructionExecution.TestCaseExecutionUuid
		currentTestCaseExecutionVersion = testInstructionExecution.TestCaseExecutionVersion
		currentStatus = testInstructionExecution.TestInstructionExecutionStatus
		foundRule = false

		//
		currentStatusOrder, existInMap = statusOrderDecisionMap[currentStatus]
		if existInMap == false {
			// Status must exist in map
			executionEngine.logger.WithFields(logrus.Fields{
				"id":            "5f662a03-5fd2-41ec-a4bf-a91f6676aad8",
				"currentStatus": currentStatus,
			}).Error(fmt.Sprintf("Couldn't find TestInstructionExecutionStatus in 'statusOrderDecisionMap'"))

			errorId := "859de457-d6d8-4e23-8691-7581adfb7738"
			err = errors.New(fmt.Sprintf("Couldn't find TestInstructionExecutionStatus, '%s', in 'statusOrderDecisionMap'. [ErrorID='%s']", currentStatus, errorId))

			return nil, err
		}

		// First TestCaseExecution in the list
		if testInstructionExecutionCounter == 0 {

			foundRule = true
			ruleResult = ruleFirstTestCaseExecution
		}

		// Not the first TestCaseExecution, but the same TestCaseExecution as previous one
		if foundRule == false &&
			currentTestCaseExecutionUuid == previousTestCaseExecutionUuid &&
			currentTestCaseExecutionVersion == previousTestCaseExecutionVersion {

			foundRule = true
			ruleResult = ruleSameTestCaseExecution
		}

		// Not the first TestCaseExecution, but a new TestCaseExecution compared to previous TestCaseExecution
		if foundRule == false &&
			currentTestCaseExecutionUuid != previousTestCaseExecutionUuid &&
			currentTestCaseExecutionVersion != previousTestCaseExecutionVersion {

			foundRule = true
			ruleResult = ruleNewTestCaseExecutionButNotTheFirstOne
		}

		switch ruleResult {

		case ruleFirstTestCaseExecution:
			// First TestCaseExecution in the list

			// Create TestCaseExecutionStatus
			var testCaseExecutionStatus *testCaseExecutionStatusStruct
			testCaseExecutionStatus = &testCaseExecutionStatusStruct{
				TestCaseExecutionUuid:    currentTestCaseExecutionUuid,
				TestCaseExecutionVersion: currentTestCaseExecutionVersion,
				TestCaseExecutionStatus:  currentStatus,
			}

			// Add the TestCaseExecution with status based on status from   first TestInstructionExecution
			testCaseExecutionStatusMessages = append(testCaseExecutionStatusMessages, testCaseExecutionStatus)

		case ruleSameTestCaseExecution:
			// Not the first TestCaseExecution, but the same TestCaseExecution as previous one

			// Check if 'currentStatusOrder' is higher than previous saved StatusOrder
			savedStatusOrder, _ = statusOrderDecisionMap[testCaseExecutionStatusMessages[len(testCaseExecutionStatusMessages)-1].TestCaseExecutionStatus]
			if currentStatusOrder > savedStatusOrder {
				// Replace previous Status
				testCaseExecutionStatusMessages[len(testCaseExecutionStatusMessages)-1].TestCaseExecutionStatus = currentStatus
			}

		case ruleNewTestCaseExecutionButNotTheFirstOne:
			// Not the first TestCaseExecution, but a new TestCaseExecution compared to previous TestCaseExecution
			// Create TestCaseExecutionStatus
			var testCaseExecutionStatus *testCaseExecutionStatusStruct
			testCaseExecutionStatus = &testCaseExecutionStatusStruct{
				TestCaseExecutionUuid:    currentTestCaseExecutionUuid,
				TestCaseExecutionVersion: currentTestCaseExecutionVersion,
				TestCaseExecutionStatus:  currentStatus,
			}

			// Add the TestCaseExecution with status based on status from  new TestInstructionExecution
			testCaseExecutionStatusMessages = append(testCaseExecutionStatusMessages, testCaseExecutionStatus)

		default:
			// Unhandled ruleResult
			errorId := "d3ce2635-cb4e-4574-a4ce-a791bac049ed"
			err = errors.New(fmt.Sprintf("Unhandled ruleResult: '%s'. [ErrorID='%s']", ruleResult, errorId))

			return nil, err

		}

		// Move 'current' values to 'previous'
		previousTestCaseExecutionUuid = currentTestCaseExecutionUuid
		previousTestCaseExecutionVersion = currentTestCaseExecutionVersion
	}

	return testCaseExecutionStatusMessages, err

}
