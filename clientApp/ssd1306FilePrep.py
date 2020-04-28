class ssd1306FilePrep(object):
    # the file input file can be any image file
    # the output file should be a file ending in .b
    class imageWrongSize(Exception):
        pass

    def __init__(self, DisplayX=128, DisplayY=64):
        from PIL import Image
        self.image = Image.new('L', [DisplayX,DisplayY])
        self.sizex = DisplayX
        self.sizey = DisplayY
        self.pixleStorage = self.image.load()
        self.constructByteStore = []

    def convertImageToBytes(self, monochromePixleObject=None):
        if monochromePixleObject == None:
            monochromePixleObject = self.pixleStorage

        bytestorage = []
        for i in range(0,int(self.sizey/8)):
            for j in range(0, int(self.sizex)):
                # construct virtical byte
                vpic = 0
                # bit 8
                if monochromePixleObject[j, 0 + (8*i)] == 255:
                    vpic += 1
                # bit 7
                if monochromePixleObject[j, 1 + (8*i)] == 255:
                    vpic += 2
                # bit 6
                if monochromePixleObject[j, 2 + (8*i)] == 255:
                    vpic += 4
                # bit 5
                if monochromePixleObject[j, 3 + (8*i)] == 255:
                    vpic += 8
                # bit 4
                if monochromePixleObject[j, 4 + (8*i)] == 255:
                    vpic += 16
                # bit 3
                if monochromePixleObject[j, 5 + (8*i)] == 255:
                    vpic += 32
                # bit 2
                if monochromePixleObject[j, 6 + (8*i)] == 255:
                    vpic += 64
                # bit 1
                if monochromePixleObject[j, 7 + (8*i)] == 255:
                    vpic += 128
                bytestorage.append(bytes((vpic,)))
        self.constructByteStore = bytestorage
        return bytestorage
    
    # Converts image to black and white
    def convertBlackAndWhite(self, threshold=200, img=None):
        if img == None:
            img=self.image

        fn = lambda x : 255 if x > threshold else 0
        r = img.convert('L').point(fn, mode='1')
        px = r.load()
        self.pixleStorage = px
        return px

    def loadImage(self, filename):
        from pathlib import Path
        from PIL import Image

        img = Image.open(Path(filename))

        if(img.size[0] != 128 or img.size[1] != 64 ):
            raise self.imageWrongSize
        
        self.image = img
        return img
    
    def saveToFile(self,filename,bytestorage=None):
        if bytestorage == None:
            bytestorage = self.constructByteStore
        
        f = open(filename, 'wb')
        for i in bytestorage:
            f.write(i)
        f.close()

def printHelp():
    print("usage -i <inputfile> -o <outputfile>")
    print(" Required Arguments:")
    print(" -i, --inputFile     : specifys the input image file")
    print(" -o, --outputFile    : specifys the output .b file")
    print(" Optional Arguments:")
    print(" -x, --displaySizeX  : sets the with of the screen (defaults to 128)")
    print(" -y, --displaySizeY  : sets the Height of the screen (defaults to 64)")
    print(" -t, --threshold     : sets the threshold value for the conversion (defaults to 200, from 0-255)")
    print(" -h, --help          : prints this help")

def main(argv):
    import getopt
    from sys import exit
    inputfile = '.'
    outputfile = '.'
    displaySizeX = 128
    displaySizeY = 64
    threshold = 200
    try:
        opts, args = getopt.getopt(argv, "xythi:o:", ["inputFile=", "outputFile=", "displaySizeX", "displaySizeY", "threshold", "help"])
    except getopt.GetoptError:
        printHelp()
        exit(2)
    for opt, arg in opts:
        if opt in ('-h', '--help'):
            printHelp()
            exit(2)
        elif opt in ('-i','--inputFile'):
            inputfile = arg
        elif opt in ('-o','--outputFile'):
            outputfile = arg
        elif opt in ('-x','--displaySizeX'):
            displaySizeX = int(arg)
        elif opt in ('-y','--displaySizeY'):
            displaySizeY = int(arg)
        elif opt in ('-t','--threshold'):
            threshold = int(arg)
            if threshold < 0 or threshold > 255:
                print('Threshold value out of bounds')
                exit(2)
    converter = ssd1306FilePrep(displaySizeX,displaySizeY)
    converter.loadImage(inputfile)
    converter.convertBlackAndWhite(threshold)
    converter.convertImageToBytes()
    converter.saveToFile(outputfile)

if __name__ == '__main__':
    import sys
    main(sys.argv[1:])