package main

import (
	"fmt"

	"github.com/google/nomos/pkg/util/repo"
)

func main() {
	fmt.Printf("%s\n", repo.CurrentVersion)
}
