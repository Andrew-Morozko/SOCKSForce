//go:build enableDebug

package main

import "log"

const DEBUG = true

func init() {
	log.Print("SOCKSForce debug enabled")
}
