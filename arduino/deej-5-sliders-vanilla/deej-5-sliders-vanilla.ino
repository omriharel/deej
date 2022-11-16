const int NUM_SLIDERS = 5;
const int MIN_DELTA = 2;
const int analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};

int analogSliderValues[NUM_SLIDERS];
bool doUpdate = false;

void setup() { 
  for (int i = 0; i < NUM_SLIDERS; i++) {
    pinMode(analogInputs[i], INPUT);
  }

  Serial.begin(9600);
}

void loop() {
  updateSliderValues();
  if(doUpdate)
  {
    sendSliderValues();
    doUpdate = false;
  }

  // printSliderValues(); // For debug
  delay(10);
}

void updateSliderValues() {
  int newVal = 0;

  for (int i = 0; i < NUM_SLIDERS; i++)
  {
    newVal = analogRead(analogInputs[i]);

    // Debounce serial comms
    if(abs(newVal - analogSliderValues[i]) >= MIN_DELTA)
    {
      analogSliderValues[i] = newVal;
      doUpdate = true;
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
