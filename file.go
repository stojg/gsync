package main

import (
	"math/rand"
	"time"
)

var fileSizes = [4]int64{
	128 * 1000,
	512 * 1000,
	1000 * 1000,
	2 * 1000 * 1000,
}

var fileRatios = [4]float64{
	0.7,
	0.2,
	0.05,
	0.05,
}

type fHolder struct {
	fileSizeIndex int
	size          int64
}

func fileOrder(numFiles int) []*fHolder {
	a := make([]*fHolder, 0)

	for i := 0; i < len(fileSizes); i++ {

		filesInThisRange := int(fileRatios[i] * float64(numFiles))

		for j := 0; j < filesInThisRange; j++ {
			a = append(a, &fHolder{
				fileSizeIndex: i,
				size:          fileSizes[i],
			})
		}
	}
	rand.Seed(time.Now().Unix())
	result := make([]*fHolder, len(a))
	perm := rand.Perm(len(a))
	for i, v := range perm {
		result[v] = a[i]
	}
	return result
}

type randByteMaker struct {
	src rand.Source
}

func (r *randByteMaker) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(r.src.Int63() & 0xff) // mask to only the first 255 byte values
	}
	return len(p), nil
}
