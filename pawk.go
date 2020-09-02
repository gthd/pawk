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
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/gthd/goawk/interp"
	"github.com/gthd/goawk/parser"
	"github.com/gthd/helper"
	"github.com/pborman/getopt/v2"
)

type chunk struct {
	buff []byte
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	value           helper.Helper
	numberOfThreads = runtime.GOMAXPROCS(0)
	fieldSeparator  = ":"
	fileName        = ""
	dumpFile        = ""
)

func init() {
	getopt.FlagLong(&fieldSeparator, "field-separator", 'F', "the path")
	getopt.FlagLong(&numberOfThreads, "threads", 'n', "the number of threads to be used")
	getopt.FlagLong(&fileName, "progfile", 'f', "the file name")
	getopt.FlagLong(&dumpFile, "dump-variables", 'd', "the file to print the global variables")
	getopt.FlagLong(&value, "string", 'v', "strings")
}

func getCommand(commandFile string) string {
	command := ""
	f, err := os.Open(commandFile) //open the file to process it
	check(err)
	finfo, err := f.Stat()
	check(err)
	fsize := int(finfo.Size())
	buf := make([]byte, fsize)
	bytesContained, err := f.Read(buf)
	check(err)
	command = string(buf[:bytesContained])
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

func goAwk(chunk []byte, prog *parser.Program, fieldSeparator string) ([]float64, bool) {
	config := &interp.Config{
		Stdin: bytes.NewReader(chunk),
		Vars:  []string{"OFS", fieldSeparator},
	}
	_, err, res, hasPrint := interp.ExecProgram(prog, config)
	check(err)
	return res, hasPrint
}

func isContained(e string, s []string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func main() {

	getopt.Parse()
	args := getopt.Args()

	awkCommand := ""
	if fileName == "" {
		awkCommand = args[0]
		args = args[1:]
	} else {
		awkCommand = getCommand(fileName)
	}

	values := value.ParseMultipleOptions()

	var periodContextFmt string = `[Bb][Ee][Gg][Ii][Nn]\s*{`
	sent := regexp.MustCompile(periodContextFmt)
	ind := sent.FindAllStringIndex(awkCommand, -1)

	var argString string
	for _, va := range values {
		argString = argString + va + ";"
	}

	var newAwkCommand string
	if len(ind) > 0 {
		newAwkCommand = string(awkCommand[:ind[0][1]]) + argString + string(awkCommand[ind[0][1]:])
	} else {
		newAwkCommand = "BEGIN { " + argString[:len(argString)-1] + "} " + awkCommand
	}

	fmt.Println(newAwkCommand)

	channel := make(chan []string)

	prog, err, varTypes := parser.ParseProgram([]byte(newAwkCommand), nil)
	check(err)

	if len(varTypes) > 1 {
		panic("Cannot handle awk command that contains local variables")
	}

	if dumpFile != "" {
		dumpFile = `/home/george/Desktop/Github/pawk/text_files/` + dumpFile
		fmt.Println(dumpFile)
		if _, err := os.Stat(dumpFile); err == nil {
			err := os.Remove(dumpFile)
			check(err)
		}
		dfile, err := os.OpenFile(dumpFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		check(err)
		defer dfile.Close()
		for k := range varTypes[""] {
			if k == "ARGV" {
				continue
			}
			_, err = dfile.Write([]byte(k + "\n"))
			check(err)
		}
	}

	var myVariable []string
	var actionArgument string
	var proceed = true

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
			if string(char) == "+" || string(char) == "-" || string(char) == "=" && proceed {
				myVariable = append(myVariable, actionArgument)
				actionArgument = ""
				proceed = false
			} else if string(char) != " " && proceed {
				actionArgument = actionArgument + string(char)
			} else if uint64([]byte(string(char))[0]) == 10 {
				proceed = true
			}
		}
	}

	var variable []string
	for _, vvv := range myVariable {
		if len([]byte(vvv)) > 0 {
			variable = append(variable, vvv)
		}
	}

	array := make([][]string, numberOfThreads)
	for _, file := range args {
		file := openFile(file)
		defer file.Close()
		chunks := divideFile(file, numberOfThreads)
		for i := 0; i < numberOfThreads; i++ {
			go func(chunks []chunk, i int, r chan<- []string) {
				var ar []string
				chunk := chunks[i]
				result, hasPrint := goAwk(chunk.buff, prog, fieldSeparator)
				for _, r := range result {
					ar = append(ar, strconv.FormatFloat(r, 'f', -1, 64))
				}
				ar = append(ar, strconv.FormatBool(hasPrint))
				r <- ar
			}(chunks, i, channel)
		}
		for i := 0; i < numberOfThreads; i++ {
			array[i] = <-channel
		}
	}

	sum := make(map[string]float64)
	var hasPrint bool
	for _, ar := range array {
		for iter, a := range ar {
			if iter < len(ar)-1 {
				num, err := strconv.ParseFloat(a, 64)
				check(err)
				sum[variable[iter]] += num
			}
		}
		hasPrint, err = strconv.ParseBool(array[0][len(array[0])-1])
		check(err)
	}

	if len(prog.Begin) > 0 {
		beginStatement := prog.Begin[0].String()
		j := 0
		for _, char := range beginStatement {
			if string(char) == " " {
				j++
			} else {
				break
			}
		}

		beginStatement = beginStatement[j:]
		beginVariable := ""
		variables := []string{}
		for _, char := range beginStatement {
			if string(char) == "," {
				for _, element := range variables {
					_ = element
				}
				variables = append(variables, ",")
				continue
			}
			if string(char) != " " {
				beginVariable = beginVariable + string(char)
			}
			if string(char) == " " {
				variables = append(variables, beginVariable)
				beginVariable = ""
			}
		}

		variables = append(variables, beginVariable)
		variables[len(variables)-1] = strings.TrimSuffix(variables[len(variables)-1], "\n")
		var newVariables []string
		for _, va := range variables {
			if len(va) > 0 {
				newVariables = append(newVariables, strings.TrimSuffix(va, "\n"))
			}
		}

		_ = hasPrint
		if newVariables[0] == "print" {
			for _, element := range newVariables[1:] {
				if string(element[0]) == "\"" && string(element[len(element)-1]) == "\"" {
					fmt.Printf(" %s ", element[1:len(element)-1])
					fmt.Println()
				}
			}
		}
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
			if isContained(element, variable) {
				fmt.Printf(" %d ", int(sum[element]))
			} else {
				if string(element[0]) == "\"" && string(element[len(element)-1]) == "\"" {
					fmt.Printf(" %s ", element[1:len(element)-1])
				} else {
					panic("END contains unknown variable")
				}
			}
		}
	} else {
		fmt.Println("Only print operations supported in the END statement")
	}
	fmt.Println("")
}
