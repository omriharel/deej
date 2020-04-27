////////////////////////////////////////////////////////////////////////
// Charge Pump Regulator
////////////////////////////////////////////////////////////////////////
#define OLED_CHARGEPUMP                               0x8D
#define OLED_CHARGEPUMP_ON                            0x14
#define OLED_CHARGEPUMP_OFF                           0x10
////////////////////////////////////////////////////////////////////////
// Timing and Driving Scheme Settings
////////////////////////////////////////////////////////////////////////
#define OLED_SETDISPLAYCLOCKDIV                       0xD5
#define OLED_SETPRECHARGE                             0xD9
#define OLED_SETVCOMDESELECT                          0xDB
#define OLED_NOP                                      0xE3
////////////////////////////////////////////////////////////////////////
// Hardware Configuration
////////////////////////////////////////////////////////////////////////
// 40-7F - set address startline from 0-127 (6-bits)
#define OLED_SETSTARTLINE_ZERO                        0x40
// Y Direction
#define OLED_SEGREMAPNORMAL                           0xA0
#define OLED_SEGREMAPINV                              0xA1
#define OLED_SETMULTIPLEX                             0xA8
// 0xA8, number of rows -1 ... e.g. 0xA8, 63
// X Direction
#define OLED_COMSCANINC                               0xC0
#define OLED_COMSCANDEC                               0xC8
// double byte with image wrap ...probably should be 0
#define OLED_SETDISPLAYOFFSET                         0xD3
// Double Byte Hardware com pins configuration
#define OLED_SETCOMPINS                               0xDA
// legal values 0x02, 0x12, 0x022, 0x032
////////////////////////////////////////////////////////////////////////
// Address Setting Command Table
////////////////////////////////////////////////////////////////////////
// 00-0F - set lower nibble of page address
// 10-1F - set upper niddle of page address
#define OLED_SETMEMORYMODE                            0x20
#define OLED_SETMEMORYMODE_HORIZONTAL                 0x00
#define OLED_SETMEMORYMODE_VERTICAL                   0x01
#define OLED_SETMEMORYMODE_PAGE                       0x02
// 0x20 + 00 = horizontal, 01 = vertical 2= page >=3=illegal
// Only used for horizonal and vertical address modes
#define OLED_SETCOLUMNADDR                            0x21
// 2 byte Parameter
// 0-127 column start address 
// 0-127 column end address
#define OLED_SETPAGEADDR                              0x22
// 2 byte parameter
// 0-7 page start address
// 0-7 page end Address
// 0xB0 -0xB7 ..... Pick page 0-7
////////////////////////////////////////////////////////////////////////
// Fundamental Command Table Page 28
////////////////////////////////////////////////////////////////////////
#define OLED_SETCONTRAST                              0x81
// 0x81 + 0-0xFF Contrast ... reset = 0x7F
 
// A4/A5 commands to resume displaying data
// A4 = Resume to RAM content display
// A5 = Ignore RAM content (but why?)
#define OLED_DISPLAYALLONRESUME                       0xA4
#define OLED_DISPLAYALLONIGNORE                       0xA5
 
// 0xA6/A7 Normal 1=white 0=black Inverse 0=white  1=black
#define OLED_DISPLAYNORMAL                            0xA6
#define OLED_DISPLAYINVERT                            0xA7
 
// 0xAE/AF are a pair to turn screen off/on
#define OLED_DISPLAYOFF                               0xAE
#define OLED_DISPLAYON                                0xAF
////////////////////////////////////////////////////////////////////////
// Scroll
////////////////////////////////////////////////////////////////////////
#define OLED_DEACTIVATE_SCROLL                        0x2E