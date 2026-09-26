import math, itertools, sys
def hx(h): h=h.lstrip('#'); return [int(h[i:i+2],16)/255 for i in (0,2,4)]
def lin(c): return c/12.92 if c<=0.04045 else ((c+0.055)/1.055)**2.4
def delin(c): c=max(0,min(1,c)); return 12.92*c if c<=0.0031308 else 1.055*c**(1/2.4)-0.055
def lum(rgb): r,g,b=[lin(c) for c in rgb]; return 0.2126*r+0.7152*g+0.0722*b
def contrast(a,b): la,lb=sorted([lum(a),lum(b)],reverse=True); return (la+0.05)/(lb+0.05)
M={ # Machado 2009, severity 1.0, op lineaire RGB
 'protan':[[0.152286,1.052583,-0.204868],[0.114503,0.786281,0.099216],[-0.003882,-0.048116,1.051998]],
 'deutan':[[0.367322,0.860646,-0.227968],[0.280085,0.672501,0.047413],[-0.011820,0.042940,0.968881]],
 'tritan':[[1.255528,-0.076749,-0.178779],[-0.078411,0.930809,0.147602],[0.004733,0.691367,0.303900]]}
def sim(rgb,k):
    if k=='normaal': return rgb
    l=[lin(c) for c in rgb]; m=M[k]
    return [delin(sum(m[i][j]*l[j] for j in range(3))) for i in range(3)]
def lab(rgb):
    r,g,b=[lin(c) for c in rgb]
    x=(0.4124*r+0.3576*g+0.1805*b)/0.95047; y=0.2126*r+0.7152*g+0.0722*b; z=(0.0193*r+0.1192*g+0.9505*b)/1.08883
    f=lambda t: t**(1/3) if t>0.008856 else 7.787*t+16/116
    return 116*f(y)-16, 500*(f(x)-f(y)), 200*(f(y)-f(z))
def de2000(a,b):
    L1,a1,b1=a; L2,a2,b2=b
    C1=math.hypot(a1,b1); C2=math.hypot(a2,b2); Cb=(C1+C2)/2
    G=0.5*(1-math.sqrt(Cb**7/(Cb**7+25**7)))
    a1p,a2p=(1+G)*a1,(1+G)*a2; C1p,C2p=math.hypot(a1p,b1),math.hypot(a2p,b2)
    h1=math.degrees(math.atan2(b1,a1p))%360; h2=math.degrees(math.atan2(b2,a2p))%360
    dL=L2-L1; dC=C2p-C1p
    dh=0 if C1p*C2p==0 else (h2-h1 if abs(h2-h1)<=180 else h2-h1-360 if h2>h1 else h2-h1+360)
    dH=2*math.sqrt(C1p*C2p)*math.sin(math.radians(dh/2))
    Lb=(L1+L2)/2; Cbp=(C1p+C2p)/2
    hb=(h1+h2)/2 if abs(h1-h2)<=180 else (h1+h2+360)/2 if h1+h2<360 else (h1+h2-360)/2
    if C1p*C2p==0: hb=h1+h2
    T=1-0.17*math.cos(math.radians(hb-30))+0.24*math.cos(math.radians(2*hb))+0.32*math.cos(math.radians(3*hb+6))-0.2*math.cos(math.radians(4*hb-63))
    SL=1+0.015*(Lb-50)**2/math.sqrt(20+(Lb-50)**2); SC=1+0.045*Cbp; SH=1+0.015*Cbp*T
    RT=-2*math.sqrt(Cbp**7/(Cbp**7+25**7))*math.sin(math.radians(60*math.exp(-((hb-275)/25)**2)))
    return math.sqrt((dL/SL)**2+(dC/SC)**2+(dH/SH)**2+RT*(dC/SC)*(dH/SH))
def rapport(naam, pal, bg='#ffffff'):
    print(f"== {naam}  (achtergrond {bg})")
    print("  contrast met achtergrond:", {k: round(contrast(hx(v),hx(bg)),1) for k,v in pal.items()})
    for vis in ('normaal','protan','deutan','tritan'):
        paren=[(round(de2000(lab(sim(hx(pal[a]),vis)),lab(sim(hx(pal[b]),vis))),1),a[:5]+'/'+b[:5]) for a,b in itertools.combinations(pal,2)]
        print(f"  {vis:8} kleinste ΔE {min(paren)}")
nu={'soeverein':'#146c43','voldoende':'#45700d','afhankelijk':'#b02a37','onbekend':'#495057'}
rapport("nu", nu)
kand={'soeverein':'#0072b2','voldoende':'#56b4e9','afhankelijk':'#d55e00','onbekend':'#8c8c8c'}
rapport("kandidaat Okabe-Ito", kand)

def score(pal, bg):
    c=min(contrast(hx(v),hx(bg)) for v in pal.values())
    d=min(de2000(lab(sim(hx(pal[a]),vis)),lab(sim(hx(pal[b]),vis))) for vis in ('normaal','protan','deutan','tritan') for a,b in itertools.combinations(pal,2))
    return c,d
if len(sys.argv)>1:
    import random; random.seed(1)
    # soeverein: blauwtinten, voldoende: lichter blauw/cyaan, afhankelijk: oranje/vermiljoen, onbekend: grijs
    blauw=['#0072b2','#005a9c','#1f5fa8','#004c8c','#0b63a8']
    licht=['#3d8fd1','#2b8cc4','#4a90c8','#3a86c8','#2e8bcc','#1f88c9','#3b8dbd']
    oranje=['#c85200','#d55e00','#b84a00','#c0560a','#cc5500','#a84300']
    grijs=['#6b6b6b','#707070','#767676','#7a7a7a','#666666']
    best=None
    for s in blauw:
      for v in licht:
        for a in oranje:
          for o in grijs:
            p={'soeverein':s,'voldoende':v,'afhankelijk':a,'onbekend':o}
            c,d=score(p,'#ffffff'); c2,d2=score(p,'#1a1a1a') if False else (0,0)
            if c>=3.0 and (best is None or d>best[0]): best=(d,c,p)
    print("beste:", round(best[0],1), "min contrast", round(best[1],2), best[2])
    rapport("beste", best[2])
