from os import listdir
from os.path import isfile, join
import subprocess, resource, os

class Benchmark:

    def __init__(self, numberOfThreads=8, fileToWrite='/home/george/Desktop/Github/pawk/results.txt', version='/home/george/Desktop/Github/pawk/./pawk', fileToRead='/home/george/Desktop/Github/pawk/text_files/data.txt'):
        self.numberOfThreads = numberOfThreads
        self.file = fileToWrite
        self.version = version
        self.fileToRead = fileToRead

    def createFile(self):
        if os.path.exists(self.file) and self.version == '/home/george/Desktop/Github/pawk/./pawk':
            os.remove(self.file)
            return open(self.file, 'a')
        else:
            return open(self.file, 'a')

    def getFiles(self):
        onlyfiles = ['tt.03', 'tt.03a', 'tt.04', 'tt.05',  'tt.06', 'stats.txt', 'sum.txt', 'minmax.txt']
        return onlyfiles

    def formCommand(self, awkFiles):
        commands = []
        if self.version == '/home/george/Desktop/Github/pawk/./pawk':
            for thread in range(1, self.numberOfThreads+1):
                command = [self.version, "-n", str(thread), "-f"]
                for benchmarkCommand in awkFiles:
                    command.append(benchmarkCommand)
                    commands.append(command)
                    command = [self.version, "-n", str(thread), "-f"]
            return commands
        else:
            command = [self.version, "-f"]
            for benchmarkCommand in awkFiles:
                command.append(benchmarkCommand)
                commands.append(command)
                command = [self.version, "-f"]
            return commands

    def executeCommand(self, command):
        command.append(self.fileToRead)
        usage_start = resource.getrusage(resource.RUSAGE_CHILDREN)
        subprocess.run(command, stdout=subprocess.DEVNULL)
        usage_end = resource.getrusage(resource.RUSAGE_CHILDREN)
        cpu_time = usage_end.ru_utime - usage_start.ru_utime
        return cpu_time

    def noteTimes(self, cpu_time, test, numOfThreads, resultsFile):
        resultsFile.write("File: " + test + " " + "Threads Used: " + str(numOfThreads) + " " + "Time it took: " + str(cpu_time) + "\n")

    def noteOthersTimes(self, cpu_time, test, resultsFile):
        resultsFile.write("Run With: " + self.version + " File: " + test + " " + "Time it took: " + str(cpu_time) + "\n")

    def main(self):
        files = self.getFiles()
        commands = self.formCommand(files)
        resultsFile = self.createFile()
        for command in commands:
            cpu_time = self.executeCommand(command)
            if self.version == '/home/george/Desktop/Github/pawk/./pawk':
                self.noteTimes(cpu_time, command[4], command[2], resultsFile)
            else:
                self.noteOthersTimes(cpu_time, command[2], resultsFile)
        resultsFile.close()

benchmark1 = Benchmark()
benchmark1.main()

benchmark2 = Benchmark(version='gawk')
benchmark2.main()

benchmark3 = Benchmark(version='/home/george/go/bin/./goawk')
benchmark3.main()

benchmark4 = Benchmark(fileToWrite='/home/george/Desktop/Github/pawk/results2.txt', fileToRead='/home/george/Desktop/Github/pawk/text_files/bigdata.txt')
benchmark4.main()

benchmark5 = Benchmark(fileToWrite='/home/george/Desktop/Github/pawk/results2.txt', fileToRead='/home/george/Desktop/Github/pawk/text_files/bigdata.txt', version='gawk')
benchmark5.main()

benchmark6 = Benchmark(version='/home/george/go/bin/./goawk', fileToWrite='/home/george/Desktop/Github/pawk/results2.txt', fileToRead='/home/george/Desktop/Github/pawk/text_files/bigdata.txt')
benchmark6.main()
