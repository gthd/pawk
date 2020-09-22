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
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"log"
	"net/http"
	_ "net/http/pprof"

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
	value                helper.Helper
	numberOfThreads      = runtime.GOMAXPROCS(0)
	fieldSeparator       = " "
	offsetFieldSeparator = ":"
	fileName             = ""
	dumpFile             = ""
	eventualAwkCommand   string
	endStatement         string
	nameSlice            []string
	min                  float64
	max                  float64
	indexEnd             [][]int
	emptyStmt            bool
	text                 []byte
	pp                   *parser.Program
	hasEnd 								bool
	hasBegin bool
	bbb string
	associativeValues map[string]map[string]float64
	associativeValue map[string]float64
	associativeArrays map[int]map[string]float64
)

type received struct {
	results         []float64
	nativeFunctions []bool
	functionNames   []string
	associativeArray map[string]float64
}

// Used to parse input arguments given by the user from console
func init() {
	getopt.FlagLong(&fieldSeparator, "field-separator", 'F', "the field separator")
	getopt.FlagLong(&numberOfThreads, "threads", 'n', "the number of threads to be used")
	getopt.FlagLong(&fileName, "progfile", 'f', "the file name")
	getopt.FlagLong(&dumpFile, "dump-variables", 'd', "the file to print the global variables")
	getopt.FlagLong(&value, "string", 'v', "strings")
	getopt.FlagLong(&offsetFieldSeparator, "offset-field-separator", 'o', "the offset field separator")
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
func returnBeginPrintIndices(statement string) ([]int, []int) {
	var phrase = `print`
	var startingIndex []int
	var endingIndex []int
	compiled := regexp.MustCompile(phrase)
	index := compiled.FindAllStringIndex(statement, -1)
	if len(index) > 0 {
		for i := range index {
			startingIndex = append(startingIndex, index[i][0])
		}

		for iter, b := range []byte(statement) {
			if b == 59 {
				endingIndex = append(endingIndex, iter)
			}
		}

		// checks whether the first ending index is after the first starting index

		for {
			if len(endingIndex) > 0 && endingIndex[0] < startingIndex[0] {
				endingIndex = endingIndex[1:]
			} else {
				break
			}
		}

		if len(endingIndex) == 0 {
			endingIndex = append(endingIndex, len(statement))
		} else if startingIndex[len(startingIndex)-1] > endingIndex[len(endingIndex)-1] {
			endingIndex = append(endingIndex, len(statement))
		}

		// checks whether all ending indexes are after their respective starting indexes
		var tracker = 0
		var test []int

		// Since ending Index should contain
		endingIndex = endingIndex[:len(startingIndex)]

		for i := 0; i < len(endingIndex); i++ {
			if endingIndex[i] > startingIndex[tracker] {
				tracker++
				test = append(test, endingIndex[i])
			}
		}
		endingIndex = test
		return startingIndex, endingIndex
	}
	return startingIndex, endingIndex
}

// Used to divide the file to n equal parts that will be fed to the n different processors running in parallel
func divideFile(file *os.File, n int) []chunk {
	chunk := make([]chunk, n)
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
		_, err := file.Read(b)
		check(err)
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
		} else {
			chunk[thread].buff = b[:end]
		}
		o, err = file.Seek(o+int64(end), 0)
		check(err)
		// fmt.Println(chunk[thread].buff)
	}
	return chunk
}

// Responsible for communicating with the goAwk dependency. Returns the parsed awk Command
func goAwk(chunk []byte, prog *parser.Program, fieldSeparator string, offsetFieldSeparator string, funcs map[string]interface{}) ([]float64, []bool, []string, map[string]float64) {
	config := &interp.Config{
		Stdin: bytes.NewReader(chunk),
		Vars:  []string{"OFS", offsetFieldSeparator, "FS", fieldSeparator},
		Funcs: funcs,
	}
	_, err, res, natives, names, arrays := interp.ExecProgram(prog, config)
	check(err)
	return res, natives, names, arrays
}

// Checks whether a string is contained inside a slice.
func isContained(s string, slice []string) bool {
	flag := false
	for _, k := range slice {
		if k == s {
			flag = true
		}
	}
	return flag
}

func getFunctions() map[string]interface{} {

	funcs := map[string]interface{}{
		"min": func(num1 float64, num2 float64) float64 {
			if num1 < num2 {
				return num1
			}
			return num2
		},
		"max": func(num1 float64, num2 float64) float64 {
			if num1 > num2 {
				return num1
			}
			return num2
		},
		"and": func(bool1 bool, bool2 bool) bool {
			return bool1 && bool2
		},
		"or": func(bool1 bool, bool2 bool) bool {
			return bool1 || bool2
		},
		"xor": func(bool1 bool, bool2 bool) bool {
			return bool1 != bool2
		},
	}
	return funcs
}

func main() {

	debug.SetGCPercent(50)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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
	var periodContextFmt = `[Bb][Ee][Gg][Ii][Nn]\s*{`
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

	// Handles variable assignment in BEGIN as well as print statement
	// CANNOT have something like this BEGIN {print "cndckd" ; emp=1 ; print "kcndkc"}
	// SHOULD BE BEGIN {print "cndckd" ; print "kcndkc" ; emp=1}
	// OR BEGIN {emp=1 ; print "cndckd" ; print "kcndkc"}
	if strings.Contains(newAwkCommand, "BEGIN") { //Is it only BEGIN ? Or it can be Begin ?
		// beginStatement := prog.Begin[0].String()
		hasBegin = true
		beginStatement := newAwkCommand[:strings.Index(newAwkCommand, "}")+1]

		printStartIndex, printEndIndex := returnBeginPrintIndices(beginStatement)

		// If print exists in BEGIN
		if len(printStartIndex) > 0 {
			// checks that print operation have something to print
			for i := 0; i < len(printEndIndex); i++ {
				if printEndIndex[i]-printStartIndex[i] <= 1 {
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
				if string(printvariable[6]) == "\"" && string(printvariable[len(printvariable)-1]) == "\"" {
					fmt.Printf(" %s ", printvariable[7:len(printvariable)-1])
				} else if string(printvariable[6]) == "\"" && string(printvariable[len(printvariable)-2]) == "\"" {
					fmt.Printf(" %s ", printvariable[7:len(printvariable)-2])
				} else {
					panic("Not provided a valid argument")
				}
			}
		} else {
			eventualAwkCommand = newAwkCommand
		}
	} else {
		eventualAwkCommand = newAwkCommand
	}

	// Remove the END statement gets handled on its own at the end
	if strings.Contains(newAwkCommand, "END") { //Is it only BEGIN ? Or it can be Begin ?
		hasEnd = true
		var regexstring = `[Ee][Nn][Dd]\s*{`
		comp := regexp.MustCompile(regexstring)
		indexEnd = comp.FindAllStringIndex(eventualAwkCommand, -1)

		endStatement = eventualAwkCommand[indexEnd[0][0]:]

		eventualAwkCommand = strings.ReplaceAll(eventualAwkCommand, endStatement, "")

	}

	funcs := getFunctions()

	channel := make(chan *received)
	config := &parser.ParserConfig{
		Funcs: funcs,
	}

	init := eventualAwkCommand
	if hasEnd && hasBegin {
		bbb = eventualAwkCommand[strings.Index(eventualAwkCommand, "}")+1 : indexEnd[0][0]]
	} else if hasEnd && !hasBegin {
		bbb = eventualAwkCommand[:indexEnd[0][0]]
	} else if hasBegin && !hasEnd {
		bbb = eventualAwkCommand[strings.Index(eventualAwkCommand, "}")+1:]
	} else {
		bbb = eventualAwkCommand
	}

	printStartIndex, printEndIndex := returnBeginPrintIndices(bbb)

	// Responsible for removing print statements from action statement
	if len(printStartIndex) > 0 && !strings.Contains(eventualAwkCommand, "for") {
		// checks that print operation have something to print
		for i := 0; i < len(printEndIndex); i++ {
			if printEndIndex[i]-printStartIndex[i] <= 1 {
				panic("Wrong syntax! Print No " + strconv.Itoa(i+1) + " does not contain anything")
			}
		}

		// builds new string that contains everything except print statements
		var str strings.Builder
		str.WriteString(bbb[:printStartIndex[0]])
		for iter := 1; iter < len(printEndIndex); iter++ {
			str.WriteString(bbb[printEndIndex[iter-1]:printStartIndex[iter]])
		}
		str.WriteString(bbb[printEndIndex[len(printEndIndex)-1]:])

		mystring := str.String()

		// indexOfBegin := strings.Index(newAwkCommand, `}`)

		if string(mystring[len(mystring)-1]) != "}" {
			mystring = mystring + `}`
		}

		if len(strings.TrimSpace(mystring)) == 2 {
			emptyStmt = true
		}

		if hasBegin && hasEnd {
			abc := eventualAwkCommand[:strings.Index(eventualAwkCommand, "}")+1]
			def := eventualAwkCommand[indexEnd[0][0]:]
			eventualAwkCommand = abc + mystring + def
		} else if hasBegin && !hasEnd {
			abc := eventualAwkCommand[:strings.Index(eventualAwkCommand, "}")+1]
			eventualAwkCommand = abc + mystring
		} else if !hasBegin && hasEnd {
			def := newAwkCommand[indexEnd[0][0]:]
			eventualAwkCommand = mystring + def
		} else if !hasBegin && !hasEnd {
			eventualAwkCommand = mystring
		}
	} else {
		eventualAwkCommand = init
	}

	prog, err, varTypes := parser.ParseProgram([]byte(eventualAwkCommand), config)
	check(err)

	// Responsible for executing the print statements that exist in the action statement. Uses one thread since print cannot be parallelised
	if len(printStartIndex) > 0 && len(prog.Actions) == 1 && !strings.Contains(eventualAwkCommand, "for"){
		if len(prog.Actions) == 1 {
			pp, err, _ = parser.ParseProgram([]byte(bbb), nil)
			check(err)
		} else {
			pp, err, _ = parser.ParseProgram([]byte(bbb[printStartIndex[0]-1:printEndIndex[0]]), nil)
			check(err)
		}

		for _, file := range args {
			file := openFile(file)
			defer file.Close()
			text = append(text, divideFile(file, 1)[0].buff...)
		}

		input := bytes.NewReader(text)

		configEnd := &interp.Config{
			Stdin:  input,
			Output: nil,
			Error:  ioutil.Discard,
			Vars:   []string{"OFS", offsetFieldSeparator, "FS", fieldSeparator},
		}

		_, err, _ = interp.ExecOneThread(pp, configEnd, associativeArrays)
		check(err)
	}

	funcnames := make([]string, 0, len(funcs))
	for k := range funcs {
		funcnames = append(funcnames, k)
	}

	if len(varTypes) > 1 {
		fmt.Println("PP")
		oneThreadProg, err, _ := parser.ParseProgram([]byte(awkCommand), config)
		check(err)
		for _, file := range args {
			file := openFile(file)
			defer file.Close()
			text = append(text, divideFile(file, 1)[0].buff...)
		}
		input := bytes.NewReader(text)
		oneThreadConfig := &interp.Config{
			Stdin:  input,
			Output: nil,
			Error:  ioutil.Discard,
			Vars:   []string{"OFS", offsetFieldSeparator, "FS", fieldSeparator},
		}
		_, err, _ = interp.ExecOneThread(oneThreadProg, oneThreadConfig, associativeArrays)
		check(err)
		os.Exit(0)
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

	// Used for ensuring that only accumulation and assignment operations are allowed in action statements.
	if len(prog.Actions) > 0 {
		actionStatement := prog.Actions[0].Stmts.String()

		ok := false
		if len(funcnames) > 0 {
			actionSlice := strings.Fields(actionStatement)
			for _, s := range actionSlice {
				for _, n := range funcnames {
					if strings.Contains(s, n) {
						nameSlice = append(nameSlice, n)
						ok = true
					}
				}
			}
		}

		for _, char := range actionStatement {
			if string(char) == "+" || string(char) == "-" || string(char) == "="{
				ok = true
			}
		}
		_ = ok
		// If action statement does not contain a user defined function or an accumulation operation
		if !ok && !strings.Contains(actionStatement, "print") && !emptyStmt && len(actionStatement) > 0 {
			oneThreadProg, err, _ := parser.ParseProgram([]byte(awkCommand), config)
			check(err)
			for _, file := range args {
				file := openFile(file)
				defer file.Close()
				text = append(text, divideFile(file, 1)[0].buff...)
			}
			input := bytes.NewReader(text)
			oneThreadConfig := &interp.Config{
				Stdin:  input,
				Output: nil,
				Error:  ioutil.Discard,
				Vars:   []string{"OFS", offsetFieldSeparator, "FS", fieldSeparator},
			}
			_, err, _ = interp.ExecOneThread(oneThreadProg, oneThreadConfig, associativeArrays)
			check(err)
			os.Exit(0)
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
		if isContained(vvv, variable) {
			continue;
		}
		if strings.Contains(vvv, "for") {
			continue;
		}
		if len([]byte(vvv)) > 0 {
			variable = append(variable, vvv)
		}
	}

	// Goroutines usage for allowing paralle processing.
	if numberOfThreads > runtime.GOMAXPROCS(0) {
		fmt.Println("Number of threads surpasses available CPU cores. Reverting to " + strconv.Itoa(runtime.GOMAXPROCS(0)) + " threads. (Equal to the maximum number of CPU cores)")
		numberOfThreads = runtime.GOMAXPROCS(0)
	}

	array := make([]*received, numberOfThreads)
	for _, file := range args {
		file := openFile(file)
		defer file.Close()
		chunks := divideFile(file, numberOfThreads)
		// for _, c := range chunks {
		// 	fmt.Println("BEGIN")
		// 	fmt.Println(string(c.buff))
		// 	fmt.Println("END")
		// }
		for i := 0; i < numberOfThreads; i++ {
			go func(chunks []chunk, i int, r chan<- *received) {
				chunk := chunks[i]
				res, nat, names, arrays := goAwk(chunk.buff, prog, fieldSeparator, offsetFieldSeparator, funcs)
				got := &received{results: res, nativeFunctions: nat, functionNames: names, associativeArray: arrays}
				r <- got
			}(chunks, i, channel)
		}
		for i := 0; i < numberOfThreads; i++ {
			array[i] = <-channel
		}
	}

	// for i:= 0; i < numberOfThreads; i++ {
	// 	fmt.Println(array[i].results)
	// 	fmt.Println(array[i].nativeFunctions)
	// 	fmt.Println(array[i].functionNames)
	// 	fmt.Println(array[i].associativeArray)
	// }

	// Performs the suitable Reduction
	mapOfVariables := make(map[string]float64)
	j := 0
	k := 0
	for true {
		if len(array[k].nativeFunctions) > 0 {
			break
		} else {
			k += 1
		}
	}
	boolSlice := array[k].nativeFunctions
	if len(variable) > 0 {
		if len(boolSlice) == len(variable) {
			for i := 0; i < len(boolSlice); i++ {
				if boolSlice[i] { //means we deal with native function
					if nameSlice[j] == "min" {
						min = array[k].results[i]
						for _, ar := range array {
							if len(ar.results) > 0 {
								if ar.results[i] < min {
									min = ar.results[i]
								}
							}
						}
						mapOfVariables[variable[i]] = min
					} else if nameSlice[j] == "max" {
						max = array[k].results[i]
						for _, ar := range array {
							if len(ar.results) > 0 {
								if ar.results[i] > max {
									max = ar.results[i]
								}
							}
						}
						mapOfVariables[variable[i]] = max
					}
					j++
				} else {
					for _, ar := range array {
						if len(ar.results) > 0 {
							mapOfVariables[variable[i]] += ar.results[i]
						}
					}
				}
			}
		}
	}

	if len(variable) > 0 {
		if len(array[0].associativeArray) > 0 {
			associativeValue = make(map[string]float64)
			associativeValues = make(map[string]map[string]float64)
			for i:=0; i < len(variable); i++ {
				match, _ := regexp.MatchString("\\[[^\\]]*\\]", variable[i])
				if match {
					for _, ar := range array {
						for k := range ar.associativeArray {
							associativeValue[k] += ar.associativeArray[k]
							// associativeValues[variable[i]][k] += ar.associativeArray[k]
						}
					}
					variable[i] = variable[i][:strings.Index(variable[i], "[")]
					associativeValues[variable[i]] = associativeValue
				} else {
						if mapOfVariables[variable[i]] == float64(0) {
							for _, ar := range array {
								for k := range ar.associativeArray {
									mapOfVariables[variable[i]] += ar.associativeArray[k]
								}
							}
						}
					}
			}
		}
	}



	end, err, _ := parser.ParseProgram([]byte(endStatement), nil)
	check(err)

	arrayKeys := make([]string, 0, len(end.Arrays))
	for k := range end.Arrays {
		arrayKeys = append(arrayKeys, k)
	}

	associativeArrays = make(map[int]map[string]float64)

	for i, k := range arrayKeys {
		if k == "ARGV" {
			associativeArrays[i] = make(map[string]float64)
		} else {
			for _, vf := range variable {
				if vf == k {
					associativeArrays[i] = associativeValues[k]
				}
			}
		}
	}

	keys := make([]string, 0, len(mapOfVariables))
	for k := range mapOfVariables {
		keys = append(keys, k)
	}

	for _, k := range keys {
		end.Scalars[k] = mapOfVariables[k]
	}
	
	input := bytes.NewReader([]byte("foo bar\n\nbaz buz"))
	configEnd := &interp.Config{
		Stdin:  input,
		Output: nil,
		Error:  ioutil.Discard,
		Vars:   []string{"OFS", " ", "FS", " "},
		Funcs:  funcs,
	}

	_, err, _ = interp.ExecOneThread(end, configEnd, associativeArrays)
	check(err)
}
