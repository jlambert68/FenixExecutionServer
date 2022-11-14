package testInstructionExecutionEngine

import (
	"FenixExecutionServer/broadcastingEngine"
	"FenixExecutionServer/common_config"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecutionInCloudDBCommitOrRoleBack(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	//testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct,
	testCaseExecutionsReference *[]broadcastingEngine.TestCaseExecutionStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	//testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference
	testCaseExecutions := *testCaseExecutionsReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Create message to be sent to BroadcastEngine
		var broadcastingMessageForExecutions broadcastingEngine.BroadcastingMessageForExecutionsStruct
		broadcastingMessageForExecutions = broadcastingEngine.BroadcastingMessageForExecutionsStruct{
			BroadcastTimeStamp:        time.Now().String(),
			TestCaseExecutions:        testCaseExecutions,
			TestInstructionExecutions: nil,
		}

		// Send message to BroadcastEngine over channel
		broadcastingEngine.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

		/*
			// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
			go func() {
				channelCommandMessage := ChannelCommandStruct{
					ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
					ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
				}

				*executionEngine.executionEngineChannelRef <- channelCommandMessage



			}()

		*/
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

	// All TestCaseExecutions
	var testCaseExecutions []broadcastingEngine.TestCaseExecutionStruct

	defer executionEngine.updateStatusOnTestCaseExecutionInCloudDBCommitOrRoleBack(
		&txn,
		&doCommitNotRoleBack,
		//&testCaseExecutionsToProcess,
		&testCaseExecutions) //txn.Commit(context.Background())

	// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecution
	var loadTestInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct
	loadTestInstructionExecutionStatusMessages, err = executionEngine.loadTestInstructionExecutionStatusMessages(testCaseExecutionsToProcess)

	// Exit when there was a problem reading  the database
	if err != nil {
		return err
	}

	// Transform TestInstructionExecutionStatus into correct prioritized TestCaseExecutionStatus
	var testCaseExecutionStatusMessages []*testCaseExecutionStatusStruct
	testCaseExecutionStatusMessages, err = executionEngine.transformTestInstructionExecutionStatusIntoTestCaseExecutionStatus(loadTestInstructionExecutionStatusMessages)

	// Exit when there was a problem when converting into TestCaseExecutionStatus
	if err != nil {
		return err
	}

	// Get number of TestInstructionExecutions that is waiting on TestInstructionExecutionQueue
	var numberOfTestInstructionExecutionsOnExecutionQueueMap numberOfTestInstructionExecutionsOnQueueMapType
	numberOfTestInstructionExecutionsOnExecutionQueueMap, err = executionEngine.loadNumberOfTestInstructionExecutionsOnExecutionQueue(txn, testCaseExecutionsToProcess)
	// Exit when there was a problem updating the database
	if err != nil {
		return err
	}

	// Update TestExecutions in database with the new TestCaseExecutionStatus
	err = executionEngine.updateTestCaseExecutionsWithNewTestCaseExecutionStatus(txn, testCaseExecutionStatusMessages, numberOfTestInstructionExecutionsOnExecutionQueueMap)

	// Exit when there was a problem updating the database
	if err != nil {
		return err
	}

	// Prepare message data to be sent over Broadcast system
	for _, testCaseExecutionStatusMessage := range testCaseExecutionStatusMessages {

		var testCaseExecution broadcastingEngine.TestCaseExecutionStruct
		testCaseExecution = broadcastingEngine.TestCaseExecutionStruct{
			TestCaseExecutionUuid:    testCaseExecutionStatusMessage.TestCaseExecutionUuid,
			TestCaseExecutionVersion: testCaseExecutionStatusMessage.TestCaseExecutionVersion,
			TestCaseExecutionStatus:  fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_name[int32(testCaseExecutionStatusMessage.TestCaseExecutionStatus)],
		}

		// Add TestCaseExecution to slice of executions to be sent over Broadcast system
		testCaseExecutions = append(testCaseExecutions, testCaseExecution)
	}

	// No errors occurred so secure that commit is done
	doCommitNotRoleBack = true

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

// used as type when getting number of TestInstructionExecutions on TestInstructionExecutionQueue
type numberOfTestInstructionExecutionsOnQueueStruct struct {
	numberOfTestInstructionExecutionsOnQueue int
	TestInstructionExecutionUuid             string
	TestInstructionExecutionVersion          int
}

// Type used when to store Map with TestInstructionsExecutions from ExecutionQueue
type numberOfTestInstructionExecutionsOnQueueMapType map[string]*numberOfTestInstructionExecutionsOnQueueStruct

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

	// Extract data from DB result set
	for rows.Next() {
		var loadTestInstructionExecutionStatusMessage loadTestInstructionExecutionStatusMessagesStruct

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

// Transform TestInstructionExecutionStatus into correct prioritized TestCaseExecutionStatus
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
	// map[TestInstructionExecutionStatus]=PrioritizationOrder
	var statusOrderDecisionMap = map[int]int{
		0: 0,
		4: 1,
		5: 2,
		1: 3,
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

// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecution
func (executionEngine *TestInstructionExecutionEngineStruct) loadNumberOfTestInstructionExecutionsOnExecutionQueue(dbTransaction pgx.Tx, testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (numberOfTestInstructionExecutionsOnExecutionQueueMap numberOfTestInstructionExecutionsOnQueueMapType, err error) {

	// Initiate response map
	numberOfTestInstructionExecutionsOnExecutionQueueMap = make(numberOfTestInstructionExecutionsOnQueueMapType)
	var mapKey string

	//usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate WHERE-values to only target correct 'TestCaseExecutionUuid' together with 'TestCaseExecutionVersion'
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar string
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars string
	for testCaseExecutionCounter, testCaseExecution := range testCaseExecutionsToProcess {
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar =
			"(TIEQ.\"TestCaseExecutionUuid\" = '" + testCaseExecution.TestCaseExecution + "' AND " +
				"TIEQ.\"TestCaseExecutionVersion\" = " + strconv.Itoa(int(testCaseExecution.TestCaseExecutionVersion)) + ") "

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
	sqlToExecute = sqlToExecute + "SELECT COUNT(TIEQ.*), TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "GROUP BY TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + ";"

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "21aa22c2-8b39-4c9c-ace0-3ef8354f61af",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result set
	for rows.Next() {
		var numberOfTestInstructionExecutionOnExecutionQueue numberOfTestInstructionExecutionsOnQueueStruct

		err := rows.Scan(
			&numberOfTestInstructionExecutionOnExecutionQueue.numberOfTestInstructionExecutionsOnQueue,
			&numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionUuid,
			&numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionVersion,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "ac7cd745-63b5-4484-9092-1cb8471d0bb5",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add TestInstructionExecutionStatus-message to map of messages
		mapKey = numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionUuid + strconv.Itoa(numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionVersion)
		numberOfTestInstructionExecutionsOnExecutionQueueMap[mapKey] = &numberOfTestInstructionExecutionOnExecutionQueue

	}

	return numberOfTestInstructionExecutionsOnExecutionQueueMap, err

}

// Update TestExecutions in database with the new TestCaseExecutionStatus
func (executionEngine *TestInstructionExecutionEngineStruct) updateTestCaseExecutionsWithNewTestCaseExecutionStatus(dbTransaction pgx.Tx, testCaseExecutionStatusMessages []*testCaseExecutionStatusStruct, numberOfTestInstructionExecutionsOnExecutionQueueMap numberOfTestInstructionExecutionsOnQueueMapType) (err error) {

	// If there are nothing to update then exit
	if len(testCaseExecutionStatusMessages) == 0 {
		return err
	}

	var mapKey string
	var existsInMap bool

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate TimeStamp used in SQL
	testCaseExecutionUpdateTimeStamp := common_config.GenerateDatetimeTimeStampForDB()

	// Loop all TestCaseExecutions and execute SQL-Update
	for _, testCaseExecutionStatusMessage := range testCaseExecutionStatusMessages {

		// If there are any TestInstructionExecutions on the ExecutionQueue, then Upgrade status From 'OK', if so is the Case, to Ongoing Execution
		mapKey = testCaseExecutionStatusMessage.TestCaseExecutionUuid + strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionVersion)
		_, existsInMap = numberOfTestInstructionExecutionsOnExecutionQueueMap[mapKey]
		if existsInMap == true {

			// If Current Reported TestCaseStatus is "TCE_FINISHED_OK" or "TCE_FINISHED_OK_CAN_BE_RERUN" then change status to "TCE_EXECUTING"
			if int32(testCaseExecutionStatusMessage.TestCaseExecutionStatus) == int32(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK) ||
				int32(testCaseExecutionStatusMessage.TestCaseExecutionStatus) == int32(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK_CAN_BE_RERUN) {
				testCaseExecutionStatusMessage.TestCaseExecutionStatus = int(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_EXECUTING)
			}
		}

		var testCaseExecutionStatusAsString string
		var testCaseExecutionVersionAsString string
		testCaseExecutionStatusAsString = strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionStatus)
		testCaseExecutionVersionAsString = strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionVersion)

		//LOCK TABLE 'TestCasesUnderExecution' IN ROW EXCLUSIVE MODE;

		SqlToExecuteRowLock := ""
		SqlToExecuteRowLock = SqlToExecuteRowLock + "SELECT TCEUE.* "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "FROM \"FenixExecution\".\"TestCasesUnderExecution\" TCEUE "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "WHERE TCEUE.\"TestCaseExecutionUuid\" = '" + testCaseExecutionStatusMessage.TestCaseExecutionUuid + "' AND "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "TCEUE.\"TestCaseExecutionVersion\" = " + testCaseExecutionVersionAsString + " "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "FOR UPDATE; "

		// Execute Query CloudDB
		comandTagRowLock, err := dbTransaction.Exec(context.Background(), SqlToExecuteRowLock)

		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                  "87de5b59-dd8b-4256-9973-1c422fd890be",
				"Error":               err,
				"SqlToExecuteRowLock": SqlToExecuteRowLock,
			}).Error("Something went wrong when executing SQL")

			return err
		}

		// Log response from CloudDB
		common_config.Logger.WithFields(logrus.Fields{
			"Id":                       "ca11702a-2876-4558-bf05-bee83c435ede",
			"comandTag.Insert()":       comandTagRowLock.Insert(),
			"comandTag.Delete()":       comandTagRowLock.Delete(),
			"comandTag.Select()":       comandTagRowLock.Select(),
			"comandTag.Update()":       comandTagRowLock.Update(),
			"comandTag.RowsAffected()": comandTagRowLock.RowsAffected(),
			"comandTag.String()":       comandTagRowLock.String(),
		}).Debug("Return data for SQL executed in database")

		// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
		if comandTagRowLock.RowsAffected() != 1 {
			errorId := "cc47776a-959f-496a-821b-8a6e6e711549"

			err = errors.New(fmt.Sprintf("TestICaseExecutionUuid '%s' with Execution Version Number '%s' is missing in Table [ErroId: %s]", testCaseExecutionStatusMessage.TestCaseExecutionUuid, testCaseExecutionStatusMessage.TestCaseExecutionVersion, errorId))

			common_config.Logger.WithFields(logrus.Fields{
				"Id": "99420b20-0bbc-420d-8123-b7b7e5f8eac7",
				"testCaseExecutionStatusMessage.TestCaseExecutionUuid":    testCaseExecutionStatusMessage.TestCaseExecutionUuid,
				"testCaseExecutionStatusMessage.TestCaseExecutionVersion": testCaseExecutionStatusMessage.TestCaseExecutionVersion,

				"SqlToExecuteRowLock": SqlToExecuteRowLock,
			}).Error("TestInstructionExecutionUuid is missing in Table")

			return err
		}

		// Create Update Statement for each TestCaseExecution update
		sqlToExecute := ""
		sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
		sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStopTimeStamp\" = '%s', ", testCaseExecutionUpdateTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionStatus\" = %s, ", testCaseExecutionStatusAsString)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionHasFinished\" = '%s', ", "true")
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s' ", testCaseExecutionUpdateTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestCaseExecutionUuid\" = '%s' ", testCaseExecutionStatusMessage.TestCaseExecutionUuid)
		sqlToExecute = sqlToExecute + fmt.Sprintf("AND \"TestCaseExecutionVersion\" = %s ", testCaseExecutionVersionAsString)
		sqlToExecute = sqlToExecute + "; "

		// If no positive responses the just exit
		if len(sqlToExecute) == 0 {
			return nil
		}

		// Execute Query CloudDB
		comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "34b9c93b-a94a-4a65-8ccf-041bfae4b250",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when executing SQL")

			return err
		}

		// Log response from CloudDB
		common_config.Logger.WithFields(logrus.Fields{
			"Id":                       "dd728940-319c-466f-9422-373deae31523",
			"comandTag.Insert()":       comandTag.Insert(),
			"comandTag.Delete()":       comandTag.Delete(),
			"comandTag.Select()":       comandTag.Select(),
			"comandTag.Update()":       comandTag.Update(),
			"comandTag.RowsAffected()": comandTag.RowsAffected(),
			"comandTag.String()":       comandTag.String(),
		}).Debug("Return data for SQL executed in database")

		// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
		if comandTag.RowsAffected() != 1 {
			errorId := "e3df8260-d463-4644-b999-6e6e94b54956"

			err = errors.New(fmt.Sprintf("TestICaseExecutionUuid '%s' with Execution Version Number '%s' is missing in Table [ErroId: %s]", testCaseExecutionStatusMessage.TestCaseExecutionUuid, testCaseExecutionStatusMessage.TestCaseExecutionVersion, errorId))

			common_config.Logger.WithFields(logrus.Fields{
				"Id": "76be13be-8649-41e2-b7f2-3b5e54e26192",
				"testCaseExecutionStatusMessage.TestCaseExecutionUuid":    testCaseExecutionStatusMessage.TestCaseExecutionUuid,
				"testCaseExecutionStatusMessage.TestCaseExecutionVersion": testCaseExecutionStatusMessage.TestCaseExecutionVersion,

				"sqlToExecute": sqlToExecute,
			}).Error("TestInstructionExecutionUuid is missing in Table")

			return err
		}
	}

	// No errors occurred
	return err

}
