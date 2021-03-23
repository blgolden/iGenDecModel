// logger
/*
Copyright 2021 Bruce Golden and Matt Spangler

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package logger

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

var OutputMode *string // verbose or model
var User *string       // name of this user running this run
var Seed *int64        // Random number generator seed

func LogWriter(message string) {
	f, err := os.OpenFile("log.iGenDec."+strconv.FormatInt(*Seed, 10), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger := log.New(f, "iGenDec ", log.LstdFlags)
	logger.Println(message)
	return
}
func LogWriterFatal(message string) {
	f, err := os.OpenFile("log.iGenDec."+strconv.FormatInt(*Seed, 10), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger := log.New(f, "iGenDec ", log.LstdFlags)
	logger.Println(message)

	if *OutputMode == "verbose" {
		fmt.Println(message)
	}
	os.Exit(1)
}
