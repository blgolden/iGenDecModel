// slaughterCattle
package ecoIndex

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

import (
	"fmt"

	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	"github.com/blgolden/iGenDecModel/iGenDec/logger"

	"math"
	"math/rand"

	"gonum.org/v1/gonum/mat"
)

type slaughtercattleGrossRevenueByYear_t struct {
	nSteers                 float64
	SteerRevenue            float64
	DiscountedSteerRevenue  float64
	wtSteers                float64
	nHeifers                float64
	HeiferRevenue           float64
	DiscountedHeiferRevenue float64
	wtHeifers               float64
	nDead                   int
}

type slaughtercattleGrossCostsByYear_t struct {
	SteerCosts            float64
	DiscountedSteerCosts  float64
	HeiferCosts           float64
	DiscountedHeiferCosts float64
}

var slaughtercattleGrossRevenueByYear map[int]slaughtercattleGrossRevenueByYear_t
var slaughtercattleGrossCostsByYear map[int]slaughtercattleGrossCostsByYear_t

// Calculate backgrounded animals total sale revenue
func slaughtercattleSaleRevenue(calf animal.Animal) (salePrice float64) {

	weight := calf.CarcassWeight

	min := float64(int(weight/100.) * 100)
	max := float64(int((weight+100.)/100) * 100)

	if min >= 900. {
		min = 900.
		max = 9999.
	} else if max < 600. {
		max = 599.
		min = 0.
	} else {
		min = 600
		max = 900
	}

	var tsmm TraitSexMinWtMaxWt_t
	tsmm.MaxWt = max
	tsmm.MinWt = min
	tsmm.Sex = calf.Sex
	tsmm.Trait = "SC"

	//fmt.Println("LOC 1", salePrice, weight, tsmm)

	// determine the grid price
	ygs := 2.50 +
		(2.5 * calf.BackFatThickness) +
		(0.2 * 2.5) + // Standard KPH fat
		(0.0038 * calf.CarcassWeight) -
		(.32 * calf.RibEyArea)
	//fmt.Println("LOC 2", calf.Id, calf.BackFatThickness, calf.CarcassWeight, calf.RibEyArea, calf.MarblingScore)

	inp := rand.Float64()

	isInProgram := false // Is this calf in special program - e.g., CHB
	if inp <= InProgramProportion {
		isInProgram = true
	}

	qg := "Standard"
	if calf.MarblingScore >= 8.0 {
		qg = "Prime"
	} else if calf.MarblingScore >= 5.0 {
		qg = "Choice"
	} else if calf.MarblingScore >= 4.0 {
		qg = "Select"
	}

	yg := int(ygs)

	var progPremium float64
	if yg <= 3 && isInProgram && calf.MarblingScore >= 5.0 {
		var pValue GridValue_t
		pValue.QualityGrade = "Program"
		pValue.YieldGrade = yg
		progPremium = gridPrice[pValue]
	}

	if yg > 5 {
		yg = 5
	} else if yg < 1 {
		yg = 1
	}

	var gridValue GridValue_t
	gridValue.QualityGrade = qg
	gridValue.YieldGrade = yg

	if animal.CarcassPhenotypeFile != nil {
		fmt.Fprintln(animal.CarcassPhenotypeFile, calf.Id, calf.YearBorn, calf.CarcassWeight, qg, yg, pricePerPound[tsmm], gridPrice[gridValue], progPremium, calf.BackFatThickness, calf.RibEyArea, calf.MarblingScore)
	}
	return weight * (pricePerPound[tsmm] + gridPrice[gridValue] + progPremium)
}

// Revenue from sale of fed cattle
func calculateSlaughtercattleRevenueByYear() {

	slaughtercattleGrossRevenueByYear = make(map[int]slaughtercattleGrossRevenueByYear_t)

	for _, calf := range animal.Records {

		// Doing this because the distribution for CD is not initialized until after year 1
		// Should probably discard entire year 1 or more results
		if calf.YearBorn == 1 {
			calf.Dead = 0
		}

		yearHarvested := int(float64(calf.BirthDate)+205.+BackgroundDays+animal.DaysOnFeed) / 365
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer {

				w := slaughtercattleGrossRevenueByYear[yearHarvested]
				w.nSteers++
				w.SteerRevenue += slaughtercattleSaleRevenue(calf)
				p := calf.HarvestWeight
				w.wtSteers += p
				slaughtercattleGrossRevenueByYear[yearHarvested] = w

			} else if calf.Sex == animal.Heifer {

				w := slaughtercattleGrossRevenueByYear[yearHarvested]
				w.nHeifers++
				w.HeiferRevenue += slaughtercattleSaleRevenue(calf)
				p := calf.HarvestWeight
				w.wtHeifers += p
				slaughtercattleGrossRevenueByYear[yearHarvested] = w
			}
		} else {
			w := slaughtercattleGrossRevenueByYear[yearHarvested]
			w.nDead++
			slaughtercattleGrossRevenueByYear[yearHarvested] = w
		}
	}
	return
}

// Determine the discounted value of the revenue
func calculateDiscountedSlaughtercattleRevenueByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		b := slaughtercattleGrossRevenueByYear[y]

		df := math.Pow(1.+DiscountRate, period)

		b.DiscountedSteerRevenue = b.SteerRevenue / df
		b.DiscountedHeiferRevenue = b.HeiferRevenue / df

		slaughtercattleGrossRevenueByYear[y] = b
	}
}

// Value calves as fed cattle
func slaughtercattleSale(nYears int) float64 {

	if *logger.OutputMode == "verbose" {
		fmt.Println("Processing slaughtercattle sale net returns...")
	}

	calculateWeaningCostsByYear()
	calculateDiscountedWeaningCostsByYear(nYears)

	calculateBackgroundingCostsByYear()
	calculateDiscountedBackgroundingCostsByYear(nYears)

	calculateSlaughtercattleRevenueByYear()
	calculateDiscountedSlaughtercattleRevenueByYear(nYears)

	calculateSlaughtercattleCostsByYear()
	calculateDiscountedSlaughtercattleCostsByYear(nYears)

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nRevenue from slaughtercattle calf sales:")
		fmt.Println("Sale Year    n Steers	Steer $     Wt Steers   n Heifers      Heifer $    Wt Heifers    n Dead")
		for year := 1; year <= nYears; year++ {
			fmt.Printf("  %5d      %5d %12.2f  %12.1f       %5d  %12.2f  %12.1f   %7d\n", year, int(slaughtercattleGrossRevenueByYear[year].nSteers),
				slaughtercattleGrossRevenueByYear[year].SteerRevenue, slaughtercattleGrossRevenueByYear[year].wtSteers,
				int(slaughtercattleGrossRevenueByYear[year].nHeifers), slaughtercattleGrossRevenueByYear[year].HeiferRevenue, slaughtercattleGrossRevenueByYear[year].wtHeifers,
				slaughtercattleGrossRevenueByYear[year].nDead)
		}
	}

	return slaughtercattleNetReturnsToFixedCosts(nYears) // This returns total accumulated net returns/mating
	return 0.0
}

// Process an index with sale as finished cattle
func EvaluateSlaughterCattleIndex(nYears int, burninMarker int, param map[string]interface{}, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) float64 {

	var base animal.Component_t
	base.Component = "D"
	base.TraitName = "base"

	IndexNetReturns := slaughtercattleSale(nYears) // Discounted and per mating
	n := IndexNetReturns
	if *logger.OutputMode == "verbose" {
		fmt.Printf("Total Slaughter Sale Revenue: %f\n\n", IndexNetReturns)
	}

	IndexNetReturns += cullSale(nYears) // Discounted and per mating
	if *logger.OutputMode == "verbose" {
		fmt.Printf("Cull gross revenue: %f\n\n", IndexNetReturns-n)

	}

	if !IsIndexTerminal() {
		n = IndexNetReturns
		IndexNetReturns -= CowCosts(nYears)
		if *logger.OutputMode == "verbose" {
			fmt.Printf("Cow costs: (%f)\n\n", n-IndexNetReturns)
		}
	}

	if *logger.OutputMode == "verbose" {
		fmt.Printf("\nPlanning Horizon (in years):                                            %12d\n", nYears-StartYearOfNetReturns+1)
		//fmt.Printf("%d Year Discounted Net Returns to land, management and labor per exp accum: %12.2f\n", nYears-StartYearOfNetReturns+1, NetReturns)
		fmt.Printf("%d year Discounted Net Returns to land, management and labor per exposure:  %12.2f\n", nYears-StartYearOfNetReturns+1,
			IndexNetReturns)
		fmt.Println("NOTE: all net values are returns to land, management and labor")
	}

	return IndexNetReturns
}

// Determine the discounted value of the costs
func calculateDiscountedSlaughtercattleCostsByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		df := math.Pow(1.+DiscountRate, period)

		c := slaughtercattleGrossCostsByYear[y]

		c.DiscountedSteerCosts = c.SteerCosts / df
		c.DiscountedHeiferCosts = c.HeiferCosts / df

		slaughtercattleGrossCostsByYear[y] = c
	}
}

// Calculate cost to feed each year
func calculateSlaughtercattleCostsByYear() {

	slaughtercattleGrossCostsByYear = make(map[int]slaughtercattleGrossCostsByYear_t)

	for _, calf := range animal.Records {
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer { // Otherwise its a cow and was not fed
				c := slaughtercattleGrossCostsByYear[calf.YearBorn+1]
				c.SteerCosts += calf.FeedlotTotalFeedIntake * FeedlotFeedCost
				slaughtercattleGrossCostsByYear[calf.YearBorn+1] = c
			} else if calf.Sex == animal.Heifer {
				c := slaughtercattleGrossCostsByYear[calf.YearBorn+1]
				c.HeiferCosts += calf.FeedlotTotalFeedIntake * FeedlotFeedCost
				slaughtercattleGrossCostsByYear[calf.YearBorn+1] = c

			}
		}
	}
	return
}

//  The total discounted net returns to fixed costs for feeding cattle
// and optionally write a table to stdout
// Returns average net returns to LML/mating
func slaughtercattleNetReturnsToFixedCosts(nYears int) float64 {

	var TotalDiscountedRevenue float64
	var TotalDiscountedNetRevenuePerMating float64

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nDiscounted Returns and Costs for Finished Cattle:")
		fmt.Println("       Returns_____________________  Costs of Finishing_______      Costs of backgrounding______    Costs of weaning_________")
		fmt.Println("Year    $ Actual     $ Discounted    $ Actual     $ Discounted      $ Actual     $ Discounted       $ Actual     $ Discounted     $ Net/Exposure  N Cows Exposed")
	}

	for y := StartYearOfNetReturns; y <= nYears; y++ {

		TotalDiscountedRevenue += slaughtercattleGrossRevenueByYear[y].DiscountedSteerRevenue + slaughtercattleGrossRevenueByYear[y].DiscountedHeiferRevenue

		netPerExposure := (slaughtercattleGrossRevenueByYear[y].DiscountedSteerRevenue + slaughtercattleGrossRevenueByYear[y].DiscountedHeiferRevenue -
			slaughtercattleGrossCostsByYear[y].DiscountedSteerCosts - slaughtercattleGrossCostsByYear[y].DiscountedHeiferCosts -
			backgroundingGrossCostsByYear[y].DiscountedSteerCosts - backgroundingGrossCostsByYear[y].DiscountedHeiferCosts -
			weaningGrossCostsByYear[y].DiscountedSteerCosts - weaningGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])

		TotalDiscountedNetRevenuePerMating += netPerExposure

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d  %10.2f    %10.2f     %10.2f    %10.2f     %10.2f   %10.2f         %10.2f   %10.2f         %10.2f       %7d\n", y,
				slaughtercattleGrossRevenueByYear[y].SteerRevenue+slaughtercattleGrossRevenueByYear[y].HeiferRevenue,
				slaughtercattleGrossRevenueByYear[y].DiscountedSteerRevenue+slaughtercattleGrossRevenueByYear[y].DiscountedHeiferRevenue,
				slaughtercattleGrossCostsByYear[y].SteerCosts+slaughtercattleGrossCostsByYear[y].HeiferCosts,
				slaughtercattleGrossCostsByYear[y].DiscountedSteerCosts+slaughtercattleGrossCostsByYear[y].DiscountedHeiferCosts,
				backgroundingGrossCostsByYear[y].SteerCosts+backgroundingGrossCostsByYear[y].HeiferCosts,
				backgroundingGrossCostsByYear[y].DiscountedSteerCosts+backgroundingGrossCostsByYear[y].DiscountedHeiferCosts,
				weaningGrossCostsByYear[y].SteerCosts+weaningGrossCostsByYear[y].HeiferCosts,
				weaningGrossCostsByYear[y].DiscountedSteerCosts+weaningGrossCostsByYear[y].DiscountedHeiferCosts,
				netPerExposure,
				animal.CowsExposedPerYear[y])
		}

	}

	return TotalDiscountedNetRevenuePerMating / float64(nYears-StartYearOfNetReturns+1)
}
