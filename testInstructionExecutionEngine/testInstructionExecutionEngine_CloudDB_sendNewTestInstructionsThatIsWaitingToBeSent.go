package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/messagesToExecutionWorker"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Prepare for Saving the ongoing Execution of a new TestCaseExecution in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) sendNewTestInstructionsThatIsWaitingToBeSentWorker() {

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "2effe457-d6b4-47d6-989c-5b4107e52077",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'sendNewTestInstructionsThatIsWaitingToBeSentWorker'")

		return
	}
	// Standard is to do a Rollback
	doCommitNotRoleBack = false
	defer executionEngine.commitOrRoleBackParallellSave(&txn, &doCommitNotRoleBack)

	// Generate a new TestCaseExecution-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Load all TestInstructions and their attributes to be sent to the Executions Workers over gRPC
	rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err := executionEngine.loadNewTestInstructionToBeSentToExecutionWorkers() //(txn)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "25cd9e94-76f6-40ca-8f4c-eed10b618224",
			"error": err,
		}).Error("Got some error when loading the TestInstructionExecutions and Attributes for Executions")

		return
	}

	// If there are no TestInstructions  the exit
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

		return
	}

	// Transform Raw TestInstructions and Attributes, from DB, into messages ready to be sent over gRPC to Execution Workers
	testInstructionsToBeSentToExecutionWorkersAndTheResponse, err := executionEngine.transformRawTestInstructionsAndAttributeIntoGrpcMessages(rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers) //(txn)
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "25cd9e94-76f6-40ca-8f4c-eed10b618224",
			"error": err,
		}).Error("Some problem when transforming raw rawTestInstructionsToBeSentToExecutionWorkers- and Attributes-data into gRPC-messages")

		return
	}

	// If there are no TestInstructions then exit
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

		return
	}

	// Send TestInstructionExecutions with their attributes to correct Execution Worker
	err = executionEngine.sendTestInstructionExecutionsToWorker(testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "ef815265-3b67-4b59-862f-26d352795c95",
			"error": err,
		}).Error("Got some problem when sending sending TestInstructionExecutions to Worker")

		return

	}

	// Update status on TestInstructions that could be sent to workers
	err = executionEngine.updateStatusOnTestInstructionsExecutionInCloudDB(txn, testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "bdbc719a-d354-4ebb-9d4c-d002ec55426c",
			"error": err,
		}).Error("Couldn't update TestInstructionExecutionStatus in CloudDB")

		return

	}

	// Update status on TestCases that TestInstructions have been sent to workers
	err = executionEngine.updateStatusOnTestCasesExecutionInCloudDB(txn, testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "89eb961b-413d-47d1-86b1-8fc1f51c564c",
			"error": err,
		}).Error("Couldn't TestCaseExecutionStatus in CloudDB")

		return

	}

	// Commit every database change
	// TODO Uncomment this doCommitNotRoleBack = true

	return
}

// Hold one new TestInstruction to be sent to Execution Worker
type newTestInstructionToBeSentToExecutionWorkersStruct struct {
	domainUuid                        string
	domainName                        string
	executionWorkerAddress            string
	testInstructionExecutionUuid      string
	testInstructionOriginalUuid       string
	testInstructionName               string
	testInstructionMajorVersionNumber int
	testInstructionMinorVersionNumber int
	testDataSetUuid                   string
	TestCaseExecutionUuid             string
	testCaseExecutionVersion          int
}

// Hold one new TestInstructionAttribute to be sent to Execution Worker
type newTestInstructionAttributeToBeSentToExecutionWorkersStruct struct {
	testInstructionExecutionUuid     string
	testInstructionAttributeType     int
	testInstructionAttributeUuid     string
	testInstructionAttributeName     string
	attributeValueAsString           string
	attributeValueUuid               string
	testInstructionExecutionTypeUuid string
	testInstructionExecutionTypeName string
}

// Load all New TestInstructions and their attributes to be sent to the Executions Workers over gRPC
func (executionEngine *TestInstructionExecutionEngineStruct) loadNewTestInstructionToBeSentToExecutionWorkers() (rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct, rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	var testInstructionExecutionUuids []string

	// *** Process TestInstructions ***
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DP.\"DomainUuid\", DP.\"DomainName\", DP.\"ExecutionWorker Address\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionOriginalUuid\", TIUE.\"TestInstructionName\", " +
		"TIUE.\"TestInstructionMajorVersionNumber\", TIUE.\"TestInstructionMinorVersionNumber\", " +
		"TIUE.\"TestDataSetUuid\", TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE, " +
		"\"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionStatus\" = 0 AND "
	sqlToExecute = sqlToExecute + "DP.\"DomainUuid\" = TIUE.\"DomainUuid\" "
	sqlToExecute = sqlToExecute + "ORDER BY DP.\"DomainUuid\" ASC, TIUE.\"TestInstructionExecutionUuid\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "ac262275-5f05-48c8-982a-46ff2392d3f2",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionAndAttributeData newTestInstructionToBeSentToExecutionWorkersStruct

		err := rows.Scan(

			&tempTestInstructionAndAttributeData.domainUuid,
			&tempTestInstructionAndAttributeData.domainName,
			&tempTestInstructionAndAttributeData.executionWorkerAddress,
			&tempTestInstructionAndAttributeData.testInstructionExecutionUuid,
			&tempTestInstructionAndAttributeData.testInstructionOriginalUuid,
			&tempTestInstructionAndAttributeData.testInstructionName,
			&tempTestInstructionAndAttributeData.testInstructionMajorVersionNumber,
			&tempTestInstructionAndAttributeData.testInstructionMinorVersionNumber,
			&tempTestInstructionAndAttributeData.testDataSetUuid,
			&tempTestInstructionAndAttributeData.TestCaseExecutionUuid,
			&tempTestInstructionAndAttributeData.testCaseExecutionVersion,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "1173f1f3-f9e5-411d-a129-7ee5ca336762",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, nil, err
		}

		// Add Queue-message to slice of messages
		rawTestInstructionsToBeSentToExecutionWorkers = append(rawTestInstructionsToBeSentToExecutionWorkers, tempTestInstructionAndAttributeData)

		// Add TestInstructionExecutionUuid to array to be used in next SQL for Attributes
		testInstructionExecutionUuids = append(testInstructionExecutionUuids, tempTestInstructionAndAttributeData.testInstructionExecutionUuid)
	}

	// If there were no TestInstructions then it can't be any Attributes, so exit
	if len(rawTestInstructionsToBeSentToExecutionWorkers) == 0 {
		return nil, nil, nil
	}

	// *** Process TestInstructionAttributes ***
	sqlToExecute = ""
	sqlToExecute = sqlToExecute + "SELECT TIAUE.\"TestInstructionExecutionUuid\", TIAUE.\"TestInstructionAttributeType\", TIAUE.\"TestInstructionAttributeUuid\", " +
		"TIAUE.\"TestInstructionAttributeName\", TIAUE.\"AttributeValueAsString\", TIAUE.\"AttributeValueUuid\", " +
		"TIAUE.\"TestInstructionAttributeTypeUuid\", TIAUE.\"TestInstructionAttributeTypeName\" "

	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionAttributesUnderExecution\" TIAUE "
	sqlToExecute = sqlToExecute + "WHERE TIAUE.\"TestInstructionExecutionUuid\" IN " + common_config.GenerateSQLINArray(testInstructionExecutionUuids)

	sqlToExecute = sqlToExecute + "ORDER BY TIAUE.\"TestInstructionExecutionUuid\" ASC, TIAUE.\"TestInstructionAttributeUuid\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err = fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "ea92876d-7aa0-4052-afd3-60dcfc79e32c",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionAttribute newTestInstructionAttributeToBeSentToExecutionWorkersStruct

		err := rows.Scan(

			&tempTestInstructionAttribute.testInstructionExecutionUuid,
			&tempTestInstructionAttribute.testInstructionAttributeType,
			&tempTestInstructionAttribute.testInstructionAttributeUuid,
			&tempTestInstructionAttribute.testInstructionAttributeName,
			&tempTestInstructionAttribute.attributeValueAsString,
			&tempTestInstructionAttribute.attributeValueUuid,
			&tempTestInstructionAttribute.testInstructionExecutionTypeUuid,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "bb073f45-0207-475d-ae8f-255bb8ee0cf6",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, nil, err
		}

		// Add Queue-message to slice of messages
		rawTestInstructionAttributesToBeSentToExecutionWorkers = append(rawTestInstructionAttributesToBeSentToExecutionWorkers, tempTestInstructionAttribute)

	}

	return rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err

}

// Holds address, to the execution worker, and a message that will be sent to worker
type processTestInstructionExecutionRequestAndResponseMessageContainer struct {
	domainUuid                              string
	addressToExecutionWorker                string
	testCaseExecutionUuid                   string
	testCaseExecutionVersion                int
	processTestInstructionExecutionRequest  *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest
	processTestInstructionExecutionResponse *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse
}

// Transform Raw TestInstructions from DB into messages ready to be sent over gRPC to Execution Workers
func (executionEngine *TestInstructionExecutionEngineStruct) transformRawTestInstructionsAndAttributeIntoGrpcMessages(rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct, rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct) (testInstructionsToBeSentToExecutionWorkers []*processTestInstructionExecutionRequestAndResponseMessageContainer, err error) {

	attributesMap := make(map[string]*[]newTestInstructionAttributeToBeSentToExecutionWorkersStruct)

	// Create map for attributes to be able to find attributes in an easier way
	for _, rawTestInstructionAttribute := range rawTestInstructionAttributesToBeSentToExecutionWorkers {
		attributesSliceReference, existsInMap := attributesMap[rawTestInstructionAttribute.testInstructionExecutionUuid]

		if existsInMap == true {
			// Exists, so just append value to slice
			*attributesSliceReference = append(*attributesSliceReference, rawTestInstructionAttribute)
		} else {
			// Doesn't exist, so create new slice and add reference to Map
			var newAttributesSlice []newTestInstructionAttributeToBeSentToExecutionWorkersStruct
			newAttributesSlice = append(newAttributesSlice, rawTestInstructionAttribute)
			attributesMap[rawTestInstructionAttribute.testInstructionExecutionUuid] = &newAttributesSlice
		}
	}

	// Loop TestInstruction build message ready to be sent over gRPC to workers
	for _, rawTestInstructionData := range rawTestInstructionsToBeSentToExecutionWorkers {

		var attributesForTestInstruction []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage

		// Extract all attributes that belongs to a certain TestInstructionExecution
		attributesSlice, existsInMap := attributesMap[rawTestInstructionData.testInstructionExecutionUuid]
		if existsInMap == false || len(*attributesSlice) == 0 {
			// No attributes for the TestInstruction
			attributesForTestInstruction = []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage{}
		} else {
			// Create Attributes-message to be added to TestInstructionExecution-message

			var newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage

			for _, attributeInSlice := range *attributesSlice {

				newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage{
					TestInstructionAttributeType: fenixExecutionWorkerGrpcApi.TestInstructionAttributeTypeEnum(attributeInSlice.testInstructionAttributeType),
					TestInstructionAttributeUuid: attributeInSlice.testInstructionAttributeUuid,
					TestInstructionAttributeName: attributeInSlice.testInstructionAttributeName,
					AttributeValueAsString:       attributeInSlice.attributeValueAsString,
					AttributeValueUuid:           attributeInSlice.attributeValueUuid,
				}

				// Append to TestInstructionsAttributes-message
				attributesForTestInstruction = append(attributesForTestInstruction, newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage)
			}
		}

		// Create The TestInstruction and its attributes object, to be sent late
		var newTestInstructionToBeSentToExecutionWorkers *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage
		newTestInstructionToBeSentToExecutionWorkers = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage{
			TestInstructionExecutionUuid: rawTestInstructionData.testInstructionExecutionUuid,
			TestInstructionUuid:          rawTestInstructionData.testInstructionOriginalUuid,
			TestInstructionName:          rawTestInstructionData.testInstructionName,
			MajorVersionNumber:           uint32(rawTestInstructionData.testInstructionMajorVersionNumber),
			MinorVersionNumber:           uint32(rawTestInstructionData.testInstructionMinorVersionNumber),
			TestInstructionAttributes:    attributesForTestInstruction,
		}

		// Create the TestData object, to be sent later
		var newTestDataToBeSentToExecutionWorker *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage
		newTestDataToBeSentToExecutionWorker = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage{
			TestDataSetUuid:           rawTestInstructionData.testDataSetUuid,
			ManualOverrideForTestData: []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage_ManualOverrideForTestDataMessage{},
		}

		// Build the full request to later be sent to Worker
		var newProcessTestInstructionExecutionReveredRequest *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest
		newProcessTestInstructionExecutionReveredRequest = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest{
			ProtoFileVersionUsedByClient: 0,
			TestInstruction:              newTestInstructionToBeSentToExecutionWorkers,
			TestData:                     newTestDataToBeSentToExecutionWorker,
		}

		// Create one full TestInstruction-coontainer, including address to Worker, so TestInstruction can be sent to Execution Worker over gPRC
		var newProcessTestInstructionExecutionRequestMessageContainer processTestInstructionExecutionRequestAndResponseMessageContainer
		newProcessTestInstructionExecutionRequestMessageContainer = processTestInstructionExecutionRequestAndResponseMessageContainer{
			addressToExecutionWorker:               rawTestInstructionData.executionWorkerAddress,
			processTestInstructionExecutionRequest: newProcessTestInstructionExecutionReveredRequest,
			domainUuid:                             rawTestInstructionData.domainUuid,
			testCaseExecutionUuid:                  rawTestInstructionData.TestCaseExecutionUuid,
			testCaseExecutionVersion:               rawTestInstructionData.testCaseExecutionVersion,
		}

		// Add the TestInstruction-container to slice of containers
		testInstructionsToBeSentToExecutionWorkers = append(testInstructionsToBeSentToExecutionWorkers, &newProcessTestInstructionExecutionRequestMessageContainer)

	}

	return testInstructionsToBeSentToExecutionWorkers, err
}

// Transform Raw TestInstructions from DB into messages ready to be sent over gRPC to Execution Workers
func (executionEngine *TestInstructionExecutionEngineStruct) sendTestInstructionExecutionsToWorker(testInstructionsToBeSentToExecutionWorkers []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to send then just exit
	if len(testInstructionsToBeSentToExecutionWorkers) == 0 {
		return nil
	}

	// Set up instance to use for execution gPRC
	var fenixExecutionWorkerObject *messagesToExecutionWorker.MessagesToExecutionWorkerServerObjectStruct
	fenixExecutionWorkerObject = &messagesToExecutionWorker.MessagesToExecutionWorkerServerObjectStruct{Logger: executionEngine.logger}

	// Loop all TestInstructionExecutions and send them to correct Worker for executions
	for _, testInstructionToBeSentToExecutionWorkers := range testInstructionsToBeSentToExecutionWorkers {

		responseFromWorker := fenixExecutionWorkerObject.SendProcessTestInstructionExecutionToExecutionWorkerServer(testInstructionToBeSentToExecutionWorkers.domainUuid, testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest)

		testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionResponse = responseFromWorker

		//fmt.Println(responseFromWorker)
	}
	return err
}

// Update status on TestInstructions that could be sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestInstructionsExecutionInCloudDB(dbTransaction pgx.Tx, testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Update Statement for response regarding TestInstructionExecution
	for _, testInstructionExecutionResponse := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Only save information about TestInstructionExecution if we got a positive response from Worker
		if testInstructionExecutionResponse.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {

			sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
			sqlToExecute = sqlToExecute + fmt.Sprintf("SET \"ExpectedExecutionEndTimeStamp\" = '%s', ", common_config.ConvertGrpcTimeStampToStringForDB(testInstructionExecutionResponse.processTestInstructionExecutionResponse.ExpectedExecutionDuration))
			sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionStatus\" = '%s', ", "1") // TIE_EXECUTING
			sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s' ", currentDataTimeStamp)
			sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestInstructionExecutionUuid\" = '%s' ", testInstructionExecutionResponse.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)

			sqlToExecute = sqlToExecute + "; "
		}
	}

	// If no positive responses the just exit
	if len(sqlToExecute) == 0 {
		return nil
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "e2a88e5e-a3b0-47d4-b867-93324126fbe7",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                       "ffa5c358-ba6c-47bd-a828-d9e5d826f913",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}

// Update status on TestCases that TestInstructions have been sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCasesExecutionInCloudDB(dbTransaction pgx.Tx, testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Update Statement for response regarding TestInstructionExecution
	for _, testInstructionExecution := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Only save information about TestInstructionExecution if we got a positive response from Worker
		if testInstructionExecution.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {

			sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
			sqlToExecute = sqlToExecute + fmt.Sprintf("SET \"TestCaseExecutionStatus\" = '%s', ", "1") // TCE_EXECUTING
			sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s' ", currentDataTimeStamp)
			sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestCaseExecutionUuid\" = '%s' AND ", testInstructionExecution.testCaseExecutionUuid)
			sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionVersion\" = '%s' ", strconv.Itoa(testInstructionExecution.testCaseExecutionVersion))
			sqlToExecute = sqlToExecute + "; "
		}
	}

	// If no positive responses the just exit
	if len(sqlToExecute) == 0 {
		return nil
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "8a1a5099-9383-4c7c-8fc9-58c3521912a4",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                       "433ccd0c-a7a7-4245-b346-b8efedefcfd8",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}
