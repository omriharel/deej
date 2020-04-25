#include <SPI.h>
#include <SD.h>

//You must Hard Code in the number of Sliders in
const int NUM_SLIDERS = 5;
const int analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};

int analogSliderValues[NUM_SLIDERS];
bool pushSliderValuesToPC = false;

const int sd_CS = 8;

void setup() { 
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);
  }

  sd.begin(sd_CS);
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
        pushSliderValuesToPC = false
        
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