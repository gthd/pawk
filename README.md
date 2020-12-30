### Status
[![Build Status](https://travis-ci.com/gthd/pawk.svg?branch=dev)](https://travis-ci.com/gthd/pawk?branch=dev)

# PAWK
Pawk is a reimplementation of the awk pattern-directed scan-ning and processing language that allows awk programs to process input files concurrently on multi-core processors. Its design is based on the insight that an awk program’s predicates are typically stateless, while the corresponding actions can often be combined following the split-apply-combine-strategy for data analysis, as popularized by the MapReduce programming model. Specifically, the input file is split into chunks, which are processed through a dialect of awk that can be executed in parallel without conflicts. Output from record-processing predicates is serialized, while the END pseudo-predicate reduces the data stored by the parallel tasks into the result of an equivalent sequential process. Programs that are not thus parallelizable are run sequentially. pawk is implemented in GO, by extending the existing GoAwk interpreter through a shared memory model with threads. An empirical evaluation of 20 existing awk programs showed that 9 are paralellizable. Measurements show that on large data sets the parallelized programs achieved on average a 20% speedup per core.

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

## Benchmarks

After having installed pawk and its dependencies, to execute the benchmarking script you need to adjust the paths to your system, define a file where the script will write its output and finally define the file that the command will run on. To generate a file with random data, for testing purposes, you can run the text_files/filegen.py script.


## Demo

The invocation compatibility of Pawk was inspired by GNU Awk and it is as following:

    ```
    ./pawk [-n N] [-d[n]] [-F fs] [-v var=value] [prog | -f progfile] [file ...]
    ```  

where -n is the flag for the number of cores to use, -d is the flag for the file to print the global variables, -F is the flag for the field separator and -v is the flag for initialising the variables in the command.

The difference with Gawk is with respect to the use of the -d option. In GAWK if a file name is not provided then the global variables are written by default to awkvars.out in the current directory. In Pawk if a file name is not provided to the -d option then there is no file written by default.

## Usage Details

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

## Contributing

Please read [CONTRIBUTING.md](Contributing.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Authors

* [**Georgios Theodorou**](https://github.com/gthd)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

I want to acknowledge the help and guidance I received from my supervisor Professor [Diomidis Spinellis](https://www2.dmst.aueb.gr/dds/).
