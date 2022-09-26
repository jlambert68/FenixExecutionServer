package common_config

import (
	"fmt"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"time"
)

// Generates all "VALUES('xxx', 'yyy')..." for insert statements
func GenerateSQLInsertValues(testdata [][]interface{}) (sqlInsertValuesString string) {

	sqlInsertValuesString = ""

	// Loop over both rows and values
	for rowCounter, rowValues := range testdata {
		if rowCounter == 0 {
			// Only add 'VALUES' for first row
			sqlInsertValuesString = sqlInsertValuesString + "VALUES("
		} else {
			sqlInsertValuesString = sqlInsertValuesString + ",("
		}

		for valueCounter, value := range rowValues {
			switch valueType := value.(type) {

			case bool:
				sqlInsertValuesString = sqlInsertValuesString + fmt.Sprint(value)

			case int, uint32:
				sqlInsertValuesString = sqlInsertValuesString + fmt.Sprint(value)

			case string:

				sqlInsertValuesString = sqlInsertValuesString + "'" + fmt.Sprint(value) + "'"

			case *timestamppb.Timestamp:

				valueAsTimeGrpcTimeStamp := value.(*timestamppb.Timestamp)

				valueAsString := ConvertGrpcTimeStampToStringForDB(valueAsTimeGrpcTimeStamp)

				sqlInsertValuesString = sqlInsertValuesString + "'" + fmt.Sprint(valueAsString) + "'"

			case time.Time:
				valueAsString := GenerateDatetimeFromTimeInputForDB(value.(time.Time))

				sqlInsertValuesString = sqlInsertValuesString + "'" + fmt.Sprint(valueAsString) + "'"

			case fenixExecutionServerGrpcApi.ExecutionPriorityEnum:
				valueAsNumber := value.(fenixExecutionServerGrpcApi.ExecutionPriorityEnum).Number()
				sqlInsertValuesString = sqlInsertValuesString + fmt.Sprint(valueAsNumber)

			default:
				errorId := "33e11bc9-bfc7-4c2f-8440-30f8d9a89ab0"
				log.Fatalf("Unhandled type, %s, with [ErrorId: %s]", valueType, errorId)

			}

			// After the last value then add ')'
			if valueCounter == len(rowValues)-1 {
				sqlInsertValuesString = sqlInsertValuesString + ") "
			} else {
				// Not last value, so Add ','
				sqlInsertValuesString = sqlInsertValuesString + ", "
			}

		}

	}

	return sqlInsertValuesString
}

// Generates incoming values in the following form:  "('monkey', 'tiger'. 'fish')"
func GenerateSQLINArray(testdata []string) (sqlInsertValuesString string) {

	// Create a list with '' as only element if there are no elements in array
	if len(testdata) == 0 {
		sqlInsertValuesString = "('')"

		return sqlInsertValuesString
	}

	sqlInsertValuesString = "("

	// Loop over both rows and values
	for counter, value := range testdata {

		if counter == 0 {
			// Only used for first row
			sqlInsertValuesString = sqlInsertValuesString + "'" + value + "'"

		} else {

			sqlInsertValuesString = sqlInsertValuesString + ", '" + value + "'"
		}
	}

	sqlInsertValuesString = sqlInsertValuesString + ") "

	return sqlInsertValuesString
}

// Generates incoming integer values in the following form:  "(7, 2, 83)"
func GenerateSQLINIntegerArray(testdata []int) (sqlInsertValuesString string) {

	// Create a list with '' as only element if there are no elements in array
	if len(testdata) == 0 {
		sqlInsertValuesString = "()"

		return sqlInsertValuesString
	}

	sqlInsertValuesString = "("

	// Loop over both rows and values
	for counter, value := range testdata {

		if counter == 0 {
			// Only used for first row
			sqlInsertValuesString = sqlInsertValuesString + "'" + fmt.Sprint(value) + "'"

		} else {

			sqlInsertValuesString = sqlInsertValuesString + ", '" + fmt.Sprint(value) + "'"
		}
	}

	sqlInsertValuesString = sqlInsertValuesString + ") "

	return sqlInsertValuesString
}
