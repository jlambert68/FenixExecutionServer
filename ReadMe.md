MIT License

Copyright (c) 2024 Jonas Lambert

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

***

# Fenix Inception

## ExecutionServer
ExecutionWorker has the responsibility to break down a TestCaseExecution into its separate TestInstructionExecutions. Each TestInstructionExecutions are then sent towards responsible Connector that can execute the specific TestInstruction.

![Fenix Inception - ExecutionServer](./Documentation/FenixInception-Overview-NonDetailed-ExecutionServer.png "Fenix Inception - ExecutionServer")

The following environment variable is needed for ExecutionServer to be able to run.

| Environment variable                                                 | Example value                                                        | comment                                                                        |
|----------------------------------------------------------------------|----------------------------------------------------------------------|--------------------------------------------------------------------------------|
| DB_HOST                                                              | 127.0.0.1                                                            |                                                                                |
| DB_NAME                                                              | fenix_gcp_database                                                   |                                                                                |
| DB_PASS                                                              | database password                                                    |                                                                                |
| DB_POOL_MAX_CONNECTIONS                                              | 4                                                                    | The number of connections towards the database that the ExectionServer can use |
| DB_PORT                                                              | 5432                                                                 |                                                                                |
| DB_SCHEMA                                                            | Not used                                                             |                                                                                |
| DB_USER                                                              | postgres                                                             |                                                                                |
| ExecutionLocationForFenixTestExecutionServer                         | GCP                                                                  |                                                                                |
| ExecutionLocationForWorker                                           | GCP                                                                  |                                                                                |
| ExecutionStatusPubSubTopic                                           | notUsed                                                              |                                                                                |
| FenixExecutionExecutionServerPort                                    | 6672                                                                 |                                                                                |
| FenixExecutionWorkerServerAddress                                    | fenixexecutionworkerserver-must-be-logged-in-ffafweeerg-lz.a.run.app |                                                                                |
| FenixExecutionWorkerServerPort                                       | 443                                                                  |                                                                                |
| LocalServiceAccountPath                                              | #                                                                    |                                                                                |
| LogAllSQLs                                                           | true                                                                 | Used when debugging to log all full SQLs                                       |
| LoggingLevel                                                         | DebugLevel                                                           |                                                                                |
| ProjectId                                                            | mycloud-run-project                                                  |                                                                                |
| SleepTimeInSecondsBeforeClaimingTestInstructionExecutionFromDatabase | 15                                                                   |                                                                                |
| UsePubSubWhenSendingExecutionStatusToGuiExecutionServer              | false                                                                |                                                                                |
| WorkerIsUsingPubSubWhenSendingTestInstructionExecutions              | true                                                                 |                                                                                |

