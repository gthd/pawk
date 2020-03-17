package main

import (
	"fmt"
	"os"
	"sync"
	"strconv"
	"runtime"
	"time"
	"bytes"

  "github.com/benhoyt/goawk/parser"
	"github.com/benhoyt/goawk/interp"
)

type chunk struct {
	buffsize int
	offset int64
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// `ChannelSum()` spawns `n` goroutines that store their intermediate sums locally, then pass the result back through a channel.
func main() {
	start := time.Now()
	// n := 4
  n := runtime.GOMAXPROCS(0)
	// A channel of ints will collect all intermediate sums.
	src := "$2 * $3 > 5 { emp = emp + 1 } END {print emp}"
	prog, err := parser.ParseProgram([]byte(src), nil)
	check(err)

	var sm sync.Map
	res := make(chan int)
	file, err := os.Open("emp.data") //open the file to process
	check(err)
	defer file.Close()
	fileinfo, err := file.Stat()
	check(err)
	filesize := int(fileinfo.Size())
	fmt.Println(filesize)
	chunksizes := make([]chunk, n)
	BufferSize := int(filesize/int(n))
	for i := 0; i < n; i++ {
		chunksizes[i].buffsize = BufferSize
		chunksizes[i].offset = int64(BufferSize*i)
	}
	if remainder := filesize % BufferSize; remainder != 0 {
		c := chunk{buffsize: remainder, offset: int64(n * BufferSize)}
		n++ //then the last chunk is not processed in parallel, as the go-routines are one more than the cores available
		chunksizes = append(chunksizes, c)
	}
	for i := 0; i < n; i++ {
		go func(chunksizes []chunk, i int, r chan<- int) {
			chunk := chunksizes[i]
			buffer := make([]byte, chunk.buffsize)
			bytesread, err := file.ReadAt(buffer, chunk.offset)
			if err != nil {
		      fmt.Println(err)
		      return
		  }
			_ = bytesread
			// fmt.Printf("\nbytestream to string: %v , %d\n", string(buffer), i)

			ending_offset := 0
			for i :=chunk.buffsize-1; i > 0; i-- { //going backward on the last line to find where it starts
				if string(buffer[i]) == "\n" {
					ending_offset = i + 1
					break
				}
			}

			starting_offset := 0
			for i :=0; i < chunk.buffsize; i++ { //going forward on the first line to find where it ends
				if string(buffer[i]) == "\n" {
					starting_offset = i
					break
				}
			}

			_ = starting_offset
			_ = ending_offset

			num := strconv.Itoa(i+1)
			str := "start" + num
			ending := "end" + num
			sm.Store(str, string(buffer[:starting_offset]))
			sm.Store(ending, string(buffer[ending_offset:]))

			// fmt.Printf("\nbytestream to string new: %v, %d\n", string(buffer[starting_offset:ending_offset]), i)

			config := &interp.Config{
		      Stdin: bytes.NewReader([]byte(string(buffer[starting_offset:ending_offset]))),
		      Vars:  []string{"OFS", ":"},
		  }
		  _, err = interp.ExecProgram(prog, config)
			if err != nil {
		      fmt.Println(err)
		      return
		  }

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

	first_line, _ := sm.Load("start"+strconv.Itoa(1))
	fmt.Printf("line is %s \n", first_line)
	for i := 1; i < n; i++ {
		string_result_end , _ := sm.Load("end"+strconv.Itoa(i))
		string_result_start , _ := sm.Load("start"+strconv.Itoa(i+1))
		line := string_result_end.(string) + string_result_start.(string)
		fmt.Printf("line is %s \n", line)
	}

	elapsed := time.Since(start)
	fmt.Printf("\nTime elapsed for channel sum %s\n", elapsed)
}
