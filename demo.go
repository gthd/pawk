// Imports and globals
package main

import (
	"fmt"
	"os"
	// "runtime"
	"time"
)

const (
	limit = 10000000000
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
	n := 4
  // n := runtime.GOMAXPROCS(0)
	// A channel of ints will collect all intermediate sums.
	res := make(chan int)
	file, err := os.Open("demo2.txt") //open the file to process
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
			check(err)
			_ = bytesread
			fmt.Printf("\nbytestream to string: %v \n", string(buffer))

			ending_offset := 0
			for i :=chunk.buffsize-1; i > 0; i-- { //going backward on the last line to find where it starts
				if string(buffer[i]) == "\n" {
					ending_offset = i
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

			fmt.Printf("\nbytestream to string new: %v \n", string(buffer[starting_offset:ending_offset]))

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
	elapsed := time.Since(start)
	fmt.Printf("Time elapsed for channel sum %s \n", elapsed)
}
