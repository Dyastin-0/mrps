package hash

import (
	"hash/fnv"
)

func FNV(ip string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(ip))
	return h.Sum32()
}
