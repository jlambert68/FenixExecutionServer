package broadcastingEngine_ExecutionStatusUpdate

import (
	"FenixExecutionServer/common_config"
	"context"
	"encoding/json"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

// BroadcastEngineChannelSize
// The size of the channel
const BroadcastEngineChannelSize = 500

// BroadcastEngineChannelWarningLevel
// The size of warning level for the channel
const BroadcastEngineChannelWarningLevel = 400

var BroadcastEngineMessageChannel BroadcastEngineMessageChannelType

type BroadcastEngineMessageChannelType chan BroadcastingMessageForExecutionsStruct

type BroadcastingMessageForExecutionsStruct struct {
	OriginalMessageCreationTimeStamp string                                           `json:"originalmessagecreationtimestamp"`
	TestCaseExecutions               []TestCaseExecutionBroadcastMessageStruct        `json:"testcaseexecutions"`
	TestInstructionExecutions        []TestInstructionExecutionBroadcastMessageStruct `json:"testinstructionexecutions"`
}

type TestCaseExecutionBroadcastMessageStruct struct {
	TestCaseExecutionUuid          string `json:"testcaseexecutionuuid"`
	TestCaseExecutionVersion       string `json:"testcaseexecutionversion"`
	TestCaseExecutionStatus        string `json:"testcaseexecutionstatus"`
	ExecutionStartTimeStamp        string `json:"executionstarttimeStamp"`        // The timestamp when the execution was put for execution, not on queue for execution
	ExecutionStopTimeStamp         string `json:"executionstoptimestamp"`         // The timestamp when the execution was ended
	ExecutionHasFinished           string `json:"executionhasfinished"`           // A simple status telling if the execution has ended or not
	ExecutionStatusUpdateTimeStamp string `json:"executionstatusupdatetimestamp"` // The timestamp when the status was last updated
	BroadcastTimeStamp             string `json:"broadcasttimestamp"`
	PreviousBroadcastTimeStamp     string `json:"previousbroadcasttimestamp"`
	ExecutionStatusReportLevel     fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum
}

type TestInstructionExecutionBroadcastMessageStruct struct {
	TestCaseExecutionUuid                string `json:"testcaseexecutionuuid"`
	TestCaseExecutionVersion             string `json:"testcaseexecutionversion"`
	TestInstructionExecutionUuid         string `json:"testinstructionexecutionuuid"`
	TestInstructionExecutionVersion      string `json:"testinstructionexecutionversion"`
	SentTimeStamp                        string `json:"senttimestamp"`
	ExpectedExecutionEndTimeStamp        string `json:"expectedexecutionendtimestamp"`
	TestInstructionExecutionStatusName   string `json:"testinstructionexecutionstatusname"`
	TestInstructionExecutionStatusValue  string `json:"testinstructionexecutionstatusvalue"`
	TestInstructionExecutionEndTimeStamp string `json:"testinstructionexecutionendtimestamp"`
	TestInstructionExecutionHasFinished  string `json:"testinstructionexecutionhasfinished"`
	UniqueDatabaseRowCounter             string `json:"uniquedatabaserowcounter"`
	TestInstructionCanBeReExecuted       string `json:"testinstructioncanbereexecuted"`
	ExecutionStatusUpdateTimeStamp       string `json:"executionstatusupdatetimestamp"`
	BroadcastTimeStamp                   string `json:"broadcasttimestamp"`
	PreviousBroadcastTimeStamp           string `json:"previousbroadcasttimestamp"`
	ExecutionStatusReportLevel           fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum
}

var err error

func InitiateAndStartBroadcastNotifyEngine_ExecutionStatusUpdate() {

	BroadcastEngineMessageChannel = make(chan BroadcastingMessageForExecutionsStruct, BroadcastEngineChannelSize)
	var broadcastingMessageForExecutions BroadcastingMessageForExecutionsStruct
	var broadcastingMessageForExecutionsAsByteSlice []byte
	var broadcastingMessageForExecutionsAsByteSliceAsString string
	var err error
	var channelSize int
	var broadcastTimestamp time.Time
	var previousBroadCastTimestamp time.Time
	var firstTime bool = true
	var previousBroadcastTimeStampPerTestCaseExecutionMap map[string]time.Time // map[TestCaseExecutionUuid + TestCasExectionVersion]time.Time
	var previousBroadcastTimeStampPerTestCaseExecutionMapKey string
	var existInMap bool

	// Initiate Mape that keeps track of PreviousBroadcastTimeStamp used for TestCaseExecutions-data and TestInstructionExecutions-data
	previousBroadcastTimeStampPerTestCaseExecutionMap = make(map[string]time.Time) // map[TestCaseExecutionUuid + TestCasExectionVersion]time.Time

	for {

		broadcastingMessageForExecutions = <-BroadcastEngineMessageChannel

		// If size of Channel > 'TimeOutChannelWarningLevel' then log Warning message
		channelSize = len(BroadcastEngineMessageChannel)
		if channelSize > BroadcastEngineChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                                 "0e9d8dc0-08e8-4d41-ad07-0c16f5a00dde",
				"channelSize":                        channelSize,
				"BroadcastEngineChannelWarningLevel": BroadcastEngineChannelWarningLevel,
				"BroadcastEngineChannelSize":         BroadcastEngineChannelSize,
			}).Warning("Number of messages on queue for 'BroadcastEngineMessageChannel' has reached a critical level")
		}

		// Move previous Broadcast Timestamp to Previous timestamp variable
		// Use by GUI-client to secure that all  messages was received by the GUI-client
		if firstTime == true {
			firstTime = false

			broadcastTimestamp = time.Now()
			previousBroadCastTimestamp = broadcastTimestamp

		} else {

			previousBroadCastTimestamp = broadcastTimestamp

			// Generate new Broadcast Timestamp
			broadcastTimestamp = time.Now()
		}

		// secure when there exists 'nil' in message, regarding "TestCaseExecutions"
		if broadcastingMessageForExecutions.TestCaseExecutions == nil {
			broadcastingMessageForExecutions.TestCaseExecutions = make([]TestCaseExecutionBroadcastMessageStruct, 0)
		}

		// secure when there exists 'nil' in message, regarding "TestInstructionExecutions"
		if broadcastingMessageForExecutions.TestInstructionExecutions == nil {
			broadcastingMessageForExecutions.TestInstructionExecutions = make([]TestInstructionExecutionBroadcastMessageStruct, 0)
		}

		// Set Broadcast Timestamps for "TestCaseExecutions"
		for testCaseExecutionsStatusMessageCounter, tempTestCaseExecutionsStatusMessage := range broadcastingMessageForExecutions.TestCaseExecutions {
			previousBroadcastTimeStampPerTestCaseExecutionMapKey = tempTestCaseExecutionsStatusMessage.TestCaseExecutionUuid +
				tempTestCaseExecutionsStatusMessage.TestCaseExecutionVersion

			_, existInMap = previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey]
			if existInMap == false {
				// No Previous Timestamp exist for this TestCaseExecution
				previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey] = broadcastTimestamp
				previousBroadCastTimestamp = broadcastTimestamp

			} else {
				// Extract Previous Timestamp and set new Timestamp
				previousBroadCastTimestamp = previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey]
				previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey] = broadcastTimestamp
			}
			// Update TestCaseExecutionsStatusMessage with Broadcast-Timestamps
			broadcastingMessageForExecutions.TestCaseExecutions[testCaseExecutionsStatusMessageCounter].BroadcastTimeStamp = strings.Split(broadcastTimestamp.UTC().String(), " m=")[0]
			broadcastingMessageForExecutions.TestCaseExecutions[testCaseExecutionsStatusMessageCounter].PreviousBroadcastTimeStamp = strings.Split(previousBroadCastTimestamp.UTC().String(), " m=")[0]

		}

		// Set Broadcast Timestamps for "TestInstructionExecutions"
		for testInstructionExecutionsStatusMessageCounter, tempTestInstructionExecution := range broadcastingMessageForExecutions.TestInstructionExecutions {
			previousBroadcastTimeStampPerTestCaseExecutionMapKey = tempTestInstructionExecution.TestCaseExecutionUuid +
				tempTestInstructionExecution.TestCaseExecutionVersion

			_, existInMap = previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey]
			if existInMap == false {
				// No Previous Timestamp exist for this TestCaseExecution
				previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey] = broadcastTimestamp
				previousBroadCastTimestamp = broadcastTimestamp

			} else {
				// Extract Previous Timestamp and set new Timestamp
				previousBroadCastTimestamp = previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey]
				previousBroadcastTimeStampPerTestCaseExecutionMap[previousBroadcastTimeStampPerTestCaseExecutionMapKey] = broadcastTimestamp
			}
			// Update TestInstructionExecution with Broadcast-Timestamps
			broadcastingMessageForExecutions.TestInstructionExecutions[testInstructionExecutionsStatusMessageCounter].BroadcastTimeStamp = strings.Split(broadcastTimestamp.UTC().String(), " m=")[0]
			broadcastingMessageForExecutions.TestInstructionExecutions[testInstructionExecutionsStatusMessageCounter].PreviousBroadcastTimeStamp = strings.Split(previousBroadCastTimestamp.UTC().String(), " m=")[0]

		}

		// Create json as string
		broadcastingMessageForExecutionsAsByteSlice, err = json.Marshal(broadcastingMessageForExecutions)
		broadcastingMessageForExecutionsAsByteSliceAsString = string(broadcastingMessageForExecutionsAsByteSlice)

		// Should Message be sent using Postgres-Broadcast or PubSub to GuiExecutionServer
		if common_config.UsePubSubWhenSendingExecutionStatusToGuiExecutionServer == false {
			// Use Postgres-Broadcast-system

			common_config.Logger.WithFields(logrus.Fields{
				"id": "5c9019a5-1b97-4d5f-97a3-79977f6aa824",
				"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
			}).Debug("Message sent over Broadcast system")

			_, err = fenixSyncShared.DbPool.Exec(context.Background(), "SELECT pg_notify('notes', $1)", broadcastingMessageForExecutionsAsByteSlice)
			if err != nil {
				//log.Println("error sending notification:", err)
				common_config.Logger.WithFields(logrus.Fields{
					"id": "208845b7-666b-4d0b-9891-cc969ab39d2f",
					"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
					"err": err.Error(),
				}).Error("Error sending notification")
			}
		} else {

			// Use PubSub to Publish ExecutionStatus-message to GuiExecutionServer
			err = pubSubPublish(broadcastingMessageForExecutionsAsByteSliceAsString)

			// Some problem when sending over PubSub
			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"id": "208845b7-666b-4d0b-9891-cc969ab39d2f",
					"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
					"err": err.Error(),
				}).Error("Error sending notification")

				continue
			}

			// Success in sending over PubSub
			common_config.Logger.WithFields(logrus.Fields{
				"id": "e3eb343d-62cd-4308-8016-bcff4b50a062",
				"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
			}).Debug("Success in sending over PubSub")

		}

		previousBroadCastTimestamp = broadcastTimestamp
	}
}
