//Required Libraties
#include <TM1637.h>
#include <Keyboard.h>
#include <EEPROM.h>
#include <avr/wdt.h>
//Setup Display
#define CLK 8
#define DIO 9
TM1637 tm1637(CLK,DIO);
//Setup Inputs
const int NUM_SLIDERS = 5;
const int NUM_BUTTONS = 5;
const int avgSize = 10;
const int analogInputs[NUM_SLIDERS] = {A3, A2, A1, A0, A10};
const int digitalInputs[NUM_BUTTONS] = {2,3,4,5,6};
const int startAddr = 1;
//Button Variables
int buttonState[NUM_BUTTONS];
int oldButtonState[NUM_BUTTONS];
int loopTimer = 0;
//Slider Variables
unsigned int num[4];
unsigned int oldVolume[NUM_SLIDERS];
unsigned int newVolume[NUM_SLIDERS];
unsigned long changeTime[NUM_SLIDERS] = {0,0,0,0,0};
unsigned long oldMillis = 0;
unsigned long buttonHoldTime[NUM_BUTTONS];
unsigned long buttonPressTime[NUM_BUTTONS];
unsigned int analogSliderValues[NUM_SLIDERS];

void setup() { 
  wdt_disable();
  wdt_reset();
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);//Set analog pins to input    
    oldVolume[i] = knobRead(analogInputs[i]);//Get initial values
  }
  for (int i = 0; i < NUM_BUTTONS; i++) {
    pinMode(digitalInputs[i], INPUT_PULLUP);//Set digital pins to inputs   
  }  
  tm1637.init();//Initialize Display
  tm1637.set(7);//Set brightness
  Serial.begin(9600);//open serial bus
  if(EEPROM.read(startAddr)){ //checks autostart eeprom value
    launchDeej(1000); //autostarts deej
  }
}

void loop() {
  updateSliderValues();//Gets slider position
  sendSliderValues(); //Actually send data (all the time)
  buttonActions(); //Updates buttons states and actions
  getDisplayValues(); //Updates Display
  // printSliderValues();// For debug 
}

void updateSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
     analogSliderValues[i] = knobRead(analogInputs[i]); //updates slider values
  }
}

void sendSliderValues() {
  String builtString = String(""); //initalizes string

  for (int i = 0; i < NUM_SLIDERS; i++) {
    builtString += String((int)analogSliderValues[i]); //adds slider values to string

    if (i < NUM_SLIDERS - 1) {
      builtString += String("|"); //seperates string 
    }
  }
  if(Serial.availableForWrite()>32){ //makes sure serial bus is conencted
    Serial.println(builtString); //writes string to serial bus
  }
}

void buttonActions(){ //updates button states and runs functions
  for(int i = 0; i<NUM_BUTTONS; i++){ 
    oldButtonState[i]=buttonState[i]; //updates last button state  
    buttonState[i]=digitalRead(digitalInputs[i]); //gets current button state
    //Serial.println(buttonState[i]); //debug
  }
  buttonPressActions(); //runs button press actions
  buttonHoldActions(); //runs button hold actions
}

void buttonHoldActions(){
  for(int i = 0; i<NUM_BUTTONS; i++){ 
    if(buttonState[i]==0){ //if button is pressed
      buttonHoldTime[i]=millis()-buttonPressTime[i]; //upate buttom press duration
      //Serial.println(buttonHoldTime[i]); //debug
    }
    else{
      buttonHoldTime[i]=0; //reset button press duration
      buttonPressTime[i] = millis(); //resets button press time, makes the hold continuous
    }
   }
   
   if(buttonHoldTime[0]>3000){ //if button held for 3 seconds
      for(int i = 3; i>=0; i--){
        tm1637.display(i,14); //make display change to EEEE
      }
      delay(500); //hold display at EEEE
      reset(); //resets arduino
      buttonHoldTime[0]=0; //resets button hold time
      buttonPressTime[0]=millis(); //stops button press from triggering twice back to back
   }

   if(buttonHoldTime[4]>3000){
      for(int i = 3; i>=0; i--){
        tm1637.display(i,14); //changes display to EEEE
      }
      delay(500); //holds display
      eepromToggle(startAddr); //toggles autoStart EEPROM value
      buttonHoldTime[4]=0; //resets button hold time
      buttonPressTime[4]=millis(); //resets button press time
   }
       
  
}

void buttonPressActions(){    
  if(buttonState[0]==0 && oldButtonState[0] == 1){
    //loopTimer=variableToggle(loopTimer); //debug
    buttonPressTime[0]=millis(); //sets button press time
  }
  if(buttonState[1]==0 && oldButtonState[1] == 1){
    launchDeej(10); //launches deej
    buttonPressTime[1]=millis(); //sets button press time
  }
  if(buttonState[2]==0 && oldButtonState[2] == 1){
    //wdt_enable(WDTO_15MS);
    //while(1);
    buttonPressTime[2]=millis(); //sets button press time
  }
  if(buttonState[3]==0 && oldButtonState[3] == 1){

    buttonPressTime[3]=millis(); //sets button press time
  }
  if(buttonState[4]==0 && oldButtonState[4] == 1){
    outputToggle(); //toggles output device, uses keybinds and external software
    buttonPressTime[4]=millis(); //sets button press time
  }
}

void getDisplayValues(){
    for (int i = 0; i<NUM_SLIDERS; i++){
      newVolume[i] = analogSliderValues[i]; //gets slider values
      if(abs(newVolume[i]-oldVolume[i])>=50){ //if change is significant
        //updateDisplay(i);
        //oldVolume[i]=newVolume[i];
        changeTime[i] = millis(); //reset change timer
        changeTime[i] = changeTime[i] + 3000; //add 3 seconds since last change
      }  
      if(changeTime[i]>millis() && abs(newVolume[i]-oldVolume[i])>=15){ //updates display for smaller changes during 3 seconds window without resetting window
        updateDisplay(i); //updates display
      }
      else if (changeTime[0]<millis() && changeTime[1]<millis() && changeTime[2]<millis() && changeTime[3]<millis() && changeTime[4]<millis()){ //if all windows are past
        blankDisplay(); //blank display
        //Serial.println("Off"); //debug
      }
  }
}

void updateDisplay(int i){  //update display
  oldVolume[i] = newVolume[i]; //sets oldVolume to newVolume  
  num[3] = int(newVolume[i]/10.23)/1U % 10; //gets each digit of display
  num[2] = int(newVolume[i]/10.23)/10U % 10;
  num[1] = int(newVolume[i]/10.23)/100U % 10;
  num[0] = (i+1); //sets first digit to slider #
  for (int j = 3; j>=0; j--){
    if(j==1 && num[1]==0){
       tm1637.display(j,34); //leaves leading zero blank
    }
    else{
      tm1637.display(j,num[j]); //sets display to most recent change
    }
  }    
}

void updateDisplayBeta(int number){
  num[3] = number/1U %10; //gets each digit
  num[2] = number/10U %10;
  num[1] = number/100U %10;
  num[0] = number/1000U %10;
  
  for(int i=3; i>=0; i--){
    tm1637.display(i,num[i]); //updates display
  }
}
void blankDisplay(){
  for (int j = 3; j>=0; j--){
    /*if (loopTimer==1){ //debug
      updateDisplayBeta(loopTime());
      if(Serial.availableForWrite()>32){
        Serial.println(loopTime());
      }
      j=0;
    }*/
    if(Serial.availableForWrite()<32){ //makes sure serial bus is connected
      tm1637.display(j,14); //blanks display
    }
    else{
      tm1637.display(j,34); //sets display to error if disconnected from bus
      //tm1637.display(j,random(9)); //debug
    }
  }
}

void launchDeej(int wait){ //laucnhes deej
  delay(wait); //waits for USB connection
  Keyboard.press(KEY_LEFT_GUI); //opens run dialog
  Keyboard.press('r');
  delay(100); //waits
  Keyboard.releaseAll(); //releases keys
  Keyboard.println("C:\\Zip Programs\\deej-master\\run.vbs"); //types path to deej run
  delay(100);
}



void eepromToggle(int addr){ //eeprom toggle
  if(EEPROM.read(addr)==0){ //if 0
    EEPROM.write(addr,1); //make 1
  }
  else{
    EEPROM.write(addr,0); //else 0
  }
}

int variableToggle(int var){ //variable toggle
  if (var == 0){ //if 0
      var = 1; //make 1
    }
    else{
      var = 0; //else 0
    }
    return var; //return var
}

int knobRead(int knob){ //smooth knob read
  unsigned int analogAverage = 0; //gets knob variable
  for (int i = 0; i<avgSize; i++){ //for avgSize iterations
    analogAverage = analogRead(knob) + analogAverage; //take analog value
    delay(1); //every millisecond
  }
  return analogAverage/avgSize; //return avarage
}

void reset(){
  wdt_enable(WDTO_15MS); //enables watchdog timer
  while(1); //forces watchdog to reset
}

void outputToggle(){ //toggle output
  Keyboard.press(KEY_LEFT_CTRL); //active shortcut
  Keyboard.press(KEY_LEFT_ALT);
  Keyboard.press(KEY_LEFT_SHIFT);
  Keyboard.press('p');
  delay(100);
  Keyboard.releaseAll(); //release keys
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

/*int loopTime(){ //used for debug
  int loops = millis()-oldMillis; //gets time from last time function was run
  oldMillis=millis(); //sets old time to current time
  return loops; //returns loop time
}*/
