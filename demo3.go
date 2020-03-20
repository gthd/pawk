package main

import (
  "os"
  "fmt"
  "strconv"
  // "github.com/gthd/goawk/interp"
	"github.com/gthd/goawk/parser"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
  test := ""
  b, err := strconv.ParseBool(test)
  if !b {
    fmt.Println("an empty string is false")
  }
  file, err := os.Open("demo5.txt") //open the file to process
  check(err)
  fileinfo, err := file.Stat()
	check(err)
	filesize := int(fileinfo.Size())
	buffer := make([]byte, filesize)
  n1, err := file.Read(buffer)
  check(err)
  arg0 := string(buffer[:n1])
	prog, err, varTypes := parser.ParseProgram([]byte(arg0), nil)
	fmt.Println(prog)
	fmt.Println(varTypes)
}
