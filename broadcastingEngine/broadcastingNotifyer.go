package broadcastingEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"encoding/json"
	"fmt"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"log"
)

// BroadcastEngineChannelWarningLevel
// The size of the channel
const BroadcastEngineChannelSize = 100

// BroadcastEngineChannelWarningLevel
// The size of warning level for the channel
const BroadcastEngineChannelWarningLevel = 90

var BroadcastEngineMessageChannel BroadcastEngineMessageChannelType

type BroadcastEngineMessageChannelType chan BroadcastingMessageForExecutionsStruct

type BroadcastingMessageForExecutionsStruct struct {
	BroadcastTimeStamp        string                           `json:"timestamp"`
	TestCaseExecutions        []TestCaseExecutionStruct        `json:"testcaseexecutions"`
	TestInstructionExecutions []TestInstructionExecutionStruct `json:"testinstructionexecutions"`
}

type TestCaseExecutionStruct struct {
	TestCaseExecutionUuid          string `json:"testcaseexecutionuuid"`
	TestCaseExecutionVersion       string `json:"testcaseexecutionversion"`
	TestCaseExecutionStatus        string `json:"testcaseexecutionstatus"`
	ExecutionStartTimeStamp        string `json:"executionstarttimeStamp"`        // The timestamp when the execution was put for execution, not on queue for execution
	ExecutionStopTimeStamp         string `json:"executionstoptimestamp"`         // The timestamp when the execution was ended, in anyway
	ExecutionHasFinished           string `json:"executionhasfinished"`           // A simple status telling if the execution has ended or not
	ExecutionStatusUpdateTimeStamp string `json:"executionstatusupdatetimestamp"` // The timestamp when the status was last updated
}

type TestInstructionExecutionStruct struct {
	TestCaseExecutionUuid           string `json:"testcaseexecutionuuid"`
	TestCaseExecutionVersion        string `json:"testcaseexecutionversion"`
	TestInstructionExecutionUuid    string `json:"testinstructionexecutionuuid"`
	TestInstructionExecutionVersion string `json:"testinstructionexecutionversion"`
	TestInstructionExecutionStatus  string `json:"testinstructionexecutionstatus"`
}

var err error

func InitiateAndStartBroadcastNotifyEngine() {

	BroadcastEngineMessageChannel = make(chan BroadcastingMessageForExecutionsStruct, BroadcastEngineChannelSize)
	var broadcastingMessageForExecutions BroadcastingMessageForExecutionsStruct
	var broadcastingMessageForExecutionsAsByteSlice []byte
	var broadcastingMessageForExecutionsAsByteSliceAsString string
	var err error
	var channelSize int

	for {

		broadcastingMessageForExecutions = <-BroadcastEngineMessageChannel

		// If size of Channel > 'TimeOutChannelWarningLevel' then log Warning message
		channelSize = len(BroadcastEngineMessageChannel)
		if channelSize > BroadcastEngineChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                                 "7dafce7a-eed9-448c-81b6-1015733dc1cb",
				"channelSize":                        channelSize,
				"BroadcastEngineChannelWarningLevel": BroadcastEngineChannelWarningLevel,
				"BroadcastEngineChannelSize":         BroadcastEngineChannelSize,
			}).Warning("Number of messages on queue for 'BroadcastEngineMessageChannel' has reached a critical level")
		}

		// secure when there exists 'nil' in message, regarding "TestCaseExecutions"
		if broadcastingMessageForExecutions.TestCaseExecutions == nil {
			broadcastingMessageForExecutions.TestCaseExecutions = make([]TestCaseExecutionStruct, 0)
		}

		// secure when there exists 'nil' in message, regarding "TestInstructionExecutions"
		if broadcastingMessageForExecutions.TestInstructionExecutions == nil {
			broadcastingMessageForExecutions.TestInstructionExecutions = make([]TestInstructionExecutionStruct, 0)
		}

		broadcastingMessageForExecutionsAsByteSlice, err = json.Marshal(broadcastingMessageForExecutions)
		broadcastingMessageForExecutionsAsByteSliceAsString = string(broadcastingMessageForExecutionsAsByteSlice)

		fmt.Println(broadcastingMessageForExecutionsAsByteSliceAsString)

		_, err = fenixSyncShared.DbPool.Exec(context.Background(), "SELECT pg_notify('notes', $1)", broadcastingMessageForExecutionsAsByteSlice)
		if err != nil {
			log.Println("error sending notification:", err)
		}
	}
}
