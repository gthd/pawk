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
	ifStatement []string
	eventualAwkCommand string
)

// Used to parse input arguments given by the user from console
func init() {
	getopt.FlagLong(&fieldSeparator, "field-separator", 'F', "the path")
	getopt.FlagLong(&numberOfThreads, "threads", 'n', "the number of threads to be used")
	getopt.FlagLong(&fileName, "progfile", 'f', "the file name")
	getopt.FlagLong(&dumpFile, "dump-variables", 'd', "the file to print the global variables")
	getopt.FlagLong(&value, "string", 'v', "strings")
}

// Used when the awk command is provided inside a file rather than written in the console
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

// Used to open a file for reading/writing operations
func openFile(f string) *os.File {
	file, err := os.Open(f) //open the file to process
	check(err)
	return file
}

// Returns the size of the file according to which the file will be divided
func getSize(file *os.File) int {
	fileinfo, err := file.Stat()
	check(err)
	filesize := int(fileinfo.Size())
	return filesize
}

// Returns the starting index and the ending index for all the print statements of the awk command
func returnPrintIndices(statement string) ([]int, []int){
	var phrase string = `print`
	var startingIndex []int
	var endingIndex []int
	compiled := regexp.MustCompile(phrase)
	index := compiled.FindAllStringIndex(statement, -1)
	if len(index) > 0 {
		for i := range index {
			startingIndex = append(startingIndex, index[i][1])
		}
	}

	for iter, b := range []byte(statement) {
		if b == 10 {
			endingIndex = append(endingIndex, iter)
		}
	}

	// checks whether the first ending index is after the first starting index
	for true {
		if endingIndex[0] < startingIndex[0] {
			endingIndex = endingIndex[1:]
		} else {
			break
		}
	}

	// checks whether all ending indexes are after their respective starting indexes
	var tracker = 0
	var test []int

	for i := 0; i < len(endingIndex); i++ {
		if endingIndex[i] > startingIndex[tracker] {
			tracker += 1
			test = append(test, endingIndex[i])
		}
	}
	endingIndex = test
	return startingIndex, endingIndex
}

// Returns the starting index and the ending index for all the print statements of the awk command
func returnBeginPrintIndices(statement string) ([]int, []int){
	var phrase string = `print`
	var startingIndex []int
	var endingIndex []int
	compiled := regexp.MustCompile(phrase)
	index := compiled.FindAllStringIndex(statement, -1)
	if len(index) > 0 {
		for i := range index {
			startingIndex = append(startingIndex, index[i][0])
		}
	}

	for iter, b := range []byte(statement) {
		if b == 59 {
			endingIndex = append(endingIndex, iter)
		}
	}

	// checks whether the first ending index is after the first starting index

	for true {
		if len(endingIndex) > 0 && endingIndex[0] < startingIndex[0] {
			endingIndex = endingIndex[1:]
		} else {
			break
		}
	}

	if len(endingIndex) == 0 {
		endingIndex = append(endingIndex, len(statement))
	}

	// checks whether all ending indexes are after their respective starting indexes
	var tracker = 0
	var test []int

	// Since ending Index should contain
	endingIndex = endingIndex[:len(startingIndex)]

	for i := 0; i < len(endingIndex); i++ {
		if endingIndex[i] > startingIndex[tracker] {
			tracker += 1
			test = append(test, endingIndex[i])
		}
	}
	endingIndex = test
	return startingIndex, endingIndex
}

// Returns the starting index and the ending index for all the if statements of the awk command
func returnIfIndices(statement string) ([]int, []int){
	var phrase string = `if`
	var beginIndex []int
	var endIndex []int
	compiled := regexp.MustCompile(phrase)
	index := compiled.FindAllStringIndex(statement, -1)
	if len(index) > 0 {
		for i := range index {
			beginIndex = append(beginIndex, index[i][1]+2)
		}

		for iter, b := range []byte(statement) {
			if b == 41 {
				endIndex = append(endIndex, iter)
			}
		}

		for true {
			if endIndex[0] < beginIndex[0] {
				endIndex = endIndex[1:]
			} else {
				break
			}
		}

		var tracker = 0
		var test []int
		for i := 0; i < len(endIndex); i++ {
			if endIndex[i] > beginIndex[tracker] {
				tracker += 1
				test = append(test, endIndex[i])
			}
		}
		endIndex = test
		return beginIndex, endIndex
	}
	return beginIndex, endIndex
}

// Used to divide the file to n equal parts that will be fed to the n different processors running in parallel
func divideFile(file *os.File, n int) []chunk {
	chunk := make([]chunk, n)
	var data string
	o := int64(0)
	bytesToRead := 0
	end := 0
	filesize := getSize(file)
	defaultSize := int(filesize / int(n))
	for thread := 0; thread < n; thread++ {

		//In this way we check that the chunk does not end just before new line
		bytesToRead = defaultSize + (bytesToRead - end) + 1

		//the byte length that gets handled by every thread
		b := make([]byte, bytesToRead)
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

			//For all threads other than the first, start from position 1 to exclude \n at the beginning of each chunk
			chunk[thread].buff = b[1:end]
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

// Responsible for communicating with the goAwk dependency. Returns the parsed awk Command
func goAwk(chunk []byte, prog *parser.Program, fieldSeparator string) ([]float64, bool) {
	config := &interp.Config{
		Stdin: bytes.NewReader(chunk),
		Vars:  []string{"OFS", fieldSeparator},
	}
	_, err, res, hasPrint := interp.ExecProgram(prog, config)
	check(err)
	return res, hasPrint
}

// Checks whether a string is contained inside a slice
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

	// used for passing to the BEGIN statement the values given from console with -v option
	var periodContextFmt string = `[Bb][Ee][Gg][Ii][Nn]\s*{`
	sent := regexp.MustCompile(periodContextFmt)
	ind := sent.FindAllStringIndex(awkCommand, -1)

	var argString string
	for _, va := range values {
		argString = argString + va + ";"
	}

	var newAwkCommand string
	if len(values) > 0 {
		if len(ind) > 0 {
			newAwkCommand = string(awkCommand[:ind[0][1]]) + argString + string(awkCommand[ind[0][1]:])
		} else {
			newAwkCommand = "BEGIN { " + argString[:len(argString)-1] + "} " + awkCommand
		}
	} else {
		newAwkCommand = awkCommand
	}

	fmt.Println(newAwkCommand)

	// Handles variable assignment in BEGIN
	if strings.Contains(newAwkCommand, "BEGIN") { //Is it only BEGIN ? Or it can be Begin ?
		// beginStatement := prog.Begin[0].String()

		beginStatement := newAwkCommand[:strings.Index(newAwkCommand, "}")+1]

		// ifStartIndex, ifEndIndex := returnIfIndices(beginStatement)

		printStartIndex, printEndIndex := returnBeginPrintIndices(beginStatement)

		// checks that print operation have something to print
		for i := 0; i < len(printEndIndex); i++ {
			if printEndIndex[i] - printStartIndex[i] <=1 {
				panic("Wrong syntax! Print No " + strconv.Itoa(i+1) + " does not contain anything")
			}
		}

		// builds new string that contains everything except print statements
		var str strings.Builder
		str.WriteString(beginStatement[:printStartIndex[0]])
		for iter := 1; iter < len(printEndIndex); iter++ {
			str.WriteString(beginStatement[printEndIndex[iter-1]:printStartIndex[iter]])
		}
		str.WriteString(beginStatement[printEndIndex[len(printEndIndex)-1]:])

		mystring := str.String()

		indexOfBegin := strings.Index(newAwkCommand, `}`)

		if string(mystring[len(mystring)-1]) != "}" {
			mystring = mystring + `}`
		}

		eventualAwkCommand = mystring + newAwkCommand[indexOfBegin+1:]

		for iter := 0; iter < len(printEndIndex); iter++ {
			printvariable := beginStatement[printStartIndex[iter]:printEndIndex[iter]]
			if string(printvariable[6]) == "\"" && string(printvariable[len(printvariable)-2]) == "\"" {
				fmt.Printf(" %s ", printvariable[7:len(printvariable)-2])
			} else {
				panic("Not provided a valid argument")
			}
		}
	}

	channel := make(chan []string)

	prog, err, varTypes := parser.ParseProgram([]byte(eventualAwkCommand), nil)
	check(err)

	if len(varTypes) > 1 {
		panic("Cannot handle awk command that contains local variables")
	}

	// Used for creating the dump file in case the -d option is passed. Unlike gawk in case -d not provided with file then the dump file is not  written
	if dumpFile != "" {
		dumpFile = `text_files/` + dumpFile
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

	// Used for ensuring that only accumulation operations are allowed in action statements. Print operations not allowed since they cannot be parallelised
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

		// stores to myVariable slice all the variables that exist in the action Statement
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

	// checks that there are not empty variables
	var variable []string
	for _, vvv := range myVariable {
		if len([]byte(vvv)) > 0 {
			variable = append(variable, vvv)
		}
	}

	// Goroutines usage for allowing paralle processing.
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

	// Responsible for receiving the result from each channel and accumulating it. Works sort of like a reduce operation for accumulation
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

	_ = hasPrint

	// if len(prog.Begin) > 0 {
	// 	beginStatement := prog.Begin[0].String()
	// 	j := 0
	// 	for _, char := range beginStatement {
	// 		if string(char) == " " {
	// 			j++
	// 		} else {
	// 			break
	// 		}
	// 	}
	//
	// 	beginStatement = beginStatement[j:]
	// 	beginVariable := ""
	// 	variables := []string{}
	// 	for _, char := range beginStatement {
	// 		if string(char) == "," {
	// 			for _, element := range variables {
	// 				_ = element
	// 			}
	// 			variables = append(variables, ",")
	// 			continue
	// 		}
	// 		if string(char) != " " {
	// 			beginVariable = beginVariable + string(char)
	// 		}
	// 		if string(char) == " " {
	// 			variables = append(variables, beginVariable)
	// 			beginVariable = ""
	// 		}
	// 	}
	//
	// 	variables = append(variables, beginVariable)
	// 	variables[len(variables)-1] = strings.TrimSuffix(variables[len(variables)-1], "\n")
	// 	var newVariables []string
	// 	for _, va := range variables {
	// 		if len(va) > 0 {
	// 			newVariables = append(newVariables, strings.TrimSuffix(va, "\n"))
	// 		}
	// 	}
	//
	// 	fmt.Println(newVariables)
	//
	// 	_ = hasPrint
	// 	if newVariables[0] == "print" {
	// 		for _, element := range newVariables[1:] {
	// 			if string(element[0]) == "\"" && string(element[len(element)-1]) == "\"" {
	// 				fmt.Printf(" %s ", element[1:len(element)-1])
	// 				fmt.Println()
	// 			}
	// 		}
	// 	}
	// }

	if len(prog.End) > 0 {

		endStatement := prog.End[0].String()

		ifStartIndex, ifEndIndex := returnIfIndices(endStatement)

		printStartIndex, printEndIndex := returnPrintIndices(endStatement)

		// checks that print operation have something to print
		for i := 0; i < len(printEndIndex); i++ {
			if printEndIndex[i] - printStartIndex[i] <=1 {
				panic("Wrong syntax! Print No " + strconv.Itoa(i+1) + " does not contain anything")
			}
		}

		var fl = true
		var toprint string
		var toprintslice []string

		// Able to assess if statement and then print accordingly, used in END
		for it, ind := range ifEndIndex {
			for iter, index := range printStartIndex {

				// fl flag used since we need to check if statement for only the first print statement after it
				if ind < index && fl {
					fl = false
					for _, item := range variable {
						ifStatement = strings.Fields(endStatement[ifStartIndex[it]:ifEndIndex[it]])

						// If argument of if statement is a variable
						if ifStatement[0] == item {
							gg, err := strconv.ParseFloat(ifStatement[len(ifStatement)-1], 64)
							if err != nil {
								panic("Need to provide a number when comparing with a variableto compare with when using ")
							}

							// for != operator
							if ifStatement[1] == `!=` {
								if sum[item] != gg {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {

										// since comma is included in the first argument of print exclude it
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}

										// used when argument to print is a string
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])

										// used when arggument to print is a variable
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}

								// for == operator
							} else if ifStatement[1] == `==` {
								if sum[item] == gg { //different types
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}

								// for >= operator
							} else if ifStatement[1] == `>=` {
								if sum[item] >= gg {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}

								// for > operator
							} else if ifStatement[1] == `>` {
								if sum[item] > gg {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}

								// for <= operator
							} else if ifStatement[1] == `<=` {
								if sum[item] <= gg {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}

								// for < operator
							} else if ifStatement[1] == `<` {
								if sum[item] < gg {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							}

							// If argument of if statement is a string
						} else if string(ifStatement[0][0]) == "\"" && string(ifStatement[0][len(ifStatement[0])-1]) == "\"" {
							op := ifStatement[len(ifStatement)-1]

							// So that things like if("a"<9) are not allowed. This is allowed if("a"<"9")
							_, err := strconv.ParseFloat(op, 64)
							if err == nil {
								panic("Need to provide a string when comparing with a string")
							}

							if ifStatement[1] == `!=` {
								if ifStatement[0] != op {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							} else if ifStatement[1] == `==` {
								if ifStatement[0] == op { //different types
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							} else if ifStatement[1] == `>=` {
								if ifStatement[0] >= op {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							} else if ifStatement[1] == `>` {
								if ifStatement[0] > op {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							} else if ifStatement[1] == `<=` {
								if ifStatement[0] <= op {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							} else if ifStatement[1] == `<` {
								if ifStatement[0] < op {
									toprint = endStatement[printStartIndex[iter]:printEndIndex[iter]]
									toprintslice = strings.Fields(toprint)
									for i, pr := range toprintslice {
										if i == 0 {
											if string(pr[len(pr)-1]) == `,` {
												pr = pr[:len(pr)-1]
											}
										}
										if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
											fmt.Printf(" %s ", pr[1:len(pr)-1])
										} else if item == pr {
											fmt.Printf(" %d ", int(sum[item]))
										}
									}
								}
							}
						}
					}
				}
			}
			fl = true
		}

		// find the index of the print statements that do not contain an if clause
		var printindices []int
		for iter, ind := range printStartIndex {
			if len(ifEndIndex) > 0 {
				if ind > ifEndIndex[len(ifEndIndex)-1] {
					printindices = append(printindices, iter)
				}
			} else {
				printindices = append(printindices, iter)
			}
		}

		// since the first corresponds to the last if statement
		if len(ifEndIndex) > 0 {
			printindices = printindices[1:]
		}

		// for each print assess whether it contains a string or a variable and print accordingly
		for _, ind := range printindices {
			toprint = endStatement[printStartIndex[ind]:printEndIndex[ind]]
			toprintslice = strings.Fields(toprint)
			for _, item := range variable {
				for i, pr := range toprintslice {
					if i == 0 {
						if string(pr[len(pr)-1]) == `,` {
							pr = pr[:len(pr)-1]
						}
					}
					if string(pr[0]) == "\"" && string(pr[len(pr)-1]) == "\"" {
						fmt.Printf(" %s ", pr[1:len(pr)-1])
					} else if item == pr {
						fmt.Printf(" %d ", int(sum[item]))
					}
				}
			}
		}

		fmt.Println()
	}
	fmt.Println()
}
