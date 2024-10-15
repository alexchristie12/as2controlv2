# Control System Design Notes from Lexie

From our sensors we get a wide range of stuff:

- Temperature (in Celsius)
- Humidity (Relative)
- Moisture sensor
    - Dry, 0-300
    - Humid, 300-700
    - Wet, 700-950

We have four types of Grass with each of the following conditions:

- **Buffalo**: 20 degrees and upwards and can't fall below 10 degrees. 40%
  sunlight and 3-4 hours of sunlight per day
- **Couch**: 20 to 32 degrees and 80% sunlight
- **Zoyisa**: 15 to 20 degrees
- **Kikuyu**: 15 to 20 degrees and 45-75% RH

All four grass are drought since they are grown in North Queensland where
droughts are prominent. Since only the relative humidity range for ideal growth
for Kikuyu grass could only be found but all grass types portray similar
growing conditions for all grass types should have a realtive humidity between
45% and 75%.

## Permanent Wilting Point

Permanent wilting point, PWP, is the minimum amount of water in the soil that
the plant requires not to wilt. This is important for ideal plant growth as
each type of soil has a different PWP, with plants typically having a PWP of
1500kPa. There is a trendline in the document.

Summary of findings: soil moisture must stay above 20% to ensure that the grass
does not start to wilt. Grass has a higher humidty in the morning (morning dew)
and lower at 6PM. We have some ideal conditions:

- Moisture level must stay above 20%
- Temperature should remain between 15 and 30 degrees celcius
- Relative humidity should remain between 45 and 75%