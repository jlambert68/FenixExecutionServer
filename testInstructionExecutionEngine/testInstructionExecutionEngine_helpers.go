package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// ExecutionLocationTypeType
// Definitions for where client and Fenix Server is running
type shouldMessageBeBroadcastedTypeType int

// Constants to be able if 'shouldMessageBeBroadcasted' should look at TestCaseExecutionStatus and/or a TestInstructionExecutionStatus
const (
	shouldMessageBeBroadcasted_ThisIsATestCaseExecution shouldMessageBeBroadcastedTypeType = iota
	shouldMessageBeBroadcasted_ThisIsATestInstructionExecution
)

// Should a TestCaseExecutionStatus and/or a TestInstructionExecutionStatus
func (executionEngine *TestInstructionExecutionEngineStruct) shouldMessageBeBroadcasted(
	shouldMessageBeBroadcastedType shouldMessageBeBroadcastedTypeType,
	executionStatusReportLevel fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum,
	testCaseExecutionStatus fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum,
	testInstructionExecutionStatus fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum) (
	messageShouldBeBroadcasted bool) {

	// Process TestCasesExecutionStatus
	if shouldMessageBeBroadcastedType == shouldMessageBeBroadcasted_ThisIsATestCaseExecution {

		switch executionStatusReportLevel {
		// Do no reporting at all
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_NO_STATUS_REPORTING_FOR_EXECUTION:
			return false

		// All changes in execution status for TestInstructions and TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ALL_STATUS_CHANGES_ON_EXECUTIONS:
			return true

		// Only changes into execution end status for TestInstructions and TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ONLY_ALL_END_STATUS_CHANGES_ON_EXECUTIONS:
			switch testCaseExecutionStatus {

			case fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_INITIATED,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_CONTROLLED_INTERRUPTION,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_NOT_OK,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_NOT_OK_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_UNEXPECTED_INTERRUPTION,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN:

				return true

			default:
				return false
			}

		// Only changes in execution status for TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ONLY_ALL_STATUS_CHANGES_ON_TESTCASE_EXECUTIONS:
			return true

		// Only changes into execution end status for TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ONLY_ALL_END_STATUS_CHANGES_ON_TESTCASE_EXECUTIONS:
			switch testCaseExecutionStatus {

			case fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_INITIATED,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_CONTROLLED_INTERRUPTION,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_NOT_OK,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_NOT_OK_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_UNEXPECTED_INTERRUPTION,
				fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN:

				return true

			default:
				return false
			}

		default:
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                         "355344d9-ad03-474d-af6e-d00ebb3f7ae5",
				"executionStatusReportLevel": executionStatusReportLevel,
			}).Error("Unhandled 'executionStatusReportLevel'")

			return false
		}
	}

	// Process TestInstructionExecutionStatus
	if shouldMessageBeBroadcastedType == shouldMessageBeBroadcasted_ThisIsATestInstructionExecution {

		switch executionStatusReportLevel {
		// Do no reporting at all
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_NO_STATUS_REPORTING_FOR_EXECUTION:
			return false

		// All changes in execution status for TestInstructions and TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ALL_STATUS_CHANGES_ON_EXECUTIONS:
			return true

		// Only changes into execution end status for TestInstructions and TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ONLY_ALL_END_STATUS_CHANGES_ON_EXECUTIONS:
			switch testInstructionExecutionStatus {

			case fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_CONTROLLED_INTERRUPTION,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_FINISHED_OK,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_FINISHED_OK_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_FINISHED_NOT_OK,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_FINISHED_NOT_OK_CAN_BE_RERUN,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION,
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN:

				return true

			default:
				return false
			}

		// Only changes in execution status for TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ONLY_ALL_STATUS_CHANGES_ON_TESTCASE_EXECUTIONS:
			return false

		// Only changes into execution end status for TestCases are reported
		case fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum_REPORT_ONLY_ALL_END_STATUS_CHANGES_ON_TESTCASE_EXECUTIONS:
			return false

		default:
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                         "355344d9-ad03-474d-af6e-d00ebb3f7ae5",
				"executionStatusReportLevel": executionStatusReportLevel,
			}).Error("Unhandled 'executionStatusReportLevel'")

			return false
		}
	}

	return messageShouldBeBroadcasted
}
