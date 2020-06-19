#include "arduino.h"
#include <avr/wdt.h>

//Microcontroller type
//#define MCU32U4 1
#define MCUA328P 1

//You must Hard Code in the number of Sliders in
#define NUM_SLIDERS 6
#define SERIALSPEED 9600
#define FrequencyMS 10
#define SerialTimeout 2000 //This is two seconds

const uint8_t analogInputs[NUM_SLIDERS] = {18, 19, 20, 21, 9, 8};

uint16_t analogSliderValues[NUM_SLIDERS];

// Constend Send
bool pushSliderValuesToPC = false;

string outboundCommands = "";

void setup() { 
  Serial.begin(SERIALSPEED);
  Serial.print("INITBEGIN");

  for (uint8_t i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);
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
  delay(FrequencyMS);
}

void reboot() {
#if MCU32U4
  wdt_disable();
  wdt_enable(WDTO_30MS);
  while (1) {}
#elif MCUA328P
  asm volatile ("  jmp 0");  
#endif
}

void updateSliderValues() {
  for (uint8_t i = 0; i < NUM_SLIDERS; i++) {
     analogSliderValues[i] = analogRead(analogInputs[i]);
  }
}

void sendSliderValues() {
  for (uint8_t i = 0; i < NUM_SLIDERS; i++) {
    Serial.print(analogSliderValues[i]);

    if (i < NUM_SLIDERS - 1) {
      Serial.print("|");
    }
  }
  if outboundCommands != "" {
    serial.print(":");
    serial.print(outboundCommands);
    outboundCommands = "";
  }

  Serial.println();
}

void addCommand(String cmd) {
  if outboundCommands != "" {
    outboundCommands += "|"
  }
  outboundCommands += cmd;
}

void printSliderValues() {
  for (uint8_t i = 0; i < NUM_SLIDERS; i++) {
    Serial.print("Slider #"+ String(i + 1) + ": " + String(analogSliderValues[i]) + " mV");

    if (i < NUM_SLIDERS - 1) {
      Serial.print(" | ");
    } else {
      Serial.println();
    }
  }
}

void checkForCommand() {
  //Check if data is waiting
  if (Serial.available() > 0) {
    //Get start time of command
    unsigned long timeStart = millis();

    //Get data from Serial
    String input = Serial.readStringUntil('\n');  // Read chars from serial monitor

    //If data takes to long
    if(millis()-timeStart >= SerialTimeout) {
      Serial.println("TIMEOUT");
      return;
    }

    // Check and match commands
    else {

      // Start Sending Slider Values
      if ( input.equalsIgnoreCase("deej.core.start") == true ) {
        pushSliderValuesToPC = true;
      }

      // Stop Sending Slider Values
      else if ( input.equalsIgnoreCase("deej.core.stop") == true ) {
        pushSliderValuesToPC = false;
      }
      
      // Send Single Slider Values
      else if ( input.equalsIgnoreCase("deej.core.values") == true ) {
        sendSliderValues();
      }

      // Send Human Readable Slider Values 
      else if ( input.equalsIgnoreCase("deej.core.values.HR") == true ) {
        printSliderValues();
      }

      else if ( input.equalsIgnoreCase("deej.core.reboot") == true ) {
        reboot();
      }

      //Default Catch all
      else {
        Serial.println("INVALIDCOMMANDS");
      }
    }
  }
}
