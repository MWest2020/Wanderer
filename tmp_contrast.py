import sys

def lin(c):
    c = c / 255.0
    if c <= 0.03928:
        return c / 12.92
    return ((c + 0.055) / 1.055) ** 2.4

def lum(hexcolor):
    hexcolor = hexcolor.lstrip('#')
    r, g, b = int(hexcolor[0:2], 16), int(hexcolor[2:4], 16), int(hexcolor[4:6], 16)
    return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)

def contrast(hex1, hex2):
    l1, l2 = lum(hex1), lum(hex2)
    if l1 < l2:
        l1, l2 = l2, l1
    return (l1 + 0.05) / (l2 + 0.05)

def blend(fg_hex, alpha, bg_hex="#ffffff"):
    fg_hex = fg_hex.lstrip('#')
    bg_hex = bg_hex.lstrip('#')
    fr, fg, fb = int(fg_hex[0:2],16), int(fg_hex[2:4],16), int(fg_hex[4:6],16)
    br, bgc, bb = int(bg_hex[0:2],16), int(bg_hex[2:4],16), int(bg_hex[4:6],16)
    r = round(fr*alpha + br*(1-alpha))
    g = round(fg*alpha + bgc*(1-alpha))
    b = round(fb*alpha + bb*(1-alpha))
    return '#%02x%02x%02x' % (r,g,b)

if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "contrast":
        print(round(contrast(sys.argv[2], sys.argv[3]), 3))
    elif cmd == "blend":
        print(blend(sys.argv[2], float(sys.argv[3]), sys.argv[4] if len(sys.argv) > 4 else "#ffffff"))
