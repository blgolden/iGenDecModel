// genAnimal
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
package animal

import (
	"fmt"
	//"os"
	//"math"
	"strconv"

	"github.com/blgolden/iGenDecModel/iGenDec/logger"

	"gonum.org/v1/gonum/mat"
	// "gonum.org/v1/gonum/stat/distuv"
)

var Traits []string
var Components []string

// For the culling policy it is the length of ageDist in master.hjson
var MaxCowAge int

// Foundation bulls
func MakeFoundationBulls(
	gvCholesky mat.Cholesky,
	rvCholesky mat.Cholesky,
	param map[string]interface{}) {

	for _, h := range Herds {
		thisHerd := &h

		nBulls := int(param["nFoundationBulls"].(float64))
		if *logger.OutputMode == "verbose" {
			fmt.Println("Foundation bulls for: ", thisHerd.HerdName)
			fmt.Println("Number of foundation bulls:", nBulls)
		}
		idCounter := AnimalId(len(Records))
		for i := 0; i < nBulls; i++ {

			idCounter++

			var a Animal
			a.Id = idCounter
			a.Sex = Bull
			a.BirthDate = 0
			a.Active = true
			a.HerdName = h.HerdName

			GenFoundation(&a, gvCholesky, rvCholesky) // Make this Bull

			GenBullBatteryBreedComposition(&a)

			for j, m := range BullMerit {
				bv := a.BreedingValue.AtVec(j)
				bv += m
				a.BreedingValue.SetVec(j, bv)
			}

			Records = append(Records, a) // Create a record

		}
		h.Bulls = ActiveBulls(&h)
	}
}

// Make a herd of foundation cows
func MakeFoundationCowHerd(
	gvCholesky mat.Cholesky,
	rvCholesky mat.Cholesky,
	param map[string]interface{}) (cowHerdSize int) {

	if *logger.OutputMode == "verbose" {
		if len(Herds) > 1 {
			fmt.Printf("Making %d foundation herds\n", len(Herds))
		} else {
			fmt.Printf("Making 1 foundation herd\n")
		}
	}

	// Initial cow herd age distribution from the hjson file
	array := param["ageDist"].([]interface{})

	var k int
	for _, h := range Herds {
		cowHerdSize = h.NumberCows

		if *logger.OutputMode == "verbose" {
			fmt.Println("Foundation herd size:", cowHerdSize)
		}
		var v []float64 // The proportion at each age of cow in years
		var totProp float64
		for i := range array {
			f, _ := strconv.ParseFloat(array[i].(string), 64)
			v = append(v, f)
			totProp += v[i]
		}
		if totProp > 1.001 || totProp < .999 {
			//fmt.Fprintf(os.Stderr, "Cow herd proportion does not add to 1.0: %g\n", totProp)
			logger.LogWriterFatal("Cow herd proportion does not add to 1.0")
		}

		// loop through each cow age and make that proportion
		// of foundation animals
		var idCounter AnimalId
		if k > 0 {
			idCounter = AnimalId(len(Records))
		} else {
			idCounter = 0
		}
		k++

		for i := 0; i < len(v); i++ {
			thisAge := int(float64(cowHerdSize) * v[i])
			//fmt.Println("LOC 3", i, thisAge, (len(v)-i)*-1-1)
			for j := 0; j < thisAge; j++ {

				idCounter++

				var a Animal
				a.Id = idCounter
				a.Sex = Cow
				a.BirthDate = Date((len(v)-i)*-1*365 + Rng.Intn(int(h.BreedingSeasonLen)) + GestationLengthError())
				a.YearBorn = (len(v) - i) * -1
				a.Active = true
				a.HerdName = h.HerdName

				GenFoundation(&a, gvCholesky, rvCholesky) // Make this cow

				GenFoundationBreedComposition(&a)

				Records = append(Records, a) // Create a record

				//herd[h].Cows = append(herd[h].Cows, &Records[len(Records)-1]) // List of active cows
				//fmt.Printf("LOC 2 %p\n", &Records[len(Records)-1])
			}
		}
		h.Cows = ActiveCows(&h)
	}

	// for culling policy
	MaxCowAge = len(array) + 1 // +1 because it starts at 2yoa

	return
}
func GestationLength() Date {
	return 283
}

// Make the foundation heifers
func MakeFoundationHeifers(
	gvCholesky mat.Cholesky,
	rvCholesky mat.Cholesky,
	param map[string]interface{}) (cowHerdSize int) {

	if *logger.OutputMode == "verbose" {
		if len(Herds) > 1 {
			fmt.Printf("Making foundation heifers for %d herds\n", len(Herds))
		} else {
			fmt.Printf("Making foundation heifers for 1 herd\n")
		}
	}

	for _, h := range Herds {

		cowHerdSize = h.NumberCows

		nHeifers := int(.2 * float32(cowHerdSize))

		idCounter := AnimalId(len(Records))

		for i := 0; i < nHeifers; i++ {

			idCounter++

			var a Animal
			a.Id = idCounter
			a.Sex = Heifer
			a.BirthDate = h.StartBreeding + Date(Rng.Intn(int(h.BreedingSeasonLen))) + GestationLength() - 365
			a.YearBorn = 0
			a.Active = false
			a.HerdName = h.HerdName

			GenFoundation(&a, gvCholesky, rvCholesky) // Make this heifer

			GenFoundationBreedComposition(&a)

			Records = append(Records, a) // Create a record
		}

	}

	return
}
