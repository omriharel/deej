#include <RotaryEncoder.h>
#include <AisperButton.h>

#define NUM_POTS 3
#define NUM_BUTTONS 5
#define NUM_ENCODERS 2

#define LONG_CLICK_TIME 1000
#define MAX_TIME_BETWEEN_CLICKS 300

#define DATA_RESOLUTION 1024
#define DENOIZE 8

#define DATA_SEND_THRESHOLD 5
#define START_MARKER 0xAA
#define END_MARKER 0x55

const int potPins[NUM_POTS] = { A1, A2, A3 };
const int buttonPins[NUM_BUTTONS] = { 14, 15, 18, 4, 7 };
const int encoderPins[NUM_ENCODERS * 2] = { 2, 3, 5, 6 };

struct VolData {
  int value;
  bool toggleMute = false;
};

VolData lastSentValues[NUM_POTS + NUM_ENCODERS];
VolData values[NUM_POTS + NUM_ENCODERS];
Button* buttons[NUM_BUTTONS];
RotaryEncoder* encoders[NUM_ENCODERS];

void setup() {
  Serial.begin(9600);
  delay(300);
  Init();
}

void Init() {
  for (int i = 0; i < NUM_POTS; i++) {
    pinMode(potPins[i], INPUT);
  }

  for (int i = 0; i < NUM_BUTTONS; i++) {
    buttons[i] = new Button(buttonPins[i], MAX_TIME_BETWEEN_CLICKS, LONG_CLICK_TIME, true);
  }

  for (int i = 0; i < NUM_ENCODERS; i++) {
    encoders[i] = new RotaryEncoder(encoderPins[i * 2], encoderPins[i * 2 + 1], RotaryEncoder::LatchMode::TWO03);
  }
}

void loop() {
  for (int i = 0; i < NUM_BUTTONS; i++) {
    tickButton(i);
  }

  for (int i = 0; i < NUM_ENCODERS; i++) {
    tickEncoder(i);
  }

  tickPots();

  trySendValues();
  //printValues();
}

void setValue(uint8_t index, int newValue) {
  int clamped = constrain(newValue, -(DATA_RESOLUTION - 1), DATA_RESOLUTION - 1);
  values[index].value = clamped;
}

int getValue(uint8_t index) {
  return values[index].toggleMute ? 0 : values[index].value;
}

void tickEncoder(uint8_t index) {
  RotaryEncoder* encoder = encoders[index];

  encoder->tick();

  const RotaryEncoder::Direction direction = encoder->getDirection();

  const int newValue = (int)direction * 22;

  const int valueIndex = NUM_POTS + index;

  if (values[valueIndex].value != newValue) {
    setValue(valueIndex, newValue);
  }
}

void tickButton(uint8_t index) {
  Button* button = buttons[index];

  button->tick();

  if (button->isPressed() && button->currentStateTime() >= 300 && !values[index].toggleMute) {
    values[index].toggleMute = true;
    return;
  }

  if (button->isThereAnEvent()) {
    values[index].toggleMute = !values[index].toggleMute;

    button->ClearEvent();
  }
}

void tickPots() {
  for (int i = 0; i < NUM_POTS; i++) {
    int rawValue = analogRead(potPins[i]);
    int mappedValue = rawValue;

    if (abs(abs(mappedValue) - abs(values[i].value)) > DENOIZE) {
      if (mappedValue >= 1000) {
        mappedValue = 1023;
      }
      if (mappedValue <= 10) {
        mappedValue = 0;
      }

      setValue(i, mappedValue);
    }
  }
}

void trySendValues() {
  bool shouldSendValues = false;

  for (int i = 0; i < NUM_POTS + NUM_ENCODERS; i++) {
    const int minValue = min(lastSentValues[i].value, values[i].value);
    const int maxValue = max(lastSentValues[i].value, values[i].value);

    // using threshold only for pots because encoders have synthetic data that can be compared raw
    const bool changeSignificantEnough = i < NUM_POTS ? abs(maxValue - minValue) >= DATA_SEND_THRESHOLD : lastSentValues[i].value != values[i].value;

    if (changeSignificantEnough || values[i].toggleMute) {
      shouldSendValues = true;
      break;
    }
  }

  if (shouldSendValues) {
    sendValues();

    for (int i = 0; i < NUM_POTS + NUM_ENCODERS; i++) {
      values[i].toggleMute = false;
    }
  }
}

const uint8_t FRAME_SIZE = (NUM_POTS + NUM_ENCODERS);

void sendValues() {
  uint8_t buf[FRAME_SIZE * 2 + 2];

  int idx = 0;

  buf[idx++] = START_MARKER;

  for (int i = 0; i < FRAME_SIZE; i++) {
    uint16_t packed = ((values[i].toggleMute & 0x01) << 11) | ((uint16_t)values[i].value & 0x07FF);

    buf[idx++] = (packed >> 8) & 0xFF;
    buf[idx++] = packed & 0xFF;

    lastSentValues[i] = values[i];
  }

  buf[idx++] = END_MARKER;

  Serial.write(buf, idx);
}

void printValues() {
  bool shouldSendValues = false;

  for (int i = 0; i < NUM_POTS + NUM_ENCODERS; i++) {
    const int minValue = min(lastSentValues[i].value, values[i].value);
    const int maxValue = max(lastSentValues[i].value, values[i].value);

    // using threshold only for pots because encoders have synthetic data that can be compared raw
    const bool changeSignificantEnough = i < NUM_POTS ? abs(maxValue - minValue) >= DATA_SEND_THRESHOLD : lastSentValues[i].value != values[i].value;

    if (changeSignificantEnough || values[i].toggleMute) {
      shouldSendValues = true;
      break;
    }
  }

  if (!shouldSendValues) {
    return;
  }

  for (int i = 0; i < NUM_POTS + NUM_ENCODERS; i++) {
    String printedString = String("Value #") + String(i + 1) + String(": ") + String(getValue(i));

    if (values[i].toggleMute) {
      printedString += "(m)";
    }

    Serial.write(printedString.c_str());

    if (i < NUM_POTS + NUM_ENCODERS - 1) {
      Serial.write(" | ");
    } else {
      Serial.write("\n");
    }

    lastSentValues[i] = values[i];
  }
}
