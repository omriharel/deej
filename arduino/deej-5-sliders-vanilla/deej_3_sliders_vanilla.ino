constexpr byte channels = 3;
constexpr byte packLen = 4 + channels * 2;
constexpr byte analogInputs[] = {A0,A1,A2};
constexpr byte updatemillis = 17;

namespace iBus{
  unsigned short checksum;
  void packstart(){
    checksum = 0xffff - packLen - 0x40;
    Serial.write(packLen);
    Serial.write(0x40);
  }
  void packend(){
    Serial.write(checksum & 0xff);
    Serial.write(checksum >> 8);
  }
  void write(unsigned short data){
    byte b = data & 0xff;
    Serial.write(b);
    checksum -= b;
    b = data >> 8;
    Serial.write(b);
    checksum -= b;
  }
}

void setup() {
  // put your setup code here, to run once:
  Serial.begin(9600);
  pinMode(LED_BUILTIN, OUTPUT);
}

void loop() {
  unsigned long t = millis();
  iBus::packstart();
  for(int i = 0; i < channels; i++){
    iBus::write(analogRead(analogInputs[i]));
  }
  iBus::packend();
  t = millis() - t;
  if(t < updatemillis){
    delay(updatemillis - t);
  }
}
