package mr

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

	for {

		reply := CallGetTask()

		printTask(reply)
		err := error(nil)
		if reply.TaskType == 2 {
			err = Map(mapf, reply)
		} else if reply.TaskType == 3 {
			err = Reduce(reducef, reply)
		} else if reply.TaskType == 1 {
			time.Sleep(1 * time.Second)
			continue
		}

		CallFinish(reply, err)

	}

}

func printTask(reply *GetTaskReply) {
	fmt.Printf("task type %d, input file %+v, output filename %s", reply.TaskType, reply.TaskFiles, genOutputName(reply))
}

func genOutputName(reply *GetTaskReply) string {
	if reply.SpecifiedResultFileName == "" {
		return "temp_" + strconv.Itoa(reply.TaskId)
	} else {
		return reply.SpecifiedResultFileName
	}
}

func Map(mapf func(string, string) []KeyValue, reply *GetTaskReply) error {

	fContent, err := ioutil.ReadFile(reply.TaskFiles[0])
	if err != nil {
		glog.Warningf("fail to read from %s", reply.TaskFiles[0])
		return err
	}

	words := mapf("", string(fContent))
	count := make(map[string]int)
	for _, w := range words {
		count[w.Key] += 1
	}
	sortedKey := make([]string, 0, len(count)+1)
	for k, _ := range count {
		sortedKey = append(sortedKey, k)
	}
	sort.Slice(sortedKey, func(i, j int) bool { return strings.ToLower(sortedKey[i]) < strings.ToLower(sortedKey[j]) })

	f, err := os.OpenFile(genOutputName(reply), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		glog.Warningf("fail to open file %s", genOutputName(reply))
		return err
	}
	for _, t := range sortedKey {
		line := t + " " + strconv.Itoa(count[t]) + "\n"
		f.Write(([]byte)(line))
	}
	f.Close()
	return nil
}

func Reduce(reducef func(string, []string) string, reply *GetTaskReply) error {
	wordCountMap := make(map[string]int)

	for _, inputFile := range reply.TaskFiles {
		file, err := os.Open(inputFile)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			words := strings.Fields(line)
			if len(words) != 2 {
				continue
			}
			word := words[0]
			count, err := strconv.Atoi(words[1])
			if err != nil {
				continue
			}
			wordCountMap[word] += count
		}

		if err := file.Close(); err != nil {
			return err
		}
	}

	fileName := ""
	if reply.SpecifiedResultFileName == "" {
		fileName = genOutputName(reply)
	} else {
		fileName = reply.SpecifiedResultFileName
	}
	output, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer output.Close()

	sortedKey := make([]string, 0, len(wordCountMap)+1)
	for k, _ := range wordCountMap {
		sortedKey = append(sortedKey, k)
	}
	sort.Slice(sortedKey, func(i, j int) bool { return strings.ToLower(sortedKey[i]) < strings.ToLower(sortedKey[j]) })

	writer := bufio.NewWriter(output)
	for _, key := range sortedKey {
		line := fmt.Sprintf("%s %d\n", key, wordCountMap[key])
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	return nil
	return nil
}

func CallGetTask() *GetTaskReply {
	args := GetTaskArg{}
	reply := GetTaskReply{}
	ok := call("Coordinator.GetTask", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.TaskId)
	} else {
		fmt.Printf("call failed!\n")
	}
	return &reply
}

func CallFinish(task *GetTaskReply, err error) {
	args := TaskFinishArg{}
	reply := TaskFinishReply{}

	args.TaskId = task.TaskId
	if err != nil {
		args.TaskSuccess = false
	} else {
		args.TaskSuccess = true
	}
	args.ResultName = genOutputName(task)

	ok := call("Coordinator.TaskFinish", &args, &reply)
	if !ok {
		glog.Warning("fail to call TaskFinish")
	}

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
