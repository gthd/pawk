from os import listdir
from os.path import isfile, join
import subprocess, resource, os

class Benchmark:

    def __init__(self, numberOfThreads=8, file='/home/george/Desktop/Github/pawk/results.txt', version='./pawk'):
        self.numberOfThreads = numberOfThreads
        self.file = file
        self.version = version

    def createFile(self):
        if os.path.exists("/home/george/Desktop/Github/pawk/results.txt") and self.version == "./pawk":
            os.remove("/home/george/Desktop/Github/pawk/results.txt")
            return open(self.file, 'a')
        else:
            return open(self.file, 'a')

    def getFiles(self):
        onlyfiles = ['tt.01', 'tt.02', 'tt.02a', 'tt.03', 'tt.03a', 'tt.05', 'tt.06', 'tt.07', 'tt.08', 'tt.09', 'tt.10', 'tt.10a', 'tt.11', 'tt.12']
        return onlyfiles

    def formCommand(self, awkFiles):
        commands = []
        if self.version == './pawk':
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
        command.append("mybigdata.txt")
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
            if self.version == './pawk':
                self.noteTimes(cpu_time, command[4], command[2], resultsFile)
            else:
                self.noteOthersTimes(cpu_time, command[2], resultsFile)
        resultsFile.close()

benchmark = Benchmark()
benchmark.main()
