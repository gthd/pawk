### Status
[![Build Status](https://travis-ci.com/gthd/pawk.svg?branch=dev)](https://travis-ci.com/gthd/pawk?branch=dev)

# PAWK
Pawk is an AWK-like language that has been designed to speed up the execution of
all the parallelizable AWK commands by following a map-reduce architecture. More
specifically, Pawk first splits the input text file into multiple chunks and then
processes each chunk on a different thread. Finally, it combines the results from
the different threads with a suitable reduce operation.

GO was chosen as the language of implementation since it offers great support
for multi-threading, through the use of go-routines. It has to be noted here,
that there is a selected number of AWK operations that can be run in parallel.

## Getting Started

### Quickstart

1.  Clone the repository.

    ```
    git clone https://github.com/gthd/pawk.git
    ```

2.  Install the dependencies.

    ```
    go get github.com/gthd/goawk
    go get github.com/gthd/helper
    ```  

## Demo

The first line works under linux_amd64 and the second line under windows_amd64.

The invocation compatibility of Pawk was inspired by GNU Awk and it is as following:

    ```
    ./pawk [-n N] [-d[n]] [-F fs] [-v var=value] [prog | -f progfile] [file ...]
    ```  

The difference with Gawk is with respect to the use of the -d option. In GAWK if a file name is not provided then the global variables are written by default to awkvars.out in the current directory. In Pawk if a file name is not provided to the -d option then there is no file written by default.

## Details

1. When wanting to print a series of variables then they must be separated in this way:

    ```
    print a, b, v
    ```

    This does not work:

    ```
    print a,b,v
    ```

2. When having an unknown variable in a print statement then pawk just ignores it

3. When trying to run pawk with a number of threads that surpass the maximum amount of processing cores available, then an informative message is printed in the console, while threads are set to the    maximum available number of cores

4. When using a print in the BEGIN Statement then the order matters. The following order should be kept:

    ```
    BEGIN {print "hello" ; print "world" ; emp=1}
    BEGIN {emp=1 ; print "hello" ; print "world"}
    ```
    This does not work as expected:

    ```
    BEGIN {print "hello" ; emp=1 ; print "world"}
    ```

5. One should always indicate Begin statements with the keyword `BEGIN`. Any other variance like `Begin` or `begin` leads to unexpected results

6. One should always indicate End statements with the keyword `END`. Any other variance like `End` or `end` leads to unexpected results

7. Local variables are not allowed

8. Dump File is written in the sub-directory text_files/

9. Currently only accumulation is supported

## Contributing

Please read [CONTRIBUTING.md](Contributing.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Authors

* [**Georgios Theodorou**](https://github.com/gthd)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

I want to acknowledge the help and guidance I received from my supervisor Professor [Diomidis Spinellis](https://www2.dmst.aueb.gr/dds/).
