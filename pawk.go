// Copyright 2020 Georgios Theodorou
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/gthd/goawk/interp"
	"github.com/gthd/goawk/parser"
)

type chunk struct {
	buff []byte
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func receiveArguments() (string, int, string, bool) {
	numberOfThreads := runtime.GOMAXPROCS(0)
	if len(os.Args) > 1 {
		argument0 := os.Args[1]
		receivedFile := true
		if argument0 == "-f" { //then the awk command is inside a file so that we read the file name as an argument
			argument1 := os.Args[2]
			if argument1 == "-n" {
				argument2 := os.Args[3] //awk command file
				argument3 := os.Args[4] // file to process
				return argument2, numberOfThreads, argument3, receivedFile
			}
			argument2 := os.Args[3] //threads
			numberOfThreads, err := strconv.Atoi(argument2)
			check(err)
			argument3 := os.Args[4] // file to process
			return argument1, numberOfThreads, argument3, receivedFile
		}
		receivedFile = false
		if argument0 == "-n" {
			argument1 := os.Args[2] // awk command
			argument2 := os.Args[3] // file to process
			return argument1, numberOfThreads, argument2, receivedFile
		}
		argument1 := os.Args[2] // threads
		numberOfThreads, err := strconv.Atoi(argument1)
		check(err)
		argument2 := os.Args[3] // file to process
		return argument0, numberOfThreads, argument2, receivedFile
	}
	panic("Did not receive any arguments")
}

func getCommand(receivedFile bool, commandFile string) string {
	command := ""
	if receivedFile {
		f, err := os.Open(commandFile) //open the file to process it
		check(err)
		finfo, err := f.Stat()
		check(err)
		fsize := int(finfo.Size())
		buf := make([]byte, fsize)
		bytesContained, err := f.Read(buf)
		check(err)
		command = string(buf[:bytesContained])
	} else {
		command = commandFile
	}
	return command
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

func divideFile(file *os.File, n int) []chunk {
	chunk := make([]chunk, n)
	var data string
	o := int64(0)
	bytesToRead := 0
	end := 0
	filesize := getSize(file)
	defaultSize := int(filesize / int(n))
	for thread := 0; thread < n; thread++ {
		bytesToRead = defaultSize + (bytesToRead - end) + 1 //In this way we check that the chunk does not end just before new line
		b := make([]byte, bytesToRead)                      //the byte length that gets handled by every thread
		textBytes, err := file.Read(b)
		check(err)
		_ = textBytes
		_ = o
		for i := bytesToRead - 1; i > 0; i-- {
			if string(b[i]) == "\n" {
				end = i
				break
			}
		}
		if thread > 0 {
			chunk[thread].buff = b[1:end] //start from 1 to not include the \n
			data = string(b[1:end])
		} else {
			chunk[thread].buff = b[:end]
			data = string(b[:end])
		}
		_ = data
		o, err = file.Seek(o+int64(end), 0)
		check(err)
	}
	return chunk
}

func goAwk(chunk []byte, prog *parser.Program) float64 {
	config := &interp.Config{
		Stdin: bytes.NewReader(chunk),
		Vars:  []string{"OFS", ":"},
	}
	_, err, res := interp.ExecProgram(prog, config)
	check(err)
	return res
}

func main() {
	arg0, n, arg1, commandInFile := receiveArguments()
	awkCommand := getCommand(commandInFile, arg0)
	fmt.Println(awkCommand)
	res := make(chan float64)
	prog, err, varTypes := parser.ParseProgram([]byte(awkCommand), nil)
	check(err)
	if len(varTypes) > 1 {
		panic("Cannot handle awk command that contains local variables")
	}

	// if len(prog.Actions[0].Pattern) > 0 {
	// 	actionPattern := prog.Actions[0].Pattern[0]
	// }
	// _ = actionPattern
	variable := ""

	if len(prog.Actions) > 0 {
		actionStatement := prog.Actions[0].Stmts.String()

		flag := true
		for _, char := range actionStatement {
			if string(char) == "+" || string(char) == "-" {
				flag = false
			}
		}

		if flag || strings.Contains(actionStatement, "print") {
			panic("Cannot handle awk commands that cannot be parallelized")
		}

		for _, char := range actionStatement {
			if string(char) == "+" || string(char) == "-" || string(char) == "=" {
				break
			}
			if string(char) != " " {
				variable = variable + string(char)
			}
		}
	}

	file := openFile(arg1)
	defer file.Close()
	chunks := divideFile(file, n)
	for i := 0; i < n; i++ {
		go func(chunks []chunk, i int, r chan<- float64) {
			chunk := chunks[i]
			result := goAwk(chunk.buff, prog)
			r <- result
		}(chunks, i, res)
	}
	sum := float64(0)
	for i := 0; i < n; i++ {
		sum += <-res
	}

	endStatement := prog.End[0].String()
	i := 0
	for _, char := range endStatement {
		if string(char) == " " {
			i++
		} else {
			break
		}
	}

	endStatement = endStatement[i:]
	endVariable := ""
	variables := []string{}
	for _, char := range endStatement {
		if string(char) == "," {
			continue
		}
		if string(char) != " " {
			endVariable = endVariable + string(char)
		}
		if string(char) == " " {
			variables = append(variables, endVariable)
			endVariable = ""
		}
	}

	variables = append(variables, endVariable)
	variables[len(variables)-1] = strings.TrimSuffix(variables[len(variables)-1], "\n")

	if variables[0] == "print" {
		for _, element := range variables[1:] {
			if element == variable && variable != "" {
				fmt.Printf(" %d ", int(sum))
			} else {
				if string(element[0]) == "\"" && string(element[len(element)-1]) == "\"" {
					fmt.Printf(" %s ", element[1:len(element)-1])
				} else {
					panic("END conatins argument that was not existent in the Action Statement!")
				}
			}
		}
	} else {
		fmt.Println("Only print operations supported in the END statement")
	}
	fmt.Println("")
}
