package broadcastingEngine

import (
	"context"
	"encoding/json"
	"fmt"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"log"
)

var BroadcastEngineMessageChannel BroadcastEngineMessageChannelType

type BroadcastEngineMessageChannelType chan BroadcastingMessageForExecutionsStruct

type BroadcastingMessageForExecutionsStruct struct {
	BroadcastTimeStamp        string                           `json:"timestamp"`
	TestCaseExecutions        []TestCaseExecutionStruct        `json:"testcaseexecutions"`
	TestInstructionExecutions []TestInstructionExecutionStruct `json:"testinstructionexecutions"`
}

type TestCaseExecutionStruct struct {
	TestCaseExecutionUuid    string `json:"testcaseexecutionuuid"`
	TestCaseExecutionVersion int    `json:"testcaseexecutionversion"`
	TestCaseExecutionStatus  string `json:"testcaseexecutionstatus"`
}

type TestInstructionExecutionStruct struct {
	TestInstructionExecutionUuid   string `json:"testinstructionuuid"`
	TestInstructionExecutionStatus string `json:"testinstructionstatus"`
}

var err error

func InitiateAndStartBroadcastNotifyEngine() {

	BroadcastEngineMessageChannel = make(chan BroadcastingMessageForExecutionsStruct, 10)
	var broadcastingMessageForExecutions BroadcastingMessageForExecutionsStruct
	var broadcastingMessageForExecutionsAsByteSlice []byte
	var broadcastingMessageForExecutionsAsByteSliceAsString string
	var err error

	for {

		broadcastingMessageForExecutions = <-BroadcastEngineMessageChannel

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
