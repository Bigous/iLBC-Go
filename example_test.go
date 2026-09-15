package ilbc_test

import (
	"fmt"

	"github.com/Bigous/iLBC-Go"
)

func ExampleEncoder_Encode() {
	encoder, err := ilbc.NewEncoder(ilbc.Mode20)
	if err != nil {
		panic(err)
	}
	pcm := make([]int16, encoder.FrameSamples())
	packet := make([]byte, encoder.FrameBytes())
	if err = encoder.Encode(packet, pcm); err != nil {
		panic(err)
	}
	fmt.Println(len(pcm), len(packet))
	// Output: 160 38
}
