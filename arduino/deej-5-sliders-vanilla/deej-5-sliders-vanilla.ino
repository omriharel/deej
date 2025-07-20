const int NUM_POTS = 5;
const int analogInputs[NUM_POTS] = {A0, A1, A2, A3, A4};

int analogSliderValues[NUM_POTS];

void setup() { 
  for (int i = 0; i < NUM_POTS; i++) {
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
  for (int i = 0; i < NUM_POTS; i++) {
     analogSliderValues[i] = analogRead(analogInputs[i]);
  }
}

void sendSliderValues() {
  String builtString = String("");

  for (int i = 0; i < NUM_POTS; i++) {
    builtString += String((int)analogSliderValues[i]);

    if (i < NUM_POTS - 1) {
      builtString += String("|");
    }
  }
  
  Serial.println(builtString);
}

void printSliderValues() {
  for (int i = 0; i < NUM_POTS; i++) {
    String printedString = String("Slider #") + String(i + 1) + String(": ") + String(analogSliderValues[i]) + String(" mV");
    Serial.write(printedString.c_str());

    if (i < NUM_POTS - 1) {
      Serial.write(" | ");
    } else {
      Serial.write("\n");
    }
  }
}
