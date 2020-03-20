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

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
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

func openFile(f string) (*os.File){
	file, err := os.Open(f) //open the file to process
	check(err)
	return file
}

func getSize(file *os.File) (int){
	fileinfo, err := file.Stat()
	check(err)
	filesize := int(fileinfo.Size())
	return filesize
}

func divideFile(filesize int, n int) (int, []chunk){
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

// `ChannelSum()` spawns `n` goroutines that store their intermediate sums locally, then pass the result back through a channel.
func main() {
	start := time.Now()
	arg0, arg1, n := receiveArguments()
	// n := runtime.GOMAXPROCS(0)
	// src := "$2 * $3 > 5 { emp = emp + 1 } END {print emp}"
	prog, err := parser.ParseProgram([]byte(arg0), nil)
	check(err)
	var sm sync.Map
	res := make(chan int)
	// arg1 := flag.Arg(1)
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
			// fmt.Printf("\nbytestream to string: %v , %d\n", string(buffer), i)

			endingOffset := 0
			for j := chunk.buffsize - 1; j > 0; j-- { //going backward on the last line to find where it starts
				if string(buffer[j]) == "\n" {
					endingOffset = j + 1
					break
				}
			}

			startingOffset := 0
			for k := 0; k < chunk.buffsize; k++ { //going forward on the first line to find where it ends
				if string(buffer[k]) == "\n" {
					startingOffset = k
					break
				}
			}

			_ = startingOffset
			_ = endingOffset

			num := strconv.Itoa(i + 1)
			str := "start" + num
			ending := "end" + num
			sm.Store(str, string(buffer[:startingOffset]))
			sm.Store(ending, string(buffer[endingOffset:]))

			// fmt.Printf("\nbytestream to string new: %v, %d\n", string(buffer[startingOffset:endingOffset]), i)

			config := &interp.Config{
				Stdin: bytes.NewReader([]byte(string(buffer[startingOffset:endingOffset]))),
				Vars:  []string{"OFS", ":"},
			}
			_, err = interp.ExecProgram(prog, config)
			check(err)

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

	// firstLine, _ := sm.Load("start" + strconv.Itoa(1))
	// fmt.Printf("line is %s \n", firstLine)
	// for i := 1; i < n; i++ {
	// 	stringResultEnd, _ := sm.Load("end" + strconv.Itoa(i))
	// 	stringResultStart, _ := sm.Load("start" + strconv.Itoa(i+1))
	// 	line := stringResultEnd.(string) + stringResultStart.(string)
	// 	fmt.Printf("line is %s \n", line)
	// }

	elapsed := time.Since(start)
	fmt.Printf("\nTime elapsed for channel sum %s\n", elapsed)
}
