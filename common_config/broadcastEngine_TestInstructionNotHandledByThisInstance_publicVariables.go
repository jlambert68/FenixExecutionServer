package common_config

import "time"

var BroadcastEngineMessageChannel BroadcastEngineMessageChannelType

type BroadcastEngineMessageChannelType chan BroadcastingMessageForTestInstructionExecutionsStruct

type BroadcastingMessageForTestInstructionExecutionsStruct struct {
	OriginalMessageCreationTimeStamp string                                           `json:"originalmessagecreationtimestamp"`
	TestInstructionExecutions        []TestInstructionExecutionBroadcastMessageStruct `json:"testinstructionsexecutions"`
}

// TestInstructionExecutionBroadcastMessageStruct
// Structure used when broadcasting that other Execution-instance need need to process message
type TestInstructionExecutionBroadcastMessageStruct struct {
	TestInstructionExecutionUuid                                string                                                          `json:"testinstructionexecutionuuid"`
	TestInstructionExecutionVersion                             string                                                          `json:"testinstructionexecutionversion"`
	TestInstructionExecutionMessageReceivedByWrongExecutionType TestInstructionExecutionMessageReceivedByWrongExecutionTypeType `json:"testinstructionexecutionmessagereceivedbywrongexecutiontype"`
}

// TestInstructionExecutionMessageReceivedByWrongExecutionTypeType
// Type use to specify which message that
type TestInstructionExecutionMessageReceivedByWrongExecutionTypeType int

// The type of message.
// 1) used when storing/reading message in database that the current Execution-instance is NOT the one to handle the message
// 2) used when broadcasting that other Execution-instance need to process message
const (
	FinalTestInstructionExecutionResultMessageType TestInstructionExecutionMessageReceivedByWrongExecutionTypeType = iota
	ProcessTestInstructionExecutionResponseStatus
)

// Message struct used for when saving and loading a 'general' TestInstructionExecution-message that was received by the wrong
// ExecutionServer-instance meaning that TestInstructionExecution is not in memory
type TestInstructionExecutionMessageReceivedByWrongExecutionStruct struct {
	ApplicatonExecutionRuntimeUuid  string
	TestInstructionExecutionUuid    string
	TestInstructionExecutionVersion int
	TimeStamp                       time.Time
	MessageType                     TestInstructionExecutionMessageReceivedByWrongExecutionTypeType
	MessageAsJsonString             string
}
