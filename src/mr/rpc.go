package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type GetTaskArg struct {
}

type GetTaskReply struct {
	// 0 for no task, 1 for wait, 2 for Map, 3 for reduce
	TaskType  int
	TaskId    int
	TaskFiles []string

	SpecifiedResultFileName string
}

type TaskFinishArg struct {
	TaskId int
}

type TaskFinishReply struct {
	TaskId int
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
