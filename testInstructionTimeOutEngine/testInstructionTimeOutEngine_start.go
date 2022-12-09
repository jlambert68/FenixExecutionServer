package testInstructionTimeOutEngine

// InitiateTestInstructionExecutionTimeOutEngineChannelReader
// Initiate the channel reader which is used for sending commands to TimeOutEngine for TestInstructionExecutions
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) InitiateTestInstructionExecutionTimeOutEngineChannelReader() {

	// Initiate channels and maps
	timeOutMap = make(map[string]*timeOutMapStruct)

	TimeOutChannelEngineCommandChannel = make(chan TimeOutChannelCommandStruct, TimeOutChannelSize)

	// Start Channel reader
	go testInstructionExecutionTimeOutEngineObject.startTimeOutChannelReader()

}
