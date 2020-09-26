package main

import (
	"syscall"
	"os"
)

// --- foundation ---

// assert assert expression `bol` with caller `caller` and message `msg`
func assert(bol bool, msg string, caller string) {
	if !bol {
		panic2(caller, msg)
	}
}

// throw panic with error message `s`
func throw(s string) {
	panic(s)
}

var __func__ string = "__func__"

// panic print error message `x` and exit
func panic(x string) {
	var s = "panic: " + x + "\n\n"
	syscall.Write(1, []uint8(s))
	os.Exit(1)
}

// panic2 print error message `x` & caller `caller` and exit
func panic2(caller string, x string) {
	panic("[" + caller + "] " + x)
}