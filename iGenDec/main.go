// iGenDec project main.go
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

package main

import (
	"fmt"
	//"os"
	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	"github.com/blgolden/iGenDecModel/iGenDec/ecoIndex"
	"github.com/blgolden/iGenDecModel/iGenDec/logger"
	"github.com/blgolden/iGenDecModel/iGenDec/varStuff"
)

var version = "beta0.0.5a"

var burninMarker int // Length of the Burnin Records in Records[]

func printTables() {
	if *logger.OutputMode == "verbose" || *logger.OutputMode == "conception" {

		var cowRate, heiferRate float64

		// Print the breeding summary table
		fmt.Printf("               ________Cows___________________________   | _____________Heifers___________________\n")
		fmt.Printf("Year   Herd    Exposed    Bred    Open    Rate     Old   |  Exposed   Bred    Open    Rate    Died\n")
		for i := animal.Burnin + 1; i <= nYears; i++ {

			for n := range animal.Herds {
				var h animal.HerdYear_t
				h.Year = i
				h.Herd = n

				t := animal.BreedingRecordsYearTable[h]

				fmt.Printf("%4d   %-7s %7d %7d %7d %7.1f %7d   | %7d %7d %7d %7.1f %7d\n",
					h.Year,
					h.Herd,
					t.CowsExposed,
					t.CowsBred,
					t.CowsCulledOpen,
					float64(t.CowsBred)/float64(t.CowsExposed)*100.,
					t.CowsCulledOld,
					t.HeifersExposed,
					t.HeifersBred,
					t.HeifersCulledOpen,
					float64(t.HeifersBred)/float64(t.HeifersExposed)*100.,
					t.HeifersDiedCalving)

				cowRate = cowRate + float64(t.CowsBred)/float64(t.CowsExposed)*100.
				heiferRate = heiferRate + float64(t.HeifersBred)/float64(t.HeifersExposed)*100.
			}
		}
		fmt.Printf("Average:                             %10.2f                                  %10.2f\n", cowRate/float64(nYears-animal.Burnin), heiferRate/float64(nYears-animal.Burnin))
	}
}

func main() {

	initSimulation() // Initialize everything

	animal.MakeFoundationCowHerd(varStuff.GvCholesky, varStuff.RvCholesky, param)

	animal.MakeFoundationHeifers(varStuff.GvCholesky, varStuff.RvCholesky, param)

	animal.MakeFoundationBulls(varStuff.GvCholesky, varStuff.RvCholesky, param)

	simulateYears()

	printTables()

	if *indexParm != "" {
		ecoIndex.ProcessNetReturns(indexParm, nYears, burninMarker, param, varStuff.GvCholesky, varStuff.RvCholesky)
	}

	animal.DumpRecords()

	animal.DumpBreedingRecords()

}
