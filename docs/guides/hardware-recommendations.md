# Hardware recommendations

What to buy for each cashier till in Indonesia: a barcode scanner, a thermal
receipt printer, and a cash drawer. Every model listed is sold on the usual
Indonesian marketplaces and every price is an **indicative** figure taken from
listings in October 2026, excluding shipping and promo vouchers — check the
current listing before ordering.

This is a purchasing guide, not a compatibility claim. The software side of
each device is described at the end so you can check the one thing that
actually matters about a listing: the port it has.

## The three devices, and the one decision that couples them

| Device | Connects to | Drives |
| ------ | ----------- | ------ |
| Barcode scanner | the till PC, USB | nothing — it is a keyboard |
| Thermal printer | the till PC, USB **or** Ethernet | printing, and the drawer |
| Cash drawer | **the printer**, RJ11 | nothing — it is a latch |

The drawer is the reason the printer choice comes first. Cash drawers in
Indonesia are RJ11 and *printer-driven*: the drawer's own cable goes into the
printer's drawer (kick-out) port and the printer supplies the pulse that throws
the latch. There is no USB drawer, and no drawer cable goes anywhere near the
PC. So a printer without a drawer port cannot open a drawer at all, no matter
what else it can do — **check the exact SKU for a drawer port before you buy,
printer and drawer together.**

## Barcode scanner

Any USB HID scanner works — "keyboard wedge" is the whole integration. The POS
has no scanner driver; the scanner types the code into the product search box
and the Enter at the end of the code adds the top match.

| Model | Type | Indicative price |
| ----- | ---- | ---------------- |
| Blueprint BP-ECO X1D | 1D laser, USB, 350 scans/s, Linux/Windows/Mac | Rp175.000 – 185.000 |
| Xantri U3-C | 1D, USB | Rp258.000 |
| Kanalogic LS-31S | 1D laser, USB | Rp350.000 |
| Deli ES212 | handheld, USB — confirm 1D/2D on the listing | Rp411.730 |
| iData J16 | 2D imager — reads QR from a phone screen | Rp2.300.000 |
| Blueprint BP-Lite W2D | 2D, USB + Bluetooth + wireless, battery | Rp530.000 |
| Blueprint OM200 | 2D CMOS, auto-sense — reads QR, red and curved barcodes | Rp735.000 |

**1D is enough for a normal retail shop.** Indonesian product barcodes are
EAN-13/UPC, which a 1D laser reads fine, and the difference between these
models is build quality rather than capability. Pay more only if one of these
applies:

- You scan **QR codes from a customer's phone** (member QR, payment QR) — a
  laser cannot read a screen. Take a 2D imager: the iData J16, or the much
  cheaper Blueprint OM200 if you also print red or curved labels.
- You print your own **labels in red ink**. A laser reads black on white only;
  the BP-ECO X1D's own listing says so explicitly. A 2D CMOS scanner reads
  colour.

**Two things to configure or check on the listing:**

1. **The suffix must be Enter (CR), not TAB.** That is the factory default on
   almost every wedge scanner, and it is what makes a scan count as a search
   submit. A scanner set to TAB moves focus instead of adding the item.
2. **Prefer wired over wireless.** A till scanner sits next to the PC and is
   never moved. A wireless or Bluetooth one adds a battery to charge and a
   pairing that can drop mid-sale, for no benefit here.

## Thermal receipt printer

**58 mm, ESC/POS, and a drawer port.** The agent renders a 58 mm receipt —
it centres the headers and pads each line by character count — so a 58 mm
printer is the size everything was written for. An 80 mm printer will print it
too, since alignment is a printer-side command, but you are paying for paper
you will only half use.

| Model | Interfaces | Drawer port | Indicative price |
| ----- | ---------- | ----------- | ---------------- |
| Xprinter XP-58IIZ | USB + Bluetooth | RJ11 | Rp281.250 |
| Xprinter XP-58IIH | USB (Bluetooth variant exists) | RJ11, 6-pin, 12 V/1 A | ≈Rp530.000 for the USB+BT bundle |
| Xprinter XP-58IIB | USB + Bluetooth | RJ11 | ask your reseller |
| Xprinter XP-58IIIK | USB + **100 Mbps Ethernet** | yes | ask your reseller |
| Xantri BT-58D | USB + Bluetooth | RJ11 | ask your reseller |
| Iware C58BT Pro | USB + Bluetooth | RJ11, ESC/POS, 90 mm/s | Rp310.000 |
| MUNBYN P068 | 80 mm — see the size note above | yes | ask your reseller |

All of the 58 mm models above are **tear-bar** machines: no cutter, you tear
the receipt along the strip. That is fine — the agent always ends a receipt
with a cut command, and a printer with no cutter is simply unaffected.

### Ethernet if any till runs Windows

This is the decision worth spending on. The agent talks to a printer over
`file`, `tcp` or `serial`:

- `serial` is configured by a device node and group membership on Linux
  (`/dev/ttyUSB0`, `dialout`). The agent exposes no baud or parity setting at
  all, so the line rate is whatever the driver defaults to, and that path is
  only exercised on Linux.
- `tcp` takes one `PRINT_TCP_ADDR=192.168.x.x:9100` and works identically on
  every operating system, with no device permissions and no cable conventions.

A mixed Windows/Linux fleet — the normal case once the shop has grown a bit —
therefore points at an **Ethernet printer**. It also means one printer can be
shared between two tills if you ever need that, and the agent keeps working if
the PC is replaced. Bluetooth is not a substitute: the agent never talks
Bluetooth, so a "Bluetooth printer" is only useful to a phone, not to this POS.

## Cash drawer

Any RJ11 printer-driven drawer will work, because the pulse comes from the
printer and the drawer is only a latch and a switch. Size, metal and the
number of coin cups are the things you are actually paying for.

| Model | Size | Notes | Indicative price |
| ----- | ---- | ----- | ---------------- |
| eppos 37×33 cm | compact | fits a small counter | Rp265.000 |
| Iware CD330 | compact | | Rp290.000 |
| Iware UL-400 | large | | Rp430.000 – 450.000 |
| Kassen K-410 | 410 × 415 × 100 mm | full metal, 5 note / 8 coin slots, 12–24 V, 2 keys, RJ11 | Rp475.000 – 600.000 |
| Xantri CD X405 / X335 | varies | | ask your reseller |

Buy the drawer that matches the counter and the number of denominations you
handle; the electronics are the same. The Kassen K-410 is the usual
"buy it once" pick — full metal, an RJ11 lead in the box, and a key lock with
the three usual positions (standby / locked / open), so it can be locked
overnight or opened by key when the printer is switched off.

## What each device does in the software

Nothing here needs a driver beyond what the OS already provides.

**Scanner.** It types into the focused product search box on the POS screen.
`F2` refocuses that box. The suffix (Enter) adds the top match, and after an
add the box is left focused and selected so the next code *replaces* the last
one rather than being appended to it — scan two items back to back and both
land. There is deliberately no "global scanner buffer": a code typed while the
search box is not focused goes nowhere, because a stray keystroke adding a
product would be worse than a missed scan.

**Printer.** One print agent per till, started with the transport the printer
needs. See [print agent production install](print-agent-production.md) for the
service, the origin allowlist and the configuration.

**Drawer.** The drawer opens automatically after a sale paid in cash (including
a split with a cash part), and never after a QRIS or card sale. During setup,
and after moving the printer, test it from the print settings panel in the
cart footer: ⚙ next to the Preview/Silent toggle, then **Open Drawer**. That
button reports whether the pulse was sent; if it says the agent is unreachable,
fix that first. `PRINT_DRAWER_PIN` and `PRINT_DRAWER_PULSE_MS` are the two
settings that cover a drawer that will not fire or fires the wrong way — both
are documented in the production guide.

## Per-till shopping list

For a shop whose tills run a mix of Windows and Linux, buying the Ethernet
printer:

| Item | Model | Qty per till |
| ---- | ----- | ------------ |
| Thermal printer | Xprinter XP-58IIIK (Ethernet) | 1 |
| Cash drawer | Kassen K-410, or eppos/Iware for a tight counter | 1 |
| Barcode scanner | Blueprint BP-ECO X1D | 1 |
| RJ11 lead | comes with the drawer | 1 |
| Thermal roll 58 mm | any brand | a box |

If **every** till is Linux and stays that way, an XP-58IIZ over USB saves
money on the printer and is the one line to swap.

Roughly **Rp750.000 – 1.300.000 per till** at these prices — the Ethernet
printer is above the top of that range and is the part worth not economising
on — plus the PC. Before ordering, walk the [checklist](#before-you-order)
below against the exact listings.

## Before you order

- [ ] The printer's spec sheet says **ESC/POS** (every model above does).
- [ ] The printer's spec sheet has a **drawer / kick-out port, RJ11** — not
      just a USB port.
- [ ] The drawer listing says **RJ11 / printer-driven**, and comes with the
      lead.
- [ ] If any till runs Windows: the printer has **Ethernet**, and the listing
      says so for the exact SKU (the same model letter ships USB-only and
      USB+LAN).
- [ ] The scanner's suffix is **Enter/CR** (the default), and it is a wired
      USB model.
- [ ] **1D** is all you need unless you scan phone screens or print red
      labels.
