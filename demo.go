package main

import (
	"fmt"
	"os"
	"flag"
	"strconv"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	var end int
	var bytes_to_read int
	var data string
	o := int64(0)
	bytes_to_read = 0
	end = 0
	flag.Parse()
	s1 := flag.Arg(0)
	number_of_threads, err := strconv.Atoi(s1)
	check(err)
	file := flag.Arg(1)
	f, err := os.Open(file)
	check(err)
	file_size, err := f.Stat()
	check(err)
	size:= file_size.Size()
	fmt.Printf("the file size is: %d\n", size)
	default_size := int(size/int64(number_of_threads))
	for thread :=0; thread < number_of_threads; thread++ {
		//fmt.Println(bytes_to_read - end)
		if thread == number_of_threads - 1 {
				bytes_to_read = default_size + (bytes_to_read - end) + 2
		} else {
			bytes_to_read = default_size + (bytes_to_read - end)
		}
		b := make([]byte, bytes_to_read) //the byte length that gets handled by every thread
		n, err := f.Read(b)
		check(err)
		fmt.Printf("\n\n%d bytes @ %d\n", n, o)
		//fmt.Printf("%v\n", string(b[:n]))
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
		//fmt.Printf("%d \n \n \n",end)
		o, err = f.Seek(o+int64(end),0)
		check(err)
	}
}
