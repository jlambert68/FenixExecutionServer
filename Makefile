RunGrpcGui:
	cd ~/egen_kod/go/go_workspace/src/jlambert/grpcui/standalone && grpcui -plaintext localhost:6672

filename :=
filenamePartFirst := FenixGuiCrossBuild_
filenamePartLast := .exe
filenamePartFirstLinux=FenixBuild
filenamePartLinuxLast = .Linux
datetime := `date +'%y%m%d_%H%M%S'`

GenerateDateTime:
	$(eval fileName := $(filenamePartFirst)$(datetime)$(filenamePartLast))

	echo $(fileName)

BuildExeForLinux:
	$(eval fileName := $(filenamePartFirstLinux)$(datetime)$(filenamePartLinuxLast))
	GOOD=linux GOARCH=amd64 go build -o $(fileName) -ldflags=" -X 'main.BuildVariableApplicationGrpcPort=6668' -X 'main.BuildVariableExecutionLocationForFenixGuiExecutionServer=LOCALHOST_NODOCKER' -X 'main.BuildVariableExecutionLocationForFenixGuiTestCaseBuilderServer=GCP' -X 'main.BuildVariableExecutionLocationForThisApplication=LOCALHOST_NODOCKER' -X 'main.BuildVariableFenixGuiExecutionServerAddress=127.0.0.1' -X 'main.BuildVariableFenixGuiExecutionServerPort=6669' -X 'main.BuildVariableFenixGuiTestCaseBuilderServerAddress=fenixguitestcasebuilderserver-must-be-logged-in-nwxrrpoxea-lz.a.run.app' -X 'main.BuildVariableFenixGuiTestCaseBuilderServerPort=443' -X 'main.BuildVariableFYNE_SCALE=0.6' -X 'main.BuildVariableGCPAuthentication=true' -X 'main.BuildVariableRunAsTrayApplication=NO' -X 'main.BuildVariableUseServiceAccountForGuiExecutionServer=true' -X 'main.BuildVariableUseServiceAccountForGuiTestCaseBuilderServer=true" .

