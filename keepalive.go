package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"time"
)

// KeepAliveToken represents a unique token per calendar month.
type KeepAliveToken struct{}

func (g KeepAliveToken) MarshalJSON() ([]byte, error) { return json.Marshal(g.String()) }

func (KeepAliveToken) String() string {
	// sha1 encoded month of year
	return fmt.Sprintf("%x", sha1.Sum([]byte{byte(time.Now().Month())}))
}
