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
    ```  

## Demo

The first line works under linux_amd64 and the second line under windows_amd64.

To execute Pawk one does not need to install its dependencies, since an executable
is included in the repository. There are four options to execute Pawk.

1. If the command to run exists in the file `awk_command.txt` and the user does
  not specify the number of threads that are going to be used:

    ```
    ./pawk -f -n awk_command.txt data.txt
    ./pawk.exe -f -n awk_command.txt data.txt
    ```

2. If the command to run exists in the file `awk_command.txt` and the user does
  specify the number of threads, 7 in this example, that are going to be used:

    ```
    ./pawk -f awk_command.txt 7 data.txt
    ./pawk.exe -f awk_command.txt 7 data.txt
    ```

3. If the command to run is given in the terminal and the user does not specify
  the number of threads that are going to be used:

    ```
    ./pawk -n '$2*$3 > 5 { emp = emp + 1 } END {print emp}' data.txt
    ./pawk.exe -n '$2*$3 > 5 { emp = emp + 1 } END {print emp}' data.txt
    ```

4. If the command to run is given in the terminal and the user does specify
  the number of threads, 7 in this example, that are going to be used:

    ```
    ./pawk '$2*$3 > 5 { emp = emp + 1 } END {print emp}' 7 data.txt
    ./pawk.exe '$2*$3 > 5 { emp = emp + 1 } END {print emp}' 7 data.txt
    ```

## Contributing

Please read [CONTRIBUTING.md](Contributing.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Authors

* [**Georgios Theodorou**](https://github.com/gthd)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

I want to acknowledge the help and guidance I received from my supervisor [Diomidis Spinellis](https://www2.dmst.aueb.gr/dds/).
