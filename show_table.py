import re
import sys


CONTROL_CHARS = {
      0: "\u2400",
      1: "\u2401",
      2: "\u2402",
      3: "\u2403",
      4: "\u2404",
      5: "\u2405",
      6: "\u2406",
      7: "\u2407",
      8: "\u2408",
      9: "\u2409",
     10: "\u240a",
     11: "\u240b",
     12: "\u240c",
     13: "\u240d",
     14: "\u240e",
     15: "\u240f",
     16: "\u2410",
     17: "\u2411",
     18: "\u2412",
     19: "\u2413",
     20: "\u2414",
     21: "\u2415",
     22: "\u2416",
     23: "\u2417",
     24: "\u2418",
     25: "\u2419",
     26: "\u241a",
     27: "\u241b",
     28: "\u241c",
     29: "\u241d",
     30: "\u241e",
     31: "\u241f",
     32: "\u2420",
    127: "\u2421",
    128: " ",
    129: " ",
    130: " ",
    131: " ",
    132: " ",
    133: " ",
    134: " ",
    135: " ",
    136: " ",
    137: " ",
    138: " ",
    139: " ",
    140: " ",
    141: " ",
    142: " ",
    143: " ",
    144: " ",
    145: " ",
    146: " ",
    147: " ",
    148: " ",
    149: " ",
    150: " ",
    151: " ",
    152: " ",
    153: " ",
    154: " ",
    155: " ",
    156: " ",
    157: " ",
    158: " ",
    159: " ",
}


def render(f):
    s = 0

    ma = {}
    code_set_name = None

    for l in f:
        l = l.strip()
        if s == 0:
            m = re.match(r"^<code_set_name>\s*\"([^\"]*)\"", l)
            if m is not None:
                code_set_name = m.group(1)
            elif l == "CHARMAP":
                s = 1
        elif s == 1:
            m = re.match(r"<U([0-9a-fA-F]+)>\s+\\x([0-9a-fA-F]+)\s*|\s*[0-9]", l)
            if m is None:
                if l != "END CHARMAP":
                    raise Exception(l)
                break
            ma[int(m.group(2), 16)] = int(m.group(1), 16)

    if code_set_name is not None:
        print(f"### {code_set_name}")
        print()

    print("".join(["|        "] + [f"| X'{x*16:02X}' " for x in range(16)] + ["|"]))
    print("".join(["| ------ "] + [f"| ----- " for x in range(16)] + ["|"]))

    for y in range(16):
        buf = [f"| +X'{y:02X}'"]
        for x in range(16):
            cc = x * 16 + y
            buf.append(" |  `")
            v = ma.get(cc)
            c = (CONTROL_CHARS.get(v) or chr(v)) if v is not None else " "
            if c in ("`", "|"):
                buf.append("\\")
            buf.append(c)
            buf.append("` ")
        buf.append(" |")
        print("".join(buf))

    print()
    print("| EBCDIC code | Unicode codepoint | character |")
    print("| ----------- | ----------------- | --------- |")
    for k, v in sorted(ma.items(), key=lambda pair: pair[0]):
        c = CONTROL_CHARS.get(v) or chr(v)
        s = "\\" if c in ("`", "|") else ""
        print(f"| X'{k:02X}'       | U+{v:04X}            | `{s}{c}`       |")
    print()


for fn in sys.argv[1:]:
    with open(fn) as f:
        render(f)
        print()
