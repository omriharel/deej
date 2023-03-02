import machine
import utime
 
# GND is hardware pin 38
# 3.3v is hardware pin 36
analog_value1 = machine.ADC(26) # Hardware pin 31
analog_value2 = machine.ADC(27) # Hardware pin 32
analog_value3 = machine.ADC(28) # Hardware pin 34
 
while True:
    #Tested on a B100k pot, gives a range from 0-65535
    #Divide by 64 to give close to the expected range of 0-1023
    norm1 = (analog_value1.read_u16())/64
    norm2 = (analog_value2.read_u16())/64
    norm3 = (analog_value3.read_u16())/64
    print(f"{int(norm1)}|{int(norm2)}|{int(norm3)}")
    utime.sleep(0.2)