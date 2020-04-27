#include <SPI.h>
#include <Wire.h>
#include <SD.h>
#include "ssd1306CMD.h"
//You must Hard Code in the number of Sliders in
const int NUM_SLIDERS = 5;
const int analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};

int analogSliderValues[NUM_SLIDERS];

const int numDisplays = 5;

// Constent Send
bool pushSliderValuesToPC = false;

// Sd Card CS
const int sdChipSelect = 10;

// I2C Addresses
const byte i2cMultiplexerAddress = 0x70;
const byte i2cDisplayAddress = 0x3C;

// GFX Settings
const int SCREEN_WIDTH = 128; // OLED display width, in pixels
const int SCREEN_HEIGHT = 64; // OLED display height, in pixels

void setup() { 
  // Set up Wire for multiplexer
  Wire.begin();
  Serial.begin(115200);
  Serial.print("INITSTART ");
  for (int i = 0; i < numDisplays; i++) {
    Serial.print("DSP" + String(i) + "INIT ");
    tcaselect(i);
    dspInit();
    dspClear();
    dspSendData(i);
  }
  
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);
  }
  Serial.print("SDINIT ");
  if (!SD.begin(sdChipSelect)){
    Serial.println("SDERROR ");
    while(1);
  }
  Serial.println("INITDONE");
}

void loop() {
  checkForCommand();

  updateSliderValues();

  //Check for data chanel to be open
  if(pushSliderValuesToPC) {
    sendSliderValues(); // Actually send data
  }
  // printSliderValues(); // For debug
  delay(10);
}

// upates the array of slider values
void updateSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
     analogSliderValues[i] = analogRead(analogInputs[i]);
  }
}

// sends the machine readable values of the sliders
void sendSliderValues() {
  String builtString = String("");

  for (int i = 0; i < NUM_SLIDERS; i++) {
    builtString += String((int)analogSliderValues[i]);

    if (i < NUM_SLIDERS - 1) {
      builtString += String("|");
    }
  }
  
  Serial.println(builtString);
}

// sends the human readable values of the sliders
void printSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
    String printedString = String("Slider #") + String(i + 1) + String(": ") + String(analogSliderValues[i]) + String(" mV");
    Serial.write(printedString.c_str());

    if (i < NUM_SLIDERS - 1) {
      Serial.write(" | ");
    } else {
      Serial.write("\n");
    }
  }
}

void checkForCommand() {
  //Check if data is waiting
  if (Serial.available() > 0) {
    //Get start time of command
    int timeStart = millis();

    //Get data from Serial
    String input = Serial.readStringUntil('\n');  // Read chars from serial monitor
    
    //Get Stop Time
    int timeStop = millis();

    //If data takes to long
    if(timeStart-timeStop >= 1000) {
      Serial.println("TIMEOUT");
    }
    // Check and match commands
    else {

      // Start Sending Slider Values
      if ( input.equalsIgnoreCase("startSlider") == true ) {
        pushSliderValuesToPC = true;
      }

      // Stop Sending Slider Values
      else if ( input.equalsIgnoreCase("stopSlider") == true ) {
        pushSliderValuesToPC = false;
      }
      
      // Send Single Slider Values
      else if ( input.equalsIgnoreCase("getSlider") == true ) {
        sendSliderValues();
      }

      // Send Human Readable Slider Values 
      else if ( input.equalsIgnoreCase("getSliderHR") == true ) {
        printSliderValues();
      }
      
      // Sends a file to the sd card 
      else if ( input.equalsIgnoreCase("sendFile") == true ) {
        getFile();
      }

      // Sets the image on a display
      else if ( input.equalsIgnoreCase("setDisplayImage") == true){
        Serial.println("What Dispaly:");
        timeStart = millis();

        //Get data from Serial
        String port = Serial.readStringUntil('\n');  // Read chars from serial monitor
        
        //Get Stop Time
        timeStop = millis();
        
        //If data takes to long
        if(timeStart-timeStop >= 1000) {
          Serial.println("TIMEOUT");
        }
        Serial.println("What ImageFile:");
        timeStart = millis();

        //Get data from Serial
        String filename = Serial.readStringUntil('\n');  // Read chars from serial monitor
        
        //Get Stop Time
        timeStop = millis();
        
        //If data takes to long
        if(timeStart-timeStop >= 1000) {
          Serial.println("TIMEOUT");
        }
        setImage(port.toInt(),filename);
      }

      //Default Catch all
      else {
        Serial.println("INVALID COMMANDS");
      }
    }
  }
}

void getFile() {
  Serial.println("Enter File Name");

  //Get start time of command
  int timeStart = millis();

  //Get data from Serial
  String filename = Serial.readStringUntil('\n');  // Read chars from serial monitor

  //Get Stop Time
  int timeStop = millis();

  //If data takes to long
  if(timeStart-timeStop >= 1000) {
    Serial.println("TIMEOUT: No Filename recived");
    return;
  }

  Serial.println("Starting File Write");
  Serial.println("Waiting for EOF");
  File imgFile = SD.open(filename, FILE_WRITE);
  int last3[3];
  while ( last3[0] != 'E' && last3[1] != 'O' && last3[2] != 'F' ) {
    if ( last3[0] != -1 ) {
      imgFile.write(last3[0]);
    }
    last3[0] = last3[1];
    last3[1] = last3[2];
    int nextByte = Serial.read();
    if (nextByte != -1) {
      last3[2] = Serial.read();
    }
  }
}

// Writes a image to the ssd1306 display
void setImage(uint8_t port, String imagefilename) {
  // select the display port
  tcaselect(port);
  // open the image file
  // also this file should almost allways contain 8192 bytes
  File imgFile = SD.open(imagefilename);
  // clear the display
  dspClear();
  // initialize some temp vars
  int inputChar;
  int maxPages = 8;

  // loop through each page 
  // each padge is 8 Vertical bytes per column
  // we write to 128 columns each column is 8 bytes tall or 8 pixel.
  // there are 8 pages [0-7] to make up the 64 pixel tall display
  // we also process all posable ascii char including newline and carrage return
  // since a char is one byte it makes it easy to read data from the file and into the buffer
  while (maxPages != 0 && inputChar != -1){
    int CharsLeftInLine = 128;
    while  (CharsLeftInLine > 0 && inputChar != -1){
      inputChar = imgFile.read();
      Serial.print(char(inputChar));
      if(inputChar == -1){
        break;
      }
      dspSendData(inputChar);
      CharsLeftInLine--;
    }
    Serial.println();
    maxPages--;
  }
  imgFile.close();
}

// breakout port select 
void tcaselect(uint8_t i) {
  if (i > 7) return;
 
  Wire.beginTransmission(i2cMultiplexerAddress);
  Wire.write(1 << i);
  Wire.endTransmission();  
}

// Send a Command to the ssd1306 
void dspSendCommand(uint8_t c){
  Wire.beginTransmission(i2cDisplayAddress);
  Wire.write(0x00);
  Wire.write(c);
  Wire.endTransmission();
}

// Send Display Data to ssd1306
void dspSendData(uint8_t c){
  Wire.beginTransmission(i2cDisplayAddress);
  Wire.write(0x40);
  Wire.write(c);
  Wire.endTransmission();
}

// ssd1306 Display initialization sequence
// see this page for the sequence for the sequence i used:
// https://iotexpert.com/2019/08/07/debugging-ssd1306-display-problems/
const char initializeCmds[]={
  //////// Fundamental Commands
  OLED_DISPLAYOFF,          // 0xAE Screen Off
  OLED_SETCONTRAST,         // 0x81 Set contrast control
  0x7F,                     // 0-FF ... default half way
  OLED_DISPLAYNORMAL,       // 0xA6, //Set normal display 
  //////// Scrolling Commands
  OLED_DEACTIVATE_SCROLL,   // Deactive scroll
  //////// Addressing Commands
  OLED_SETMEMORYMODE,       // 0x20, //Set memory address mode
  OLED_SETMEMORYMODE_HORIZONTAL,  // Page
  //////// Hardware Configuration Commands
  OLED_SEGREMAPINV,         // 0xA1, //Set segment re-map 
  OLED_SETMULTIPLEX,        // 0xA8 Set multiplex ratio
  0x3F,                     // Vertical Size - 1
  OLED_COMSCANDEC,          // 0xC0 Set COM output scan direction
  OLED_SETDISPLAYOFFSET,    // 0xD3 Set Display Offset
  0x00,                     //
  OLED_SETCOMPINS,          // 0xDA Set COM pins hardware configuration
  0x12,                     // Alternate com config & disable com left/right
  //////// Timing and Driving Settings
  OLED_SETDISPLAYCLOCKDIV,  // 0xD5 Set display oscillator frequency 0-0xF /clock divide ratio 0-0xF
  0x80,                     // Default value
  OLED_SETPRECHARGE,        // 0xD9 Set pre-changed period
  0x22,                     // Default 0x22
  OLED_SETVCOMDESELECT,     // 0xDB, //Set VCOMH Deselected level
  0x20,                     // Default 
  //////// Charge pump regulator
  OLED_CHARGEPUMP,          // 0x8D Set charge pump
  OLED_CHARGEPUMP_ON,       // 0x14 VCC generated by internal DC/DC circuit
  // Turn the screen back on...       
  OLED_DISPLAYALLONRESUME,  // 0xA4, //Set entire display on/off
  OLED_DISPLAYON,           // 0xAF  //Set display on
};

// initialize displays using the sequence
void dspInit(){
  for(int i=0;i<25;i++){
    dspSendCommand(initializeCmds[i]);
  }
}

// set the column 
// ref the ssd 1306 datasheet if you want to find out how it works
void dspSetColumn(uint8_t cstart, uint8_t cend) {
  dspSendCommand(0x21);
  dspSendCommand(cstart);
  dspSendCommand(cend);
}

// set the page
// ref the ssd 1306 datasheet if you want to find out how it works
void dspSetPage(uint8_t cstart, uint8_t cend) {
  dspSendCommand(0x22);
  dspSendCommand(cstart);
  dspSendCommand(cend);
}

// clear the display
void dspClear(){
  // go to zero and set end to full end
  dspSetColumn(0x00,0x7F);
  // go to zero and set end to full end
  dspSetPage(0x00,0x7);
  // fill the GFX Ram on the ssd1306 with zeros blanking the display
  for(int i = 0;i < (SCREEN_WIDTH * SCREEN_HEIGHT); i++){
    dspSendData(0b00000000);
  }
}