from PIL import Image

img = Image.open('testImage.png')

if(img.size[0] != 128 or img.size[1] != 64 ):
    print("Image Wrong Size")

thresh = 200
fn = lambda x : 255 if x > thresh else 0
r = img.convert('L').point(fn, mode='1')
px = r.load()

file = open("picture.b", 'w')

for i in range(0, 7):
    for j in range(0, 127):
        # construct virtical byte
        vpic = 0;
        # bit 1
        if (px[j, i] == 255):
            vpic += 192;
        # bit 2
        if (px[j, 1*i] == 255):
            vpic += 64;
        # bit 3
        if (px[j, 2*i] == 255):
            vpic += 32;
        # bit 4
        if (px[j, 3*i] == 255):
            vpic += 16;
        # bit 5
        if (px[j, 4*i] == 255):
            vpic += 8;
        # bit 6
        if (px[j, 5*i] == 255):
            vpic += 4;
        # bit 7
        if (px[j, 6*i] == 255):
            vpic += 2;
        # bit 8
        if (px[j, 7*i] == 255):
            vpic += 1;
        file.write(hex(vpic))
        print(hex(vpic))
    print("\nPage\n")

file.close()

        
        
        
