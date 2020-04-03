//Required Libraties
#include <TM1637.h>
#include <Keyboard.h>
//Setup Display
#define CLK 8
#define DIO 9
TM1637 tm1637(CLK,DIO);
//Setup Inputs
const int NUM_SLIDERS = 5;
const int NUM_BUTTONS = 5;
const int analogInputs[NUM_SLIDERS] = {A3, A2, A1, A0, A10};
const int digitalInputs[NUM_BUTTONS] = {2,3,4,5,6};
//Button Variables
int buttonState[NUM_BUTTONS];
int oldButtonState[NUM_BUTTONS];
//Slider Variables
int num[4];
int oldVolume[NUM_SLIDERS];
int newVolume[NUM_SLIDERS];
float changeTime[NUM_SLIDERS] = {0,0,0,0,0};
int analogSliderValues[NUM_SLIDERS];

void setup() { 
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);//Set analog pins to input    
    oldVolume[i] = analogRead(analogInputs[i]);//Get initial values
  }
  for (int i = 0; i < NUM_BUTTONS; i++) {
    pinMode(digitalInputs[i], INPUT_PULLUP);//Set digital pins to inputs   
  }  
  tm1637.init();//Initialize Display
  tm1637.set(7);//Set brightness
  Serial.begin(9600);//open serial bus
  delay(2000);
  Keyboard.press(KEY_LEFT_GUI);
  Keyboard.press('r');
  delay(100);
  Keyboard.releaseAll();
  Keyboard.println("C:\\Zip Programs\\deej-master\\run.vbs");
  delay(100);
}

void loop() {
  updateSliderValues();//Gets slider position
  sendSliderValues(); //Actually send data (all the time)
  buttonActions(); //Updates buttons states and actions
  getDisplayValues(); //Updates Display
  // printSliderValues();// For debug
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

void buttonActions(){
  
  for(int i = 0; i<NUM_BUTTONS; i++){ 
    oldButtonState[i]=buttonState[i];   
    buttonState[i]=digitalRead(digitalInputs[i]);
    //Serial.println(buttonState[i]);
  }
  if(buttonState[0]==0 && oldButtonState[0] == 1){
    Keyboard.print('q');
  }
  if(buttonState[1]==0 && oldButtonState[1] == 1){
    Keyboard.print('w');
  }
  if(buttonState[2]==0 && oldButtonState[2] == 1){
    Keyboard.print('e');
  }
  if(buttonState[3]==0 && oldButtonState[3] == 1){
    Keyboard.print('r');
  }
  if(buttonState[4]==0 && oldButtonState[4] == 1){
    Keyboard.press(KEY_LEFT_CTRL);
    Keyboard.press(KEY_LEFT_ALT);
    Keyboard.press(KEY_LEFT_SHIFT);
    Keyboard.press('P');
    delay(100);
    Keyboard.releaseAll();
    
  }
}

void getDisplayValues(){
    for (int i = 0; i<NUM_SLIDERS; i++){
    newVolume[i] = analogSliderValues[i];
    if(abs(newVolume[i]-oldVolume[i])>=35){
      //updateDisplay(i);
      changeTime[i] = millis();
      changeTime[i] = changeTime[i] + 3000; 
    }  
    if(changeTime[i]>millis() && abs(newVolume[i]-oldVolume[i])>=15){
      updateDisplay(i);
    }
    else if (changeTime[0]<millis() && changeTime[1]<millis() && changeTime[2]<millis() && changeTime[3]<millis() && changeTime[4]<millis()){    
      for (int j = 3; j>=0; j--){      
        tm1637.display(j,34); //sets display to most recent change
      }
      //Serial.println("Off");
    }
  }
}

void updateDisplay(int i){
  
  oldVolume[i] = newVolume[i];
  
  num[3] = int(newVolume[i]/10.23)/1U % 10;
  num[2] = int(newVolume[i]/10.23)/10U % 10;
  num[1] = int(newVolume[i]/10.23)/100U % 10;
  num[0] = (i+1);
  for (int j = 3; j>=0; j--){
    if(j==1 && num[1]==0){
       tm1637.display(j,34);
    }
    else{
      tm1637.display(j,num[j]); //sets display to most recent change
    }
  }    
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
