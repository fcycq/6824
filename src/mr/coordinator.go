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

type Task struct {
	taskInfo GetTaskReply
	time     time.Time
}

type Coordinator struct {
	// Your definitions here.
	sync.Mutex

	taskId int

	MapTask        []string
	PendingMapTask map[int]Task

	ReduceTask        []string
	PendingReduceTask map[int]Task

	finishFlag bool
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
		taskRecorder := c.getMapTask(reply)
		c.PendingMapTask[taskRecorder.taskInfo.TaskId] = taskRecorder
	} else if len(c.ReduceTask) > 2 {
		taskRecorder := c.getReduceTask(reply)
		c.PendingReduceTask[taskRecorder.taskInfo.TaskId] = taskRecorder
	} else {
		if len(c.PendingMapTask) > 0 || len(c.PendingReduceTask) > 0 {
			c.waitTask(reply)
		} else {
			if len(c.PendingReduceTask) == 1 {
				c.finishFlag = true
				c.noTaskLeft(reply)
			} else {
				glog.Fatal("task inconsistent")
			}
		}
		// 判断结束，或者等待
	}

	return nil
}

func (c *Coordinator) getMapTask(reply *GetTaskReply) (t Task) {
	reply.TaskType = 2
	reply.TaskId = c.AllocTaskId()
	reply.TaskFiles = make([]string, 1)
	reply.TaskFiles[0] = c.MapTask[0]

	c.MapTask = c.MapTask[1:]

	t.taskInfo = *reply
	t.time = time.Now()
	return

}

func (c *Coordinator) getReduceTask(reply *GetTaskReply) (t Task) {
	reply.TaskType = 3
	reply.TaskFiles = make([]string, 2, 2)
	reply.TaskFiles[0] = c.ReduceTask[0]
	reply.TaskFiles[1] = c.ReduceTask[1]

	c.ReduceTask = c.ReduceTask[2:]

	t.taskInfo = *reply
	t.time = time.Now()
	return

}

func (c *Coordinator) waitTask(reply *GetTaskReply) {
	reply.TaskType = 1
}

func (c *Coordinator) noTaskLeft(reply *GetTaskReply) {
	reply.TaskType = 0
}

func (c *Coordinator) TaskFinish(args *TaskFinishArg, reply *TaskFinishReply) error {
	if c.TryLock() {
		defer c.Unlock()
	}

	taskId := args.TaskId
	if info, found := c.PendingMapTask[taskId]; found {
		delete(c.PendingMapTask, taskId)
		if args.TaskSuccess {
			c.ReduceTask = append(c.ReduceTask, args.ResultName)
		} else {
			c.MapTask = append(c.MapTask, info.taskInfo.TaskFiles...)
		}
	} else if info, found := c.PendingReduceTask[taskId]; found {
		delete(c.PendingReduceTask, taskId)
		if args.TaskSuccess {
			c.ReduceTask = append(c.ReduceTask, args.ResultName)
		} else {
			c.ReduceTask = append(c.ReduceTask, info.taskInfo.TaskFiles...)
		}
	}
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
	c.Lock()
	defer c.Unlock()
	ret = c.finishFlag

	return ret
}

func (c *Coordinator) scanTask() {
	for {
		time.Sleep(10 * time.Second)
		c.Lock()
		for taskId, task := range c.PendingMapTask {
			if time.Since(task.time) > 20*time.Second {
				finishArg := TaskFinishArg{}
				finishArg.TaskId = taskId
				finishArg.TaskSuccess = false
				finishArg.ResultName = ""

				finishReply := TaskFinishReply{}

				c.TaskFinish(&finishArg, &finishReply)
			}
		}
		c.Unlock()
	}
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.finishFlag = false
	c.PendingMapTask = make(map[int]Task)
	c.PendingReduceTask = make(map[int]Task)
	res := filtInputFile(files[0])

	glog.Infof("%d file in total", len(res))

	c.MapTask = append(c.MapTask, res...)

	go c.scanTask()
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
