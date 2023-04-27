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
	"sync"
	"time"

	"github.com/golang/glog"
)

type MapTask struct {
	file      string
	taskId    int
	startTime time.Time
}

type ReduceTask struct {
	files     []string
	taskId    int
	startTime time.Time
}

type Coordinator struct {
	// Your definitions here.
	sync.Mutex

	taskId int

	MapTask        []string
	PendingMapTask map[int]MapTask

	ReduceTask        []string
	PendingReduceTask map[int]ReduceTask
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) AllocTaskId() int {
	shouldUnlock := c.TryLock()
	t := c.taskId
	c.taskId += 1
	if shouldUnlock {
		c.Unlock()
	}
	return t
}

func (c *Coordinator) GetTask(args *GetTaskArg, reply *GetTaskReply) error {
	c.Lock()
	defer c.Unlock()

	// 应该优化一下
	if len(c.MapTask) > 0 {
		c.getMapTask(reply)
	} else if len(c.ReduceTask) > 2 {
		c.getReduceTask(reply)
	} else {
		// 判断结束，或者等待
	}
	reply.TaskType = 0
	reply.TaskId = c.taskId
	c.taskId += c.AllocTaskId()
	reply.TaskFiles = make([]string, 0, 2)
	return nil
}

func (c *Coordinator) getMapTask(reply *GetTaskReply) {

}

func (c *Coordinator) getReduceTask(reply *GetTaskReply) {

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
	res := filtInputFile(files[0])

	glog.Infof("%d file in total", len(res))

	c.MapTask = append(c.MapTask, res...)

	c.server()

	return &c
}

func filtInputFile(s string) []string {
	res := make([]string, 0, 10)
	file, err := ioutil.ReadDir(".")
	if err != nil {
		fmt.Print(err.Error())
	}
	re := regexp.MustCompile(s)
	for _, f := range file {
		if !f.IsDir() {
			if re.Match([]byte(f.Name())) {
				res = append(res, f.Name())
			}
		}
	}
	return res
}
