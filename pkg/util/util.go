package util

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
)

// HandleSigterm Ctrl+C or most other means of "controlled" shutdown gracefully. Invokes the supplied func before exiting.
func HandleSigterm(handleExit func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		handleExit()
		os.Exit(1)
	}()
}

// RemoveDuplicatesForSlice remove duplicate item from slice
func RemoveDuplicatesForSlice(slice ...interface{}) []string {
	encountered := map[string]bool{}
	for _, v := range slice {
		encountered[v.(string)] = true
	}
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
