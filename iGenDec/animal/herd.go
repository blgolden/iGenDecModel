// herd project herd.go
// Defines the herd characteristics and members
// There can be more than 1 herd - e.g., spring and fall
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
	"math"
	"os"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat/distuv"
)

var Records []Animal
var Herds map[string]Herd // can be more than 1 such as spring v fall

var CowAgeFile *os.File            // For cowagefilename: in master.hjson if exists
var RecordsDumpFile = ""           // name of file in recordsdump:
var BreedingRecordsDumpFile = ""   // name of file in breedingrecordsdump:
var CowsExposedPerYear map[int]int // Counts of the number of cows exposed each year
var Burnin int                     // Number of years to simulate before calculating the MEV
var YearsPlanningHorizon int       // Total Number of years to run the simulation after the burnin for MEV calculation

type BreedComposition_t struct { // Filled in by the CowHerdBreedComposition: key table
	Proportion       float64 // cumulative 1st field of CowHerdBreedComposition: key so last value is 100% or 1.0
	BreedProportions map[string]float64
}

var FoundationCowHerdBreedCompositionTable []BreedComposition_t
var BullBatteryBreedCompositionTable []BreedComposition_t
var CurrentCalvesBreedCompositionTable []BreedComposition_t

type Herd struct { // There can be more than one

	// These are the parameters read from herds: in the hjson
	HerdName          string  // For example "Spring" or "Fall"
	NumberCows        int     // Target number of cows in this herd
	StartBreeding     Date    // Date starting to breed
	BreedingSeasonLen Date    // Length of breeding season in days
	CowConceptionRate float64 // Average Conception rate per 21 d cycle
	Mean3CycleRate    float64 // stay is based on 3 cycles of exposure

	CalvingDifficultyDistribution distuv.Normal // Unadjusted phenotype probability threshold for breeding set in MakeFoundationHeifers()
	InitialCalvingDeathLessRate   float64       // Initial calving difficulty death loss rate

	// These are periodically reset
	Cows   []*Animal // List of cows active in the herd
	Calves []*Animal // List of pre-weaning calves active in the herd
	Bulls  []*Animal // Herd bulls - can be cleanup

	SumBirthDates []float64 // Sum of the birth dates within a year
	NBorn         []float64 // number of calves born in a birth year

}

type HerdYear_t struct {
	Herd string
	Year int
}

type BreedingRecordsTable_t struct {
	CowsExposed        int
	CowsBred           int
	CowsCulledOpen     int
	CowsCulledOld      int
	HeifersExposed     int
	HeifersBred        int
	HeifersCulledOpen  int
	HeifersDiedCalving int
}

var BreedingRecordsYearTable map[HerdYear_t]BreedingRecordsTable_t

//var HeiferResetList []int // List of heifer locates in Records that need to be reset to heifer when bumping index components
var CowResetList []Animal // List of Records[] to reset to active cows when bumping index components

func DumpRecords() {
	if RecordsDumpFile == "" {
		return
	}

	f, _ := os.Create(RecordsDumpFile)
	defer f.Close()
	/*fmt.Fprintf(f, "RecNo ID Dam ")
	for name, _ := range TraitMean {
		fmt.Fprintf(f, "%v ", name)
	}
	fmt.Fprintf(f, "\n")*/

	for i := 0; i < len(Records); i++ {
		w := MatureWeightAtAgePhenotype(Records[i], Date(1735))
		p := Records[i].BreedingValue.AtVec(13)
		fmt.Fprintf(f, "%5d %5d %s %5d %f %f ", i, Records[i].Id, Records[i].Sex,
			Records[i].BirthDate, w, p)
		/*for t := range Traits {
			if r, ok := Phenotype(Records[i], Traits[t]); ok {
				fmt.Fprintf(f, "%5s: %7.2f ", Traits[t], r)
			} else {
				fmt.Fprintf(f, "%5s: -      ", Traits[t])
				/*
					for k := 0; k < Records[i].BreedingValue.Len(); k++ {
						fmt.Fprint(f, Records[i].BreedingValue.AtVec(k), " ")
					}
					fmt.Fprintf(f, "\n")
					//
			}
		}*/
		fmt.Fprintf(f, "\n")
		//resIdx := ResidualIndex("USREA")
		//fmt.Fprintln(f, Records[i].Residual.At(resIdx, resIdx))
	}
}
func Year(birthDate Date) int {
	return int(float64(birthDate)/365. + .5)
}

/*
func PrintCowAgeDistribution(herds []Herd, thisYear int, param map[string]interface{}) {

	for h := range herds {
		herds[h].Cows = ActiveCows(&herds[h])

		thisHerd := &herds[h]

		//array := param["ageDist"].([]interface{})

		var ageCounts = make([]int, MaxCowAge+1)

		fmt.Println("Herd name:", thisHerd.HerdName)
		fmt.Println("Total number of cows in this herd (after culling opens): ", len(thisHerd.Cows))
		fmt.Println("Number cows calved by age for Year: ", thisYear+1)
		for i := 0; i < len(thisHerd.Cows); i++ {

			thisAge := int(math.Round((float64(
				thisHerd.Cows[i].BreedingRecords[len(thisHerd.Cows[i].BreedingRecords)-1].CalvingDate-
					thisHerd.Cows[i].BirthDate) / 365.)))
			//fmt.Println("LOC 3", thisAge)
			ageCounts[thisAge]++
		}

		for i := 2; i < len(ageCounts); i++ {
			fmt.Printf("% 5d % 5d\n", i, ageCounts[i])
		}
	}

}
*/
func WriteCowAgeDistribution(herds map[string]Herd, thisYear int, param map[string]interface{}) {

	if CowAgeFile == nil {
		return
	}

	for _, h := range herds {
		h.Cows = ActiveCows(&h)

		thisHerd := &h

		var ageCounts = make([]int, MaxCowAge+1)

		for i := 0; i < len(thisHerd.Cows); i++ {

			thisAge := int(math.Round((float64(
				thisHerd.Cows[i].BreedingRecords[len(thisHerd.Cows[i].BreedingRecords)-1].CalvingDate-
					thisHerd.Cows[i].BirthDate) / 365.)))
			if thisAge > MaxCowAge {
				thisAge = MaxCowAge
			}

			ageCounts[thisAge]++
		}

		fmt.Fprintf(CowAgeFile, "%5d ", thisYear)
		for i := 2; i < len(ageCounts); i++ {
			fmt.Fprintf(CowAgeFile, "%5d", ageCounts[i])
		}
		fmt.Fprintf(CowAgeFile, "\n")
	}

}

// Simulate a base set of records to calculate where the MEV deviate from
func SimulateBase(Param map[string]interface{}, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) {

	// Must go 2 years beyond to get tthe heifers out, etc.

	var bPlusPh int
	if IndexTerminal {
		bPlusPh = Burnin + YearsPlanningHorizon
	} else {
		bPlusPh = Burnin + YearsPlanningHorizon + 2
	}

	for year := Burnin + 1; year <= bPlusPh; year++ {
		for _, h := range Herds {

			Breed(&h, year)

			Calve(&h, year, gvCholesky, rvCholesky)

			CullOpen(&h, year)

			WriteCowAgeDistribution(Herds, year-1, Param)

			CullOld(&h, year)

			DetermineCowAum(&h, year)

		}
	}
}

// Determine the AUM consumption for active cows at end of year
// Considering that some may be heifers that entered
func DetermineCowAum(h *Herd, year int) {
	h.Cows = ActiveCows(h)

	for _, c := range h.Cows {

		startMonth := int(float64(year*365-c.DateCowEntered)/30.42) + 1

		if startMonth > 12 { // She entered in the next year e.g. heifer from spring weaning
			startMonth = 1
		}
		if startMonth < 1 {
			return
		}

		for m := startMonth; m <= 12; m++ {
			var aum Aum_t
			aum.Year = year
			aum.MonthOfYear = m
			w := MatureWeightAtAgePhenotype(*c, Date(year*365+m*30))
			aum.Aum = w / 1000. * CowAumAt1000
			aum.Location = 1
			if w > 0 {
				Records[c.Id-1].CowAum = append(Records[c.Id-1].CowAum, aum)
			}
		}
	}

}
