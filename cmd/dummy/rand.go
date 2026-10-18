package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
)

// randIntn returns a uniform random int in [0, n) using crypto/rand.
func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return int(v.Int64())
}

// randFloat64 returns a uniform random float in [0, 1) using crypto/rand.
func randFloat64() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return float64(binary.LittleEndian.Uint64(b[:])>>11) / (1 << 53)
}

// randShuffle shuffles n elements using the Fisher-Yates algorithm.
func randShuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		swap(i, randIntn(i+1))
	}
}
