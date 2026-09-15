"""Rebuild golden vectors with the independently compiled RFC C reference.
Requires Python 3 and either gcc/clang on PATH or an MSVC developer shell.
The Go package and its normal tests do not require Python or a C compiler.
"""
import pathlib, re, tempfile, shutil, subprocess, hashlib
ROOT=pathlib.Path(__file__).resolve().parent.parent

def correct(filename,s):
    if filename=='iLBC_decode.c':
        s=s.replace('int lag, ilag;', 'int lag, ilag, corrLen, corrStart;')
        s=s.replace('/* Find last lag */','/* Find last lag within the initialized frame. */\n        corrLen = iLBCdec_inst->blockl == 20*8 ? 40 : 80;\n        corrStart = iLBCdec_inst->blockl - corrLen;')
        s=s.replace('BLOCKL_MAX-ENH_BLOCKL','corrStart')
        s=s.replace('lag], ENH_BLOCKL)', 'lag], corrLen)')
        s=s.replace('ilag],\n                ENH_BLOCKL)', 'ilag],\n                corrLen)')
    if filename=='doCPLC.c':
        s=s.replace('iLBCdec_inst->consPLICount += 1;', 'if (iLBCdec_inst->consPLICount < 9) iLBCdec_inst->consPLICount += 1;')
        a=s.index('        use_gain=1.0;');b=s.index('        /* mix noise',a)
        s=s[:a]+'''        use_gain=1.0;
        if (iLBCdec_inst->consPLICount*iLBCdec_inst->blockl>4*320)
            use_gain=(float)0.0;
        else if (iLBCdec_inst->consPLICount*iLBCdec_inst->blockl>3*320)
            use_gain=(float)0.5;
        else if (iLBCdec_inst->consPLICount*iLBCdec_inst->blockl>2*320)
            use_gain=(float)0.7;
        else if (iLBCdec_inst->consPLICount*iLBCdec_inst->blockl>320)
            use_gain=(float)0.9;

'''+s[b:]
        s=s.replace('PLCresidual[i] = randvec[i];','PLCresidual[i] = use_gain*randvec[i];')
    if filename=='enhancer.c':
        s=s.replace('int lag=0, ilag, i, ioffset;', 'int lag=0, ilag, i, ioffset, lastLag;')
        s=s.replace('inlag=(int)enh_period', 'inlag=(int)enh_period')
        s,n=re.subn(r'(    if\s*\(iLBCdec_inst->prev_enh_pl\s*==\s*1\))',r'    lastLag=lag*2;\n\1',s)
        assert n==1
        s=s.replace('return (lag*2);','return lastLag;')
    return s

def main():
    data=ROOT/'src'/'testdata';reference=data/'reference'
    with tempfile.TemporaryDirectory(prefix='ilbc-reference-') as directory:
        build=pathlib.Path(directory)
        for src in (reference/'original').glob('*.[ch]'):
            (build/src.name).write_text(correct(src.name,src.read_text()))
        shutil.copyfile(reference/'oracle.c',build/'oracle.c')
        sources=[str(p) for p in build.glob('*.c') if p.name!='iLBC_test.c']
        compiler=shutil.which('gcc') or shutil.which('clang') or shutil.which('cl')
        if not compiler:raise SystemExit('Install a C compiler or use an MSVC developer shell.')
        exe=build/('oracle.exe' if __import__('os').name=='nt' else 'oracle')
        if pathlib.Path(compiler).stem.lower()=='cl':
            args=[compiler,'/nologo','/O2','/fp:precise','/D_CRT_SECURE_NO_WARNINGS','/Fe:'+str(exe)]+sources
        else:
            args=[compiler,'-O2','-ffp-contract=off','-fno-fast-math']+sources+['-lm','-o',str(exe)]
        subprocess.run(args,cwd=build,check=True)
        for mode in (20,30):
            for enhance in (0,1):
                flag='true' if enhance else 'false'
                subprocess.run([str(exe),str(mode),str(enhance),str(data/f'input{mode}.pcm'),str(data/f'encoded{mode}.bin'),str(data/f'decoded{mode}_{flag}.pcm'),str(data/f'loss{mode}.bin')],check=True)
    for f in sorted(data.glob('*')):
        if f.is_file() and f.suffix in ('.bin','.pcm'):print(hashlib.sha256(f.read_bytes()).hexdigest(),f.name)

if __name__=='__main__':main()
