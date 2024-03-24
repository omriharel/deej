#include "MultiMap.h"
enum PotentiometerType {
  LINEAR,
  LOGARITHMIC
};

const int NUM_SLIDERS = 5;
const int analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};
const PotentiometerType POTENTIOMETER_TYPE = LINEAR; //LINEAR OR LOGARITHMIC

int analogSliderValues[NUM_SLIDERS];

void setup() { 
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);
  }

  Serial.begin(9600);
}

void loop() {
  updateSliderValues();
  sendSliderValues(); // Actually send data (all the time)
  // printSliderValues(); // For debug
  delay(10);
}

void updateSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
    if(POTENTIOMETER_TYPE == 1) {
     analogSliderValues[i] = logarithmicToLinearValue(analogRead(analogInputs[i]));
    } else {
      analogSliderValues[i] = analogRead(analogInputs[i]);
    }
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

int inputMap[]  = {0, 1, 4, 15, 27, 56, 83, 185, 308, 520, 720, 979, 1023};
int outputMap[] = {0, 89, 178, 267, 356, 445, 534, 623, 712, 801, 890, 979, 1023};

int logarithmicToLinearValue(int logarithmicValue) {
  int linearValue = multiMap<int>(logarithmicValue, inputMap, outputMap, 13);
  return linearValue;
}
