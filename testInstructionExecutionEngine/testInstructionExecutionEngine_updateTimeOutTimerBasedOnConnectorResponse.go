package testInstructionExecutionEngine

/*
// TODO -remove this because it is not used as of now.....
func (executionEngine *TestInstructionExecutionEngineStruct) updateTimeOutTimerBasedOnConnectorResponse(
	executionTrackNumber int,
	processTestInstructionExecutionResponseStatus *fenixExecutionServerGrpcApi.ProcessTestInstructionExecutionResponseStatus) {

	// *** Check if the TestInstruction is kept in this ExecutionServer-instance ***

	// Create Response channel from TimeOutEngine to get answer if TestInstructionExecution is handled by this instance
	var timeOutResponseChannelForIsThisHandledByThisExecutionInstance common_config.
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceType
	timeOutResponseChannelForIsThisHandledByThisExecutionInstance = make(chan common_config.
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct)

	var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
	tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
		TestCaseExecutionUuid:                   "",
		TestCaseExecutionVersion:                0,
		TestInstructionExecutionUuid:            processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion:         1,
		TestInstructionExecutionCanBeReExecuted: false,
		TimeOutTime:                             time.Time{},
	}

	var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
		TimeOutChannelCommand: common_config.
			TimeOutChannelCommandVerifyIfTestInstructionIsHandledByThisExecutionInstance,
		TimeOutChannelTestInstructionExecutions:                                 tempTimeOutChannelTestInstructionExecutions,
		TimeOutReturnChannelForTimeOutHasOccurred:                               nil,
		TimeOutResponseChannelForDurationUntilTimeOutOccurs:                     nil,
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance: &timeOutResponseChannelForIsThisHandledByThisExecutionInstance,
		SendID:                         "30303c99-11ca-494d-a082-9f0e46bc3364",
		MessageInitiatedFromPubSubSend: false,
	}

	// Calculate Execution Track for TimeOutEngine
	var executionTrackForTimeOutEngine int
	executionTrackForTimeOutEngine = common_config.CalculateExecutionTrackNumber(
		processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid)

	// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution already has TimedOut
	*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackForTimeOutEngine] <- tempTimeOutChannelCommand

	// Response from TimeOutEngine
	var timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct

	// Wait for response from TimeOutEngine
	timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue = <-timeOutResponseChannelForIsThisHandledByThisExecutionInstance

	// Verify that TestInstructionExecution is handled by this Execution-instance
	if timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue.TestInstructionIsHandledByThisExecutionInstance == true {
		// *** TestInstructionExecution is handled by this Execution-instance ***

		// Process only TestInstructionExecutions if we got an AckNack=true as response
		if processTestInstructionExecutionResponseStatus.AckNackResponse.AckNack == true {
			// Create a message with TestInstructionExecution to be sent to TimeOutEngine

			// Set TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=true as respons
			var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
			tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:                   "",
				TestCaseExecutionVersion:                0,
				TestInstructionExecutionUuid:            processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion:         1,
				TestInstructionExecutionCanBeReExecuted: processTestInstructionExecutionResponseStatus.TestInstructionCanBeReExecuted,
				TimeOutTime:                             processTestInstructionExecutionResponseStatus.ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID:                         "9d59fc1b-9b11-4adf-b175-1ebbc60eceae",
				MessageInitiatedFromPubSubSend: false,
			}

			// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
			*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackForTimeOutEngine] <- tempTimeOutChannelCommand
		}

		//Remove Allocation for TimeOut-timer because we got an AckNack=false as response

		// Process only TestInstructionExecutions if we got an AckNack=false as response
		if processTestInstructionExecutionResponseStatus.AckNackResponse.AckNack == false {
			// Create a message with TestInstructionExecution to be sent to TimeOutEngine
			var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
			tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:                   "",
				TestCaseExecutionVersion:                0,
				TestInstructionExecutionUuid:            processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion:         1,
				TestInstructionExecutionCanBeReExecuted: processTestInstructionExecutionResponseStatus.TestInstructionCanBeReExecuted,
				TimeOutTime:                             processTestInstructionExecutionResponseStatus.ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand:                   common_config.TimeOutChannelCommandRemoveAllocationForTestInstructionExecutionToTimeOutTimer,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID:                         "3f5b5990-c7c6-4cba-9136-f90ba7530981",
				MessageInitiatedFromPubSubSend: false,
			}

			// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
			*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackForTimeOutEngine] <- tempTimeOutChannelCommand

		}

	} else {
		// TestInstructionExecution is NOT handled by this Execution-instance
		common_config.Logger.WithFields(logrus.Fields{
			"id": "593f0393-487e-4c9a-b651-6d19f366535b",
			"processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid": processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
		}).Info("TestInstructionExecutionUuid is not handled by this Execution-instance")

		// Create Message to be sent to TestInstructionExecutionEngine
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand: ChannelCommandProcessTestInstructionExecutionResponseStatusIsNotHandledByThisExecutionInstance,
			ProcessTestInstructionExecutionResponseStatus: processTestInstructionExecutionResponseStatus,
		}

		// Send Message to TestInstructionExecutionEngine via channel
		*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

	}
}


*/
