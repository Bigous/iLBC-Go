import pathlib, random, math, struct
root=pathlib.Path(__file__).resolve().parent.parent/'src'/'testdata'
root.mkdir(exist_ok=True)
for mode in (20,30):
    rng=random.Random(3951)
    samples=[];loss=[];n=mode*8
    for f in range(320):
        kind=(f//8)%10
        for j in range(n):
            t=f*n+j
            if kind==0:v=0
            elif kind==1:v=32767 if j==n//2 else 0
            elif kind==2:v=int(24000*math.sin(t*2*math.pi*137/8000))
            elif kind==3:v=rng.randint(-32768,32767)
            elif kind==4:v=-32768 if t%2 else 32767
            elif kind==5:v=rng.randint(-2,2)
            elif kind==6:v=16000
            elif kind==7:v=int(13000*math.sin(t*2*math.pi*93/8000)+6000*math.sin(t*2*math.pi*231/8000))
            elif kind==8:v=int(rng.uniform(-1,1)*j/n*28000)
            else:v=int(28000*math.sin(t*t*0.000007))
            samples.append(v)
        loss.append(0 if f%80 in (0,9,12,13,14) or 40<=f%80<=65 else 1)
    (root/f'input{mode}.pcm').write_bytes(struct.pack('<'+'h'*len(samples),*samples))
    (root/f'loss{mode}.bin').write_bytes(bytes(loss))
