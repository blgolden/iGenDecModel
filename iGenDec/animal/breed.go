// breed
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
	"math/rand"
	"os"

	"github.com/blgolden/iGenDecModel/iGenDec/logger"

	"gonum.org/v1/gonum/mat"
	//"gonum.org/v1/gonum/stat/distuv"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
func randRange(min, max int) int {
	if max-min <= 0 {
		return min
	}
	return rand.Intn(max-min) + min
}

// Refresh the list of active cows in the herd
func ActiveCows(herd *Herd) []*Animal {
	var cows []*Animal
	for r := range Records {
		if Records[r].Active && Records[r].Sex == Cow && Records[r].HerdName == herd.HerdName {
			cows = append(cows, &Records[r])
		}
	}
	return cows
}

var NHeifersBred map[int]int

// Refresh the list of active cows in the herd
func ActiveBulls(herd *Herd) []*Animal {
	var bulls []*Animal
	for r := range Records {
		if Records[r].Active && Records[r].Sex == Bull && Records[r].HerdName == herd.HerdName {
			bulls = append(bulls, &Records[r])
		}
	}
	return bulls
}
func GestationLengthError() int {

	r := rand.NormFloat64() * 5.0
	er := int(r)
	return er
}

// Is this going to calv as a 2 yoa heifer this year
func is2YoaHeifer(a *Animal, herd *Herd, year int) bool {
	if a.Sex != Heifer {
		return false
	}

	ageAtThisBreeding := (year-1)*365 + int(herd.StartBreeding) - int(a.BirthDate)

	is2 := false

	if ageAtThisBreeding >= 365 && ageAtThisBreeding <= 365+int(herd.StartBreeding)+1 {
		is2 = true
		//fmt.Println("LOC 1 ", a.Id, year, a.BirthDate, ageAtThisBreeding)
	}
	//*
	//if year == 4 {
	//fmt.Println("LOC 1 ", a.Id, year, a.BirthDate, ageAtThisBreeding, is2)
	//}
	//*/

	return is2

}

// Select the replacement heifers to enter the cow herd
func Replace(herd *Herd, year int) {

	herd.Cows = ActiveCows(herd)

	nReplacements := herd.NumberCows - len(herd.Cows)
	var replace int

	if year == 1 || nReplacements == 0 {
		if *logger.OutputMode == "verbose" && year > 1 {
			fmt.Printf("\nNo replacements needed in the %v herd, year: %d\n", herd.HerdName, year)
		}
		return
	}

	// THIS WILL BE MODIFIED TO SELECT HEIFERS BY INDEX
	// For now it just grabs the first available.
	for i := herd.Cows[len(herd.Cows)-1].Id - 1; i < AnimalId(len(Records)-1); i++ {
		//for i := range Records {
		if is2YoaHeifer(&Records[i], herd, year) {
			Records[i].Active = true
			Records[i].Sex = Cow
			Records[i].DateCowEntered = int(herd.SumBirthDates[year-1]/herd.NBorn[year-1]) + 205 + 365
			replace++
			if replace == nReplacements {
				if *logger.OutputMode == "verbose" {
					fmt.Printf("Replaced %d cows with heifers in the %v herd, year: %d\n", nReplacements, herd.HerdName, year)
				}
				herd.Cows = ActiveCows(herd)
				return
			}
		}

	}
	if *logger.OutputMode == "verbose" {
		fmt.Printf("\nWARNING: Insufficient replacement females to maintain %v herd\n", herd.HerdName)
		fmt.Printf("\tNeeded: %d but had %d available in year %d\n", nReplacements, replace, year)
	} // need to add logging

	herd.Cows = ActiveCows(herd)
}

// Has this animal reached puberty
func Puberty(herd *Herd, i int, bredDate Date) bool {

	ageAtExposure := bredDate - herd.Cows[i].BirthDate + 1
	ageAtPuberty := 352 // Stubbed in for now from Thallman, et al., 1999 for Angus and Hereford
	/*if ageAtExposure < 352 {
		fmt.Println("LOC 1 ", herd.Cows[i].Id, herd.Cows[i].BirthDate, bredDate, ageAtExposure)
	}*/
	if int(ageAtExposure) >= ageAtPuberty {
		return true
	} else {
		return false
	}

}

// Breed the cow herds
func Breed(herd *Herd, year int) {
	//fmt.Printf("LOC 1 %d %d\n", len(herd.Cows), year)

	herd.Cows = ActiveCows(herd)
	herd.Bulls = ActiveBulls(herd)

	Replace(herd, year) // Set the replacement heifers to active cows

	CowsExposedPerYear[year] = len(herd.Cows)

	var sumSquared float64
	var sum float64
	var n float64
	var nCycles = int(herd.BreedingSeasonLen/21 + 1)

	//fmt.Println("LOC BR", year, herd.CowConceptionRate, acum, cycp)

	for i := range herd.Cows {

		var thisBreeding BreedingRec

		// is this a heifer
		thisAgeAtBreedingStart := Date((year-1)*365) + herd.StartBreeding - herd.Cows[i].BirthDate

		for cycle := 1; cycle <= nCycles; cycle++ { // cycle is estrus

			clen := min(int(herd.BreedingSeasonLen)-(cycle-1)*21, 21)
			propClen := float64(clen) / 21.0

			breddate := Date((cycle-1)*21+randRange(1, clen)) + herd.StartBreeding

			var p float64

			isHeifer := false
			//var geneticDirectEffect float64

			// Do this here because the age effects were impactful
			if thisAgeAtBreedingStart < 365+365/2 { // Yearling heifer
				p = HeiferPregnancyPhenotype(*herd.Cows[i], Date((year-1)*365)+breddate) + herd.Mean3CycleRate*propClen
				isHeifer = true
			} else { // This is a cow
				stay := StayAtAgePhenotype(*herd.Cows[i], Date((year-1)*365)+breddate) + herd.Mean3CycleRate
				p = Stay2Concept21days(stay) * propClen
				//fmt.Println("LOC_2", stay, p, herd.Mean3CycleRate, herd.CowConceptionRate, propClen, cycle, herd.Cows[i].Id)
				//p += herd.CowConceptionRate
			}

			//conceive := rand.Float64()

			//if p > conceive {
			if p > herd.CowConceptionRate {
				/*if isHeifer {
					fmt.Println("LOC CO", herd.Cows[i].Id, year, p, concieve, cycle, cycp[cycle-1], isHeifer)
				}*/
				thisBreeding.DateBred = breddate
				thisBreeding.YearBred = year
				thisBreeding.Bred = true
				thisBreeding.Bull = herd.Bulls[randRange(0, len(herd.Bulls)-1)].Id
				thisBreeding.CalvingDate = Date((year-1)*365) + thisBreeding.DateBred + GestationLength() // need to change to account for GL std dev

				// Initialize the calving difficulty distribution
				//if isHeifer && herd.Cows[i].YearBorn == 0 && cycle == 1 {
				if isHeifer && cycle == 1 {
					pheno := CalvingDifficultyPhenotype(*herd.Cows[i], thisBreeding)
					sumSquared += pheno * pheno
					sum += pheno
					n += 1.0
					NHeifersBred[year]++
				}

				break
			}
		}
		if !thisBreeding.Bred {
			thisBreeding.Bred = Open
			thisBreeding.DateBred = Date(randRange(1, int(herd.BreedingSeasonLen)))
			thisBreeding.YearBred = year
		}

		herd.Cows[i].BreedingRecords = append(herd.Cows[i].BreedingRecords, thisBreeding)
		//fmt.Println(i, thisBreeding)
	}
	/*if n > 0 {
		// Initialize the calving difficulty distribution
		herd.CalvingDifficultyDistribution.Sigma = math.Sqrt((sumSquared - (sum * (sum / n))) / (n - 1.0))
		herd.CalvingDifficultyDistribution.Mu = TraitMean["CD"]
		Herds[herd.HerdName] = *herd
	} else {*/
	// This is initialized in initSimulation
	herd.CalvingDifficultyDistribution.Sigma = math.Sqrt(CDVar)
	herd.CalvingDifficultyDistribution.Mu = TraitMean["CD"]

	Herds[herd.HerdName] = *herd
	//}
}

func DumpBreedingRecords() {

	if BreedingRecordsDumpFile == "" {
		return
	}

	var f *os.File
	f, _ = os.Create(BreedingRecordsDumpFile)

	defer f.Close()

	for i := range Records {
		//fmt.Fprintf(f, "%d - ", thisHerd.Cows[i].Id)
		if Records[i].BreedComposition != nil {
			for j := range Records[i].BreedingRecords {
				r := Records[i].BreedingRecords[j]
				if r.Bred {
					fmt.Fprintln(f, Records[i].Id, r.DateBred, r.Bred, r.Bull, r.CalvingDate, r.YearBred)
				} else {
					fmt.Fprintln(f, Records[i].Id, r.DateBred, r.Bred, 0, r.YearBred)
				}
			}
		}
	}
}

// Cull the open cows and older than maximum age allowed in master.hjson
func CullOpen(herd *Herd, year int) {

	herd.Cows = ActiveCows(herd)

	var h HerdYear_t
	h.Herd = herd.HerdName
	h.Year = year

	b := BreedingRecordsYearTable[h]

	for r := range herd.Cows {
		thisCow := herd.Cows[r]

		if thisCow.Active {
			//fmt.Println("LOC CULL", thisCow.YearBorn, year)
			if year-thisCow.YearBorn == 2 {
				b.HeifersExposed++
				if thisCow.BreedingRecords[len(thisCow.BreedingRecords)-1].Bred == Open || thisCow.Dead > 0 {
					if thisCow.Dead == 0 {
						b.HeifersCulledOpen++
						herd.Cows[r].Active = false
						herd.Cows[r].DateCowCulled = int(herd.SumBirthDates[year]/herd.NBorn[year]) + 205
					} else {
						b.HeifersDiedCalving++
						herd.Cows[r].Active = false
						herd.Cows[r].DateCowCulled = int(thisCow.Dead)
					}

					CullAum(herd.Cows[r], year, 2)

				} else {
					b.HeifersBred++
				}
			} else {
				b.CowsExposed++
				if thisCow.BreedingRecords[len(thisCow.BreedingRecords)-1].Bred == Open {
					b.CowsCulledOpen++
					herd.Cows[r].Active = false
					herd.Cows[r].DateCowCulled = int(herd.SumBirthDates[year]/herd.NBorn[year]) + 205
					CullAum(herd.Cows[r], year, 3)
					w := MatureWeightAtAgePhenotype(*thisCow, Date(herd.Cows[r].DateCowCulled))
					s := WtCullCows[year]
					s.CumWt += w
					s.NheadOpen++
					WtCullCows[year] = s
				} else {
					b.CowsBred++
				}
			}
		}
	}
	/*
		fmt.Printf("Year %d %v Herd Breeding Summary:\n", year, herd.HerdName)
		fmt.Printf("\tCows exposed:        % 5d\n", len(herd.Cows))
		herd.Cows = ActiveCows(herd)
		fmt.Printf("\tCows bred:           % 5d\n", len(herd.Cows))
		fmt.Printf("\tCows culled open:    % 5d\n", culledOpen)
	*/

	BreedingRecordsYearTable[h] = b
}

// Cull the open cows and older than maximum age allowed in master.hjson
func CullOld(herd *Herd, year int) {
	var h HerdYear_t
	h.Herd = herd.HerdName
	h.Year = year

	b := BreedingRecordsYearTable[h]

	herd.Cows = ActiveCows(herd)

	for r := range herd.Cows {
		thisCow := herd.Cows[r]

		if herd.Cows[r].Active {
			thisAge := int(math.Round(float64(year*365-int(thisCow.BirthDate)) / 365.))

			if thisAge >= MaxCowAge {
				herd.Cows[r].Active = false
				herd.Cows[r].DateCowCulled = int(herd.SumBirthDates[year]/herd.NBorn[year]) + 205
				CullAum(herd.Cows[r], year, 4)

				b.CowsCulledOld++

				w := MatureWeightAtAgePhenotype(*thisCow, Date(year*365))
				s := WtCullCows[year]
				s.CumWt += w
				s.NheadOld++
				WtCullCows[year] = s
			}
		}
	}

	BreedingRecordsYearTable[h] = b

	herd.Cows = ActiveCows(herd)

	return
}

// Determine the AUM a cow consumed in the year it was culled
func CullAum(thisCow *Animal, year int, loc int) {

	nMonths := int(float64(thisCow.DateCowCulled-year*365)/30.42) + 1

	for m := 1; m <= nMonths; m++ {
		var a Aum_t
		yr := year
		curMonth := m
		if m > 12 {
			curMonth = m - 12
			yr = year + 1
		}
		a.MonthOfYear = curMonth
		a.Year = yr
		w := MatureWeightAtAgePhenotype(*thisCow, Date(year*365+curMonth*30))
		frac := w / 1000. * CowAumAt1000
		a.Aum = frac
		a.Location = loc
		Records[thisCow.Id-1].CowAum = append(Records[thisCow.Id-1].CowAum, a)
	}
	//fmt.Println("\nLOC CULL", thisCow.Id, Records[thisCow.Id-1].CowAum)
}

// Calculate monthly AUM consumption of new animal to yearling age
func DetermineAumToWeaning(newCalf *Animal) {

	ww, ok := WeaningWtPhenotype(*newCalf)
	bw, _ := Phenotype(*newCalf, "BW")

	if !ok {
		panic(ok)
	}

	aveWd := Herds[newCalf.HerdName].SumBirthDates[newCalf.YearBorn]/Herds[newCalf.HerdName].NBorn[newCalf.YearBorn] + 205.0
	mw := aveWd/365. - float64(int(aveWd/365.))
	monthWeaned := int(mw*12.) + 1
	if monthWeaned > 12 {
		monthWeaned = monthWeaned - 12
	}

	ageAtWeaning := aveWd - float64(newCalf.BirthDate)

	birthDayOfYear := float64(newCalf.BirthDate) - (float64(newCalf.YearBorn)*365 - 1.)

	birthMonth := int(birthDayOfYear/30.42) + 1

	fracMonth := (float64(birthMonth)*30.42 - float64(birthDayOfYear)) / 30.42

	//monthWeaned := int(ageAtWeaning/30.42) + 1

	adg := (ww - bw) / ageAtWeaning

	q := 1. / (ageAtWeaning / 2.0)

	days := fracMonth * 30.42
	cumDays := days
	cumWt := bw + q*adg*((days-.5)/2.0)*(1.0+(days-.5))

	var Aum Aum_t
	Aum.Weight = cumWt
	Aum.MonthOfYear = birthMonth
	Aum.Aum = cumWt / 500. * CalfAumAt500

	Aum.Year = newCalf.YearBorn

	newCalf.AumToWeaning = append(newCalf.AumToWeaning, Aum)
	cumAum := Aum.Aum

	month := birthMonth
	cd := cumDays
	yr := newCalf.YearBorn

	for m := cd; m < ageAtWeaning; m = m + 30.42 {

		days := ageAtWeaning - cumDays
		if days > 30.42 {
			days = 30.42
		}
		cumDays += days

		wt := bw + q*adg*((cumDays-.5)/2.0)*(1.0+(cumDays-.5)) // Subtract the .5 because the wt is mid-day of weaning day - trust me it works

		var Aumn Aum_t
		Aumn.Weight = wt
		Aumn.Aum = wt / 500. * CalfAumAt500
		cumAum += Aumn.Aum
		month++
		if month > 12 {
			month = month - 12
			yr = newCalf.YearBorn + 1
		}
		Aumn.MonthOfYear = month
		Aumn.Year = yr
		// fmt.Println("LOC INA", birthMonth, m, month, cumDays, wt, ww, Aumn.Aum, cumAum, cumDays)
		newCalf.AumToWeaning = append(newCalf.AumToWeaning, Aumn)
	}

}

// Calculate this animal's total feedlot feed intake
// And slaughter weight
func DetermineFeedlotFeedIntake(newCalf *Animal) {

	feedlotDailyFeedIntake, _ := Phenotype(*newCalf, "FI")

	newCalf.FeedlotTotalFeedIntake = feedlotDailyFeedIntake * DaysOnFeed

	newCalf.CarcassWeight, _ = Phenotype(*newCalf, "HCW")
	stdSlaughterWeight := newCalf.CarcassWeight / .63 // 63% dressing percentage

	inWeight := newCalf.AumWeanThruBackgrounding[len(newCalf.AumWeanThruBackgrounding)-1].Weight

	// feedlotAverageDailyGain := (stdSlaughterWeight - inWeight) / 140. // Standard feeding period length for the avg HCW
	feedlotAverageDailyGain := (stdSlaughterWeight - inWeight) / DaysOnFeed // Changed to reflect user's input means

	//fmt.Println("LOC 0", newCalf.Id, inWeight, feedlotAverageDailyGain, feedlotDailyFeedIntake, DaysOnFeed)
	newCalf.HarvestWeight = inWeight + feedlotAverageDailyGain*DaysOnFeed

	newCalf.MarblingScore, _ = Phenotype(*newCalf, "MS")
	newCalf.BackFatThickness, _ = Phenotype(*newCalf, "FAT")
	newCalf.RibEyArea, _ = Phenotype(*newCalf, "REA")

}

// Calculate monthly AUM consumption of new animal to yearling age
// This also determines the weight at the end of the background period
func DetermineAumThruBackgrounding(newCalf *Animal) {

	ww, ok := WeaningWtPhenotype(*newCalf)
	yw, _ := Phenotype(*newCalf, "YW")

	if !ok {
		panic(ok)
	}

	// Date weaned
	aveWd := Herds[newCalf.HerdName].SumBirthDates[newCalf.YearBorn]/Herds[newCalf.HerdName].NBorn[newCalf.YearBorn] + 205.0
	mw := aveWd/365. - float64(int(aveWd/365.))
	monthWeaned := int(mw*12.) + 1
	if monthWeaned > 12 {
		monthWeaned = monthWeaned - 12
	}

	ageAtWeaning := aveWd - float64(newCalf.BirthDate)
	AgeAtBackground := ageAtWeaning + BackgroundDays
	ageAtYearling := ageAtWeaning + 160

	adg := (yw - ww) / (ageAtYearling - ageAtWeaning)

	q := 1. / (ageAtWeaning / 2.0)

	birthDayOfYear := float64(newCalf.BirthDate) - (float64(newCalf.YearBorn)*365 - 1.)
	birthMonth := int(birthDayOfYear/30.42) + 1
	fracMonth := (float64(birthMonth)*30.42 - float64(birthDayOfYear)) / 30.42

	days := fracMonth * 30.42
	cumDays := days
	cumWt := ww + q*adg*((days-.5)/2.0)*(1.0+(days-.5))

	var Aum Aum_t
	Aum.Weight = cumWt
	Aum.MonthOfYear = monthWeaned
	Aum.Aum = cumWt / 500. * CalfAumAt500

	yearWeaned := int((aveWd + BackgroundDays) / 365.) //NOT SURE THIS IS RIGHT I think it is
	Aum.Year = yearWeaned

	newCalf.AumWeanThruBackgrounding = append(newCalf.AumWeanThruBackgrounding, Aum)
	cumAum := Aum.Aum

	month := birthMonth
	cd := cumDays
	yr := newCalf.YearBorn

	for m := cd + ageAtWeaning; m < AgeAtBackground; m = m + 30.42 {

		days := AgeAtBackground - cumDays - ageAtWeaning
		if days > 30.42 {
			days = 30.42
		}
		cumDays += days

		wt := ww + q*adg*((cumDays-.5)/2.0)*(1.0+(cumDays-.5)) // Subtract the .5 because the wt is mid-day of weaning day - trust me it works

		var Aumn Aum_t
		Aumn.Weight = wt
		Aumn.Aum = wt / 500. * CalfAumAt500
		cumAum += Aumn.Aum
		month++
		if month > 12 {
			month = month - 12
			yr = newCalf.YearBorn + 1
		}
		Aumn.MonthOfYear = month
		Aumn.Year = yr
		// fmt.Println("LOC INA", birthMonth, m, month, cumDays, wt, ww, Aumn.Aum, cumAum, cumDays)
		newCalf.AumWeanThruBackgrounding = append(newCalf.AumWeanThruBackgrounding, Aumn)
	}

}

// Calve and produce phenotypes for the calves for entire life not just this year.
func Calve(herd *Herd, year int, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) {

	herd.Cows = ActiveCows(herd)

	newCalves := len(Records)

	for i, c := range herd.Cows {
		if c.BreedingRecords[len(c.BreedingRecords)-1].Bred != Open {
			GenFromMating(herd.Cows[i], year, gvCholesky, rvCholesky)
		}
	}

	for i := newCalves; i < len(Records); i++ {
		DetermineAumToWeaning(&Records[i])

		if IndexType != "weaning" { // everything marketed after weaning is going to need Aum through backgrounding

			DetermineAumThruBackgrounding(&Records[i]) // Even calf feds get 1 day so that a feedlot inweight can be taken

			if IndexType == "fatcattle" || IndexType == "slaughtercattle" {
				DetermineFeedlotFeedIntake(&Records[i]) // It also calculates and stores slaughter weight
			}

		}
	}
}

// Did the cow or calf die in calving?
// This is called from GenFromMating() in GenBV.go
func diedCalving(cow *Animal, calf *Animal) {

	// Is this a heifer - if not then no issue
	cowAge := calf.BirthDate - cow.BirthDate
	if cowAge > 912 {
		return
	}

	for _, b := range cow.BreedingRecords {
		cd := CalvingDifficultyPhenotype(*calf, b)
		prob := Herds[Records[calf.Dam].HerdName].CalvingDifficultyDistribution.CDF(cd)
		//fmt.Println("LOC 3", cow.Id, prob, cd, Herds[Records[calf.Dam].HerdName].CalvingDifficultyDistribution, Herds[Records[calf.Dam].HerdName].InitialCalvingDeathLessRate)
		if prob >= 1.-Herds[Records[calf.Dam].HerdName].InitialCalvingDeathLessRate {
			calf.Dead = calf.BirthDate
			cow.Dead = calf.BirthDate
			//fmt.Println("LOC 6", calf.Id, calf.Dam, calf.Dead, calf.YearBorn)
		}
	}
}
