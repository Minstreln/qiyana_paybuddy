package services

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateReference(prefix string) string {
	suffix := fmt.Sprintf("%04d", rand.Intn(10000))
	return fmt.Sprintf("%s%s-%s", prefix, time.Now().Format("20060102150405"), suffix)
}
