env GOOS=linux GOARCH=arm64 go build -o smartie

MAC

pmset – manipulate power management settings
smc - Apple System Management Control (SMC)cd 

https://github.com/hholtmann/smcFanControl/tree/master/smc-command


Überschuss (Aktion immer nur alle 1min)

Lauf 1
- Suche Laptop mit geringster Battery Kapa
- Battery Maintain 80%
- Plug on 

Lauf 2
- Suche alle Plugs die aus sind.
- Sortierung nach Prio
  - 
- erstes Plug einschalten
- return

Bezug (Aktion immer nur alle 1min)

- Suche alle Plugs die an sind.
- Sortierung nach lowest Prio
- erstes Plug ausschalten
- return

Laptop leer (20%)
- Battery maintain 20%
- Plug on


Prio

0,5 / 2

1 / 2

Faktor * On-Aktionen (per day)
Laptop mit weniger Akku



Subjects

shellies.plug.*.*.relay.>
shellies.plug.*.*.relay.>
shellies.plug.*.*.relay.>
shellies.plug.*.*.relay.>

tele.*.SENSOR

smartie.laptop.air.charge
smartie.laptop.NB-



Install



brew tap insidegreen/smartie
brew install smartie
brew install battery
brew services start smartie 

open battery.app