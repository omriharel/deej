#include <TM1637.h>

#define CLK 12
#define DIO 11
TM1637 tm1637(CLK,DIO);


const int NUM_SLIDERS = 3;
const int analogInputs[NUM_SLIDERS] = {A1,A2,A3};
int num[3];
const int displayTime = 5000; //display on time, in millis

int analogSliderValues[NUM_SLIDERS];
int oldVolumes[NUM_SLIDERS];
int newVolumes[NUM_SLIDERS];
int changeTime[2];

void setup() {
  
  tm1637.init();
  tm1637.set(7);//BRIGHT_TYPICAL = 2,BRIGHT_DARKEST = 0,BRIGHTEST = 7; 
  
  for (int i = 0; i < NUM_SLIDERS; i++) { //setup Inputs
    pinMode(analogInputs[i], INPUT); 
   // pinMode(digitalInputs[i+2], INPUT_PULLUP);
  }

  for (int i = 0; i < NUM_SLIDERS; i++) {
     oldVolumes[i] = analogRead(analogInputs[i]); //poll initial input values
  }
  
  Serial.begin(9600);
}

void loop() {
  updateSliderValues(); //update resistor values
 // buttonPress(); //update button status
  updateDisplay(); //update display
  sendSliderValues(); // Actually send data (all the time)
  // printSliderValues(); // For debug
  delay(10);
}

void updateDisplay() {    
  if (changeTime[1]<millis()+displayTime) { //checks display timer
    tm1637.set(7); //turns on display
    num[3] = newVolumes[changeTime[2]]/1U % 10;
    num[2] = newVolumes[changeTime[2]]/10U % 10;
    num[1] = newVolumes[changeTime[2]]/100U % 10;
    num[0] =newVolumes[changeTime[2]]/1000U % 10;
    for (int i = 0; i<=3; i++){
      
    tm1637.display(i,num[i]); //sets display to most recent change
    }
  }
  else 
  {
    tm1637.set(7); //powers off display
  }  
}

void updateSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
     analogSliderValues[i] = analogRead(analogInputs[i]);
     newVolumes[i] = int(analogSliderValues[i]/10.23);
     if (oldVolumes[i] != newVolumes[i]){
      changeTime[1]=millis();
      changeTime[2]=i; 
    }
    oldVolumes[i] = newVolumes[i];
  }
}

void sendSliderValues() {
  String builtString = String("");

  for (int i = 0; i < NUM_SLIDERS; i++) {
    builtString += String((int)analogSliderValues[i]);

    if (i < NUM_SLIDERS) {
      builtString += String("|");
    }
  }  
  Serial.println(builtString);
}

/*void printSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
    String printedString = String("Slider #") + String(i + 1) + String(": ") + String(analogSliderValues[i]) + String(" mV");
    Serial.write(printedString.c_str());
    if (i < NUM_SLIDERS - 1) {
      Serial.write(" | ");
    } else {
      Serial.write("\n");
    }
  }
}*/
