#include <SPI.h>
#include <Wire.h>
#include <SD.h>
#include <ArduinoJson.h>

//You must Hard Code in the number of Sliders in
const int NUM_SLIDERS = 5;
const int analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};

int analogSliderValues[NUM_SLIDERS];



String configFile = "config.conf";


bool pushSliderValuesToPC = false;

const int sdChipSelect = 10;

const byte i2cMultiplexerAddress = 0x70;
const byte i2cDisplayAddress = 0x78;

struct images {
  // because my displays can only have two address's i have to use a multiplexer
  int breakoutPort;
  String imageFile;
};

images imgAssignments[NUM_SLIDERS];

void setup() { 
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);
  }
  if (!SD.begin(sdChipSelect)){
    Serial.println("SD ERROR");
  }
  Serial.begin(9600);
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

void updateSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
     analogSliderValues[i] = analogRead(analogInputs[i]);
  }
}

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

      //Default Catch all
      else {
        Serial.println("INVALID COMMANDS");
      }
    }
  }
}

void loadConfig() {
  // Open file for reading
  File file = SD.open(configFile);

  // Allocate a temporary JsonDocument
  StaticJsonDocument<620> doc;

  // Deserialize the JSON document
  DeserializationError error = deserializeJson(doc, file);
  if (error)
    Serial.println(F("Failed to read file, using default configuration"));

  // for each slider read the config
  for (int i = 0; i < NUM_SLIDERS; i++){
    // set the breakout port
    imgAssignments[i].breakoutPort = int(doc["sliders"][i]["breakoutPort"]);
    // set the image name 
    char imgNameBuff[10];
    strlcpy(imgNameBuff,doc["conf"][i]["imageFile"],sizeof(imgNameBuff));
    for (int j = 0; j < 10; j++){
      imgAssignments[i].imageFile =+ imgNameBuff[j];
    }
  }
}

//breakout port select 
void tcaselect(uint8_t i) {
  if (i > 7) return;
 
  Wire.beginTransmission(i2cMultiplexerAddress);
  Wire.write(1 << i);
  Wire.endTransmission();  
}