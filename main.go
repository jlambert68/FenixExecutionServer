package main

import (
	_ "net/http/pprof"
)

func main() {
	// *** Profiling ***
	//go func() {
	//	fmt.Println(http.ListenAndServe("localhost:6060", nil))
	//}()
	/*
		config := metrics.DefaultConfig
		config.Username = "jlambert"
		config.Password = "password"
		config.Database = "stats"

		err := metrics.RunCollector(config)
		if err != nil {
			log.Fatalf(err.Error())
		}
	*/

	fenixExecutionServerMain()
}
