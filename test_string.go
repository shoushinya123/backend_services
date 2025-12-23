package main

import (
	"fmt"
	"strings"
)

func main() {
	url := "postgresql://user:password@localhost:5432/db"
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Contains 'password=': %v\n", strings.Contains(url, "password="))
	fmt.Printf("Contains 'pass=': %v\n", strings.Contains(url, "pass="))
}
