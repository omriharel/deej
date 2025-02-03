/*
* Espruino 2v25 sketch
* */

var inputs = [new Pin(3), new Pin(6), new Pin(7), new Pin(11), new Pin(5)]; // please set your pins here
var setup = function () {
    for (var _i = 0, inputs_1 = inputs; _i < inputs_1.length; _i++) {
        var pin = inputs_1[_i];
        pinMode(pin, 'analog', true);
    }
    Serial1.setup(9600, {});
};
var loop = function () {
    var serialString = inputs
        .map(function (pin) {
        var pinValue = Math.round(analogRead(pin) * 1000) / 1000;
        return Math.floor(pinValue * 1023) || 0;
    })
        .join('|');
    Serial1.println(serialString);
};
setup();
setInterval(loop, 10);
