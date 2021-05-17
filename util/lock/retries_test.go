package lock

import (
	"fmt"
	"testing"
)

func TestRetriesTime(t *testing.T) {
	var testTry = NewRetry(0,0,false)
	fmt.Printf("Test 1 res:%v\n", testTry.RetriesTime())
	fmt.Printf("Test 2 res:%v\n", testTry.RetriesTime())
	fmt.Printf("Test 3 res:%v\n", testTry.RetriesTime())
	fmt.Printf("Test 4 res:%v\n", testTry.RetriesTime())
	fmt.Printf("Test 5 res:%v\n", testTry.RetriesTime())
	fmt.Printf("Test 6 res:%v\n", testTry.RetriesTime())

}

