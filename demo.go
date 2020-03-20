package main

import (
	"bytes"
	"fmt"
	"os"
	// "runtime"
	"flag"
	"strconv"
	"sync"
	"time"

	"github.com/gthd/goawk/interp"
	"github.com/gthd/goawk/parser"
)

type chunk struct {
	buffsize int
	offset   int64
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func receiveArguments() (string, string, int) {
	flag.Parse()
	argument0 := flag.Arg(0)
	argument1 := flag.Arg(1)
	argument2 := flag.Arg(2)
	numberOfThreads, err := strconv.Atoi(argument2)
	check(err)
	return argument0, argument1, numberOfThreads
}

func openFile(f string) *os.File {
	file, err := os.Open(f) //open the file to process
	check(err)
	return file
}

func getSize(file *os.File) int {
	fileinfo, err := file.Stat()
	check(err)
	filesize := int(fileinfo.Size())
	return filesize
}

func divideFile(filesize int, n int) (int, []chunk) {
	chunksizes := make([]chunk, n)
	BufferSize := int(filesize / int(n))
	for i := 0; i < n; i++ {
		chunksizes[i].buffsize = BufferSize
		chunksizes[i].offset = int64(BufferSize * i)
	}
	if remainder := filesize % BufferSize; remainder != 0 {
		c := chunk{buffsize: remainder, offset: int64(n * BufferSize)}
		n++ //then the last chunk is not processed in parallel, as the go-routines are one more than the cores available
		chunksizes = append(chunksizes, c)
	}
	return n, chunksizes
}

// func readFile(chunksize []chunk, buffsize int, file *os.File) ([]byte){
// 	chunk := chunksize
// 	buffer := make([]byte, buffsize)
// 	_, err := file.ReadAt(buffer, chunk.offset)
// 	check(err)
// 	return buffer
// }

func endOffset(buffsize int, buffer []byte) int {
	endingOffset := 0
	for j := buffsize - 1; j > 0; j-- { //going backward on the last line to find where it starts
		if string(buffer[j]) == "\n" {
			endingOffset = j + 1
			break
		}
	}
	return endingOffset
}

func startOffset(buffsize int, buffer []byte) int {
	startingOffset := 0
	for k := 0; k < buffsize; k++ { //going forward on the first line to find where it ends
		if string(buffer[k]) == "\n" {
			startingOffset = k
			break
		}
	}
	return startingOffset
}

func stichLines(sm sync.Map, n int) {
	firstLine, _ := sm.Load("start" + strconv.Itoa(1))
	fmt.Printf("line is %s \n", firstLine)
	for i := 1; i < n; i++ {
		stringResultEnd, _ := sm.Load("end" + strconv.Itoa(i))
		stringResultStart, _ := sm.Load("start" + strconv.Itoa(i+1))
		line := stringResultEnd.(string) + stringResultStart.(string)
		fmt.Printf("line is %s \n", line)
	}
}

func goAwk(buffer []byte, startingOffset int, endingOffset int, prog *parser.Program) {
	config := &interp.Config{
		Stdin: bytes.NewReader([]byte(string(buffer[startingOffset:endingOffset]))),
		Vars:  []string{"OFS", ":"},
	}
	_, err := interp.ExecProgram(prog, config)
	check(err)
}

// `ChannelSum()` spawns `n` goroutines that store their intermediate sums locally, then pass the result back through a channel.
func main() {
	start := time.Now()
	arg0, arg1, n := receiveArguments()
	// n := runtime.GOMAXPROCS(0)
	// src := "$2 * $3 > 5 { emp = emp + 1 } END {print emp}"
	prog, err, varTypes := parser.ParseProgram([]byte(arg0), nil)
	fmt.Println(prog)
	fmt.Println(varTypes)
	check(err)
	var sm sync.Map
	res := make(chan int)
	file := openFile(arg1)
	defer file.Close()
	filesize := getSize(file)
	n, chunksizes := divideFile(filesize, n)

	for i := 0; i < n; i++ {
		go func(chunksizes []chunk, i int, r chan<- int) {
			chunk := chunksizes[i]
			buffer := make([]byte, chunk.buffsize)
			_, err := file.ReadAt(buffer, chunk.offset)
			check(err)

			endingOffset := endOffset(chunk.buffsize, buffer)
			startingOffset := startOffset(chunk.buffsize, buffer)

			// Have to change
			num := strconv.Itoa(i + 1)
			str := "start" + num
			ending := "end" + num
			sm.Store(str, string(buffer[:startingOffset]))
			sm.Store(ending, string(buffer[endingOffset:]))

			// fmt.Printf("\nbytestream to string new: %v, %d\n", string(buffer[startingOffset:endingOffset]), i)
			goAwk(buffer, startingOffset, endingOffset, prog)

			// This local variable replaces the global slice.
			sum := 0

			//Should contain the processing that the program should do on each chunk

			//

			// Channel receives the result from processing each chunk
			r <- sum
			// Call the goroutine and pass the parameters of each chunk, the CPU core index and the channel that will receive the results.
		}(chunksizes, i, res)
	}

	sum := 0
	for i := 0; i < n; i++ {
		// Read the value from each channel and add it to `sum`.
		//
		//  Synchronization of all cores through the blocking nature of channels.
		sum += <-res
	}

	stichLines(sm, n)

	elapsed := time.Since(start)
	fmt.Printf("\nTime elapsed for channel sum %s\n", elapsed)
}
