# Solar Array & Hardware Specifications

## PV Array Configuration

The site at 1296 Wren Lake Drive, Dorset, ON uses a 400W array consisting of four 100W monocrystalline panels wired in a **2-Series, 2-Parallel (2S2P)** configuration.

```text
  String 1: [Panel 1 (100W)] ──(Series)── [Panel 2 (100W)]  (~36V–40V Vmp, 5.5A)
                   │                                │
                   ├───────────────(+)──────────────┤
                   │                                │  ===> Combined 2S2P Array
                   ├───────────────(-)──────────────┤       (36V–40V Vmp, 11.0A Imp)
                   │                                │
  String 2: [Panel 3 (100W)] ──(Series)── [Panel 4 (100W)]  (~36V–40V Vmp, 5.5A)
```

## Electrical Specifications

| Parameter | Single Panel | 2-Panel Series String | Combined 2S2P Array |
| :--- | :--- | :--- | :--- |
| **Peak Power ($P_{\text{max}}$)** | 100 W | 200 W | **400 W** |
| **Voltage at Max Power ($V_{\text{mp}}$)** | 18.0 V – 20.4 V | 36.0 V – 40.8 V | **36.0 V – 40.8 V** |
| **Current at Max Power ($I_{\text{mp}}$)** | 4.9 A – 5.5 A | 4.9 A – 5.5 A | **9.8 A – 11.0 A** |
| **Open-Circuit Voltage ($V_{\text{oc}}$)** | 21.6 V – 24.3 V | 43.2 V – 48.6 V | **43.2 V – 48.6 V** |
| **Short-Circuit Current ($I_{\text{sc}}$)** | 5.4 A – 5.9 A | 5.4 A – 5.9 A | **10.8 A – 11.8 A** |

## Charge Controller: Renogy Rover 20A MPPT

* **Model:** `RNG-CTRL-RVR20-CAN`
* **Max Solar Input Voltage ($V_{\text{oc}}$):** 100 V DC
* **Max Battery Charge Current:** 20 A
* **Nominal System Voltage:** 12 V / 24 V auto-sensing
* **Max Charging Power (12V Bank):** ~260 W – 288 W

### Over-Paneling Analysis

* **Nameplate Ratio:** 400 W array on a 20 A / 288 W max controller represents a **138% over-paneling ratio**.
* **Operational Benefit:** In northern climates (Dorset, ON: 45.186°N), over-paneling raises harvesting yields during morning/evening shoulder periods and heavy cloud cover. During clear summer midday hours, the controller automatically limits current to 20 A.

## Battery Chemistry Profiles

Default setpoints for a 12V system:

| Parameter | LiFePO4 (Lithium) | Sealed (AGM) | Gel | Flooded |
| :--- | :--- | :--- | :--- | :--- |
| **High Voltage Disconnect** | 14.8 V | 16.0 V | 16.0 V | 16.0 V |
| **Equalize Voltage** | — | — | — | 14.8 V |
| **Boost Charge Voltage** | 14.4 V | 14.4 V | 14.2 V | 14.6 V |
| **Float Charge Voltage** | — | 13.8 V | 13.8 V | 13.8 V |
| **Boost Return Voltage** | 13.2 V | 13.2 V | 13.2 V | 13.2 V |
| **Low Voltage Disconnect** | 11.1 V | 11.1 V | 11.1 V | 11.1 V |
