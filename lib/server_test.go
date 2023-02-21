package rtrlib

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func GenerateVrps(size uint32, offset uint32) []SendableData {
	vrps := make([]SendableData, size)
	for i := uint32(0); i < size; i++ {
		ipFinal := make([]byte, 4)
		binary.BigEndian.PutUint32(ipFinal, i+offset)
		vrps[i] = &VRP{
			Prefix: net.IPNet{
				IP:   net.IP(append([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, ipFinal...)),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    64496,
		}
	}
	return vrps
}

func BaseBench(base int, multiplier int) {
	benchSize1 := base * multiplier
	newVrps := GenerateVrps(uint32(benchSize1), uint32(0))
	benchSize2 := base
	prevVrps := GenerateVrps(uint32(benchSize2), uint32(benchSize1-benchSize2/2))
	ComputeDiff(newVrps, prevVrps)
}

func BenchmarkComputeDiff1000x10(b *testing.B) {
	BaseBench(1000, 10)
}

func BenchmarkComputeDiff10000x10(b *testing.B) {
	BaseBench(10000, 10)
}

func BenchmarkComputeDiff100000x1(b *testing.B) {
	BaseBench(100000, 1)
}

func TestComputeDiff(t *testing.T) {
	newVrps := []VRP{
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x3}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65003,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65002,
		},
	}
	prevVrps := []VRP{
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65001,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65002,
		},
	}

	newVrpsSD, prevVrpsAsSD := make([]SendableData, 0), make([]SendableData, 0)
	for _, v := range newVrps {
		newVrpsSD = append(newVrpsSD, v.Copy())
	}
	for _, v := range prevVrps {
		prevVrpsAsSD = append(prevVrpsAsSD, v.Copy())
	}

	added, removed, unchanged := ComputeDiff(newVrpsSD, prevVrpsAsSD)
	assert.Len(t, added, 1)
	assert.Len(t, removed, 1)
	assert.Len(t, unchanged, 1)
	assert.Equal(t, added[0].(*VRP).ASN, uint32(65003))
	assert.Equal(t, removed[0].(*VRP).ASN, uint32(65001))
	assert.Equal(t, unchanged[0].(*VRP).ASN, uint32(65002))
}

func TestApplyDiff(t *testing.T) {
	diff := []VRP{
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x3}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65003,
			Flags:  FLAG_ADDED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65002,
			Flags:  FLAG_REMOVED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x4}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65004,
			Flags:  FLAG_REMOVED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x6}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65006,
			Flags:  FLAG_REMOVED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65007,
			Flags:  FLAG_ADDED,
		},
	}
	prevVrps := []VRP{
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65001,
			Flags:  FLAG_ADDED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65002,
			Flags:  FLAG_ADDED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x5}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65005,
			Flags:  FLAG_REMOVED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x6}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65006,
			Flags:  FLAG_REMOVED,
		},
		{
			Prefix: net.IPNet{
				IP:   net.IP([]byte{0xfd, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7}),
				Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			},
			MaxLen: 128,
			ASN:    65007,
			Flags:  FLAG_REMOVED,
		},
	}
	diffSD, prevVrpsAsSD := make([]SendableData, 0), make([]SendableData, 0)
	for _, v := range diff {
		diffSD = append(diffSD, &v)
	}
	for _, v := range prevVrps {
		prevVrpsAsSD = append(prevVrpsAsSD, &v)
	}

	vrps := ApplyDiff(diffSD, prevVrpsAsSD)

	assert.Len(t, vrps, 6)
	assert.Equal(t, vrps[0].(*VRP).ASN, uint32(65001))
	assert.Equal(t, vrps[0].(*VRP).GetFlag(), uint8(FLAG_ADDED))
	assert.Equal(t, vrps[1].(*VRP).ASN, uint32(65005))
	assert.Equal(t, vrps[1].(*VRP).GetFlag(), uint8(FLAG_REMOVED))
	assert.Equal(t, vrps[2].(*VRP).ASN, uint32(65003))
	assert.Equal(t, vrps[2].(*VRP).GetFlag(), uint8(FLAG_ADDED))
	assert.Equal(t, vrps[3].(*VRP).ASN, uint32(65004))
	assert.Equal(t, vrps[3].(*VRP).GetFlag(), uint8(FLAG_REMOVED))
	assert.Equal(t, vrps[4].(*VRP).ASN, uint32(65006))
	assert.Equal(t, vrps[4].(*VRP).GetFlag(), uint8(FLAG_REMOVED))
	assert.Equal(t, vrps[5].(*VRP).ASN, uint32(65007))
	assert.Equal(t, vrps[5].(*VRP).GetFlag(), uint8(FLAG_ADDED))
}
