package broadcastingEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"encoding/json"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"log"
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
	BroadcastTimeStamp               string                                           `json:"broadcasttimestamp"`
	PreviousBroadcastTimeStamp       string                                           `json:"previousbroadcasttimestamp"`
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
}

var err error

func InitiateAndStartBroadcastNotifyEngine() {

	BroadcastEngineMessageChannel = make(chan BroadcastingMessageForExecutionsStruct, BroadcastEngineChannelSize)
	var broadcastingMessageForExecutions BroadcastingMessageForExecutionsStruct
	var broadcastingMessageForExecutionsAsByteSlice []byte
	var broadcastingMessageForExecutionsAsByteSliceAsString string
	var err error
	var channelSize int
	var broadcastTimestamp time.Time
	var previousBroadCastTimestamp time.Time
	var firstTime bool = true

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

		// Set Broadcast Timestamps
		broadcastingMessageForExecutions.BroadcastTimeStamp = strings.Split(broadcastTimestamp.UTC().String(), " m=")[0]
		broadcastingMessageForExecutions.PreviousBroadcastTimeStamp = strings.Split(previousBroadCastTimestamp.UTC().String(), " m=")[0]

		// secure when there exists 'nil' in message, regarding "TestCaseExecutions"
		if broadcastingMessageForExecutions.TestCaseExecutions == nil {
			broadcastingMessageForExecutions.TestCaseExecutions = make([]TestCaseExecutionBroadcastMessageStruct, 0)
		}

		// secure when there exists 'nil' in message, regarding "TestInstructionExecutions"
		if broadcastingMessageForExecutions.TestInstructionExecutions == nil {
			broadcastingMessageForExecutions.TestInstructionExecutions = make([]TestInstructionExecutionBroadcastMessageStruct, 0)
		}

		broadcastingMessageForExecutionsAsByteSlice, err = json.Marshal(broadcastingMessageForExecutions)
		broadcastingMessageForExecutionsAsByteSliceAsString = string(broadcastingMessageForExecutionsAsByteSlice)

		common_config.Logger.WithFields(logrus.Fields{
			"id": "5c9019a5-1b97-4d5f-97a3-79977f6aa824",
			"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
		}).Debug("Message sent over Broadcast system")

		_, err = fenixSyncShared.DbPool.Exec(context.Background(), "SELECT pg_notify('notes', $1)", broadcastingMessageForExecutionsAsByteSlice)
		if err != nil {
			log.Println("error sending notification:", err)
		}

		previousBroadCastTimestamp = broadcastTimestamp
	}
}
