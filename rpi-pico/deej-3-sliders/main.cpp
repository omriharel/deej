// TESTED ON RPI PICO W

#include <stdio.h>
#include "pico/stdlib.h"
#include "hardware/gpio.h"
#include "hardware/adc.h"

const int NUM_SLIDERS = 3;
const int ANALOG_PINS[NUM_SLIDERS] = {26, 27, 28};

uint16_t analogSliderValues[NUM_SLIDERS];

void updateSliderValues();
void sendSliderValues();

int main()
{
    stdio_init_all();

    adc_init();

    for (int i = 0; i < NUM_SLIDERS; i++)
    {
        adc_gpio_init(ANALOG_PINS[i]);
    }

    while (true)
    {
        updateSliderValues();
        sendSliderValues();
        sleep_ms(10);
    }

    return 0;
}

void updateSliderValues()
{
    for (int i = 0; i < NUM_SLIDERS; i++)
    {
        adc_select_input(i);
        analogSliderValues[i] = adc_read();
    }
}

void sendSliderValues()
{
    for (int i = 0; i < NUM_SLIDERS; i++)
    {
        printf("%d", analogSliderValues[i]);

        if (i < NUM_SLIDERS - 1)
        {
            printf("|");
        }
    }
    printf("\n");
}
