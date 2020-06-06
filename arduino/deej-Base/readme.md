# Arduino Deej Base Code
## The Basis for all modules to be built

#### Here's a summary

The Arduino is constantly listening for data to be sent to its RX port When it receives a byte it then starts to look for a newline character. The newline character marks the end of a command. After finding the new line it looks through its known command list and if it finds a match execute the appropriate code

#### Commands
##### deej.core.start
starts up a constant stream of data from the arduino to the host pc. The values are sent as 'x|x|...|x' with x being the analog value
##### deej.core.stop
##### deej.core.values
Sends one set of values sent as 'x|x|...|x' with x being the analog value
##### deej.core.values.hr
Sends one set of values as 'Slider #n:x mv | Slider #n:x mv | ... | Slider #n:x mv' with n being the slider number and x being the value
##### deej.core.reboot
Reboots the microcontroler (the serial port will have to be reopened)