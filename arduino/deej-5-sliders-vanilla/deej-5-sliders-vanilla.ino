#define BUTTON_PIN 3
#define BUTTON_LED_PIN 2

const int NUM_SLIDERS = 5;
const int analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};

int analogSliderValues[NUM_SLIDERS];

volatile int ledState = LOW;
int currentButtonState;
int lastButtonState;

void setup()
{
  pinMode(BUTTON_PIN, INPUT_PULLUP);
  pinMode(BUTTON_LED_PIN, OUTPUT);

  for (int i = 0; i < NUM_SLIDERS; i++)
  {
    pinMode(analogInputs[i], INPUT);
  }

  Serial.begin(9600);
}

void loop()
{
  handleButtonPress();
  updateSliderValues();
  sendSliderValues(); // Actually send data (all the time)
  // printSliderValues(); // For debug
  delay(10);
}

void handleButtonPress()
{
  lastButtonState = currentButtonState;
  currentButtonState = digitalRead(BUTTON_PIN);

  if (lastButtonState == HIGH && currentButtonState == LOW)
  {
    ledState = !ledState;
    Serial.println("BUTTON_PRESSED");
    digitalWrite(BUTTON_LED_PIN, ledState);
  }
}

void updateSliderValues()
{
  for (int i = 0; i < NUM_SLIDERS; i++)
  {
    analogSliderValues[i] = analogRead(analogInputs[i]);
  }
}

void sendSliderValues()
{
  String builtString = String("");

  for (int i = 0; i < NUM_SLIDERS; i++)
  {
    builtString += String((int)analogSliderValues[i]);

    if (i < NUM_SLIDERS - 1)
    {
      builtString += String("|");
    }
  }
  Serial.println(builtString);
}

void printSliderValues()
{
  for (int i = 0; i < NUM_SLIDERS; i++)
  {
    String printedString = String("Slider #") + String(i + 1) + String(": ") + String(analogSliderValues[i]) + String(" mV");
    Serial.write(printedString.c_str());

    if (i < NUM_SLIDERS - 1)
    {
      Serial.write(" | ");
    }
    else
    {
      Serial.write("\n");
    }
  }
}