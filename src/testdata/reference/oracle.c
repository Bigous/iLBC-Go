#include <stdio.h>
#include <stdlib.h>
#include "iLBC_define.h"
#include "iLBC_encode.h"
#include "iLBC_decode.h"
int main(int argc, char **argv) {
    iLBC_Enc_Inst_t enc;
    iLBC_Dec_Inst_t dec;
    FILE *input, *bits, *output, *loss;
    short pcm[240], decoded[240];
    float x[240], y[240];
    unsigned char packet[50];
    int mode, enhance, i, received;
    if (argc != 7) return 2;
    mode=atoi(argv[1]); enhance=atoi(argv[2]);
    input=fopen(argv[3],"rb"); bits=fopen(argv[4],"wb");
    output=fopen(argv[5],"wb"); loss=fopen(argv[6],"rb");
    if (!input || !bits || !output || !loss) return 3;
    initEncode(&enc,mode); initDecode(&dec,mode,enhance);
    while (fread(pcm,sizeof(short),enc.blockl,input)==enc.blockl) {
        for (i=0;i<enc.blockl;i++) x[i]=(float)pcm[i];
        iLBC_encode(packet,x,&enc);
        fwrite(packet,1,enc.no_of_bytes,bits);
        received=fgetc(loss); if(received==EOF) return 4;
        iLBC_decode(y,packet,&dec,received);
        for(i=0;i<enc.blockl;i++) {
            if (y[i]<-32768) y[i]=-32768;
            else if (y[i]>32767) y[i]=32767;
            decoded[i]=(short)y[i];
        }
        fwrite(decoded,sizeof(short),enc.blockl,output);
    }
    fclose(input); fclose(bits); fclose(output); fclose(loss);
    return 0;
}
