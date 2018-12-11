package util

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"reflect"
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

// Contain 判断obj是否在target中，target支持的类型arrary,slice,map
// https://www.cnblogs.com/zsbfree/archive/2013/05/23/3094993.html
func Contain(obj interface{}, target interface{}) bool {
    targetValue := reflect.ValueOf(target)
    switch reflect.TypeOf(target).Kind() {
    case reflect.Slice, reflect.Array:
        for i := 0; i < targetValue.Len(); i++ {
            if targetValue.Index(i).Interface() == obj {
                return true
            }
        }
    case reflect.Map:
        if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
            return true
        }
    }

    return false
}
