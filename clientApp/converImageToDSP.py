img = Image.open('testimage2.png')

if(img.size[0] != 128 or img.size[1] != 64 ):
    print("Image Wrong Size")



file = open("picture.b", 'wb')


file.close()
class ssd1306FilePrep:

    class imageWrongSize(Exception):
        pass

    def __init__(self, DisplayX=128, DisplayY=64):
        self.image = None
        self.sizex = DisplayX
        self.sizey = DisplayY
        self.pixleStorage = None
        self.constructByteStore = None

    def convertImageToBytes(self, monochromePixleObject=pixleStorage):
        bytestorage = None
        for i in range(0,self.sizey/8):
            for j in range(0, self.sizex):
                # construct virtical byte
                vpic = 0
                # bit 1
                if (monochromePixleObject[j, 0 + (8*i)] == 255):
                    vpic += 1
                # bit 2
                if (monochromePixleObject[j, 1 + (8*i)] == 255):
                    vpic += 2
                # bit 3
                if (monochromePixleObject[j, 2 + (8*i)] == 255):
                    vpic += 4
                # bit 4
                if (monochromePixleObject[j, 3 + (8*i)] == 255):
                    vpic += 8
                # bit 5
                if (monochromePixleObject[j, 4 + (8*i)] == 255):
                    vpic += 16
                # bit 6
                if (monochromePixleObject[j, 5 + (8*i)] == 255):
                    vpic += 32
                # bit 7
                if (monochromePixleObject[j, 6 + (8*i)] == 255):
                    vpic += 64
                # bit 8
                if (monochromePixleObject[j, 7 + (8*i)] == 255):
                    vpic += 128
                bytestorage.append(bytes((vpic,)))
        self.constructByteStore = bytestorage
        return bytestorage
    
    # Converts image to black and white
    def convertBlackAndWhite(self, threshold=200, img=image):
        fn = lambda x : 255 if x > threshold else 0
        r = img.convert('L').point(fn, mode='1')
        px = r.load()
        self.pixleStorage = px
        return px

    def loadImage(self, filename):
        from PIL import Image
        img = Image.open(filename)

        if(img.size[0] != 128 or img.size[1] != 64 ):
            raise self.imageWrongSize
        
        self.image = img
        return img
    
    def saveToFile(self,filename,bytestorage = constructByteStore):
        f = open(filename, 'wb')
        for i in constructByteStore:
            f.write(i)
        f.close()
