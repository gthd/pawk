import random
import string

class FileGenerator:

    def genRandomDigits(self):
        digits = "".join( [random.choice(string.digits) for i in range(random.randint(1,8))])
        return digits

    def genRandomChars(self):
        chars = "".join( [random.choice(string.ascii_letters[:26]) for i in range(random.randint(5, 15))] )
        return chars

    def createFile(self):
        randomChars = []
        for i in range(10):
            randomChars.append(self.genRandomChars())

        randomDigits = []
        for i in range(200):
            randomDigits.append(self.genRandomDigits())

        for i in range(10):
<<<<<<< HEAD
            with open('dummydata.txt', 'a') as the_file:
=======
            with open('data.txt', 'a') as the_file:
>>>>>>> 9cc91e112a19ba26c9a951a0797e18299b244397
                charIndex = random.randint(0,9)
                digfirstIndex = random.randint(0,199)
                digsecondIndex = random.randint(0,199)
                line = randomChars[charIndex] + " " + randomDigits[digfirstIndex] + " " + randomDigits[digsecondIndex] + "\n"
                the_file.write(line)

file = FileGenerator()
file.createFile()
