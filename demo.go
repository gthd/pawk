package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	start := time.Now()
	number_of_threads := runtime.GOMAXPROCS(0)
	var data string
	o := int64(0)
	bytes_to_read := 0
	end := 0
	f, err := os.Open("demo.txt")
	check(err)
	defer f.Close()
	file_size, err := f.Stat()
	check(err)
	size:= file_size.Size()
	fmt.Printf("the file size is: %d\n", size)
	default_size := int(size/int64(number_of_threads))
	for thread :=0; thread < number_of_threads; thread++ {
		if thread == number_of_threads - 1 {
				bytes_to_read = default_size + (bytes_to_read - end) + 2
		} else {
			bytes_to_read = default_size + (bytes_to_read - end)
		}
		b := make([]byte, bytes_to_read) //the byte length that gets handled by every thread
		n, err := f.Read(b)
		check(err)
		fmt.Printf("\n\n%d bytes @ %d\n", n, o)
		for i :=bytes_to_read-1; i > 0; i-- {
			if string(b[i]) == "\n" {
				end = i
				break
			}
		}
		if thread > 0 {
			data = string(b[1:end])
		} else {
			data = string(b[:end])
		}
		fmt.Printf("%v\n", data)
		o, err = f.Seek(o+int64(end),0)
		check(err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Time it took %s", elapsed)
}
