package mr

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"regexp"
)

type Coordinator struct {
	// Your definitions here.

}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) GetTask(args *GetTaskArg, reply *GetTaskReply) error {
	reply.TaskType = 0
	reply.TaskId = 1
	reply.TaskFiles = make([]string, 0, 2)
	return nil
}

func (c *Coordinator) TaskFinish(args *TaskFinishArg, reply *TaskFinishReply) error {
	reply.TaskId = 0
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	file, err := ioutil.ReadDir(".")
	if err != nil {
		fmt.Print(err.Error())
	}
	re := regexp.MustCompile(files[0])
	res := make([]string, 0, 10)
	for _, f := range file {
		if !f.IsDir() {
			if re.Match([]byte(f.Name())) {
				res = append(res, f.Name())
			}
		}
	}
	fmt.Println(res)

	c.server()
	return &c
}
