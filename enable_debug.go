// +build enableDebug

package main

import "log"

const DEBUG = true

func init() {
	log.Print("*** Go proxy debug enabled")
}
