// fatCattle
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
package ecoIndex

import (
	"fmt"

	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/animal"
	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/logger"

	"math"

	"gonum.org/v1/gonum/mat"
)

type fatcattleGrossRevenueByYear_t struct {
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

type fatcattleGrossCostsByYear_t struct {
	SteerCosts            float64
	DiscountedSteerCosts  float64
	HeiferCosts           float64
	DiscountedHeiferCosts float64
}

var fatcattleGrossRevenueByYear map[int]fatcattleGrossRevenueByYear_t
var fatcattleGrossCostsByYear map[int]fatcattleGrossCostsByYear_t

// Calculate backgrounded animals total sale revenue
func fatcattleSaleRevenue(calf animal.Animal) (salePrice float64) {

	weight := calf.HarvestWeight

	min := 0.
	max := 9999.

	var tsmm TraitSexMinWtMaxWt_t
	tsmm.MaxWt = max
	tsmm.MinWt = min
	tsmm.Sex = calf.Sex
	tsmm.Trait = "FC"

	salePrice = weight * pricePerPound[tsmm]
	//fmt.Println("LOC 1", salePrice, weight, tsmm)
	return salePrice
}

// Revenue from sale of fed cattle
func calculateFatcattleRevenueByYear() {

	fatcattleGrossRevenueByYear = make(map[int]fatcattleGrossRevenueByYear_t)

	for _, calf := range animal.Records {

		// Doing this because the distribution for CD is not initialized until after year 1
		// Should probably discard entire year 1 or more results
		if calf.YearBorn == 1 {
			calf.Dead = 0
		}

		yearHarvested := int(float64(calf.BirthDate)+205.+BackgroundDays+animal.DaysOnFeed) / 365
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer {

				w := fatcattleGrossRevenueByYear[yearHarvested]
				w.nSteers++
				w.SteerRevenue += fatcattleSaleRevenue(calf)
				p := calf.HarvestWeight
				w.wtSteers += p
				fatcattleGrossRevenueByYear[yearHarvested] = w

			} else if calf.Sex == animal.Heifer {

				w := fatcattleGrossRevenueByYear[yearHarvested]
				w.nHeifers++
				w.HeiferRevenue += fatcattleSaleRevenue(calf)
				p := calf.HarvestWeight
				w.wtHeifers += p
				fatcattleGrossRevenueByYear[yearHarvested] = w
			}
		} else {
			w := fatcattleGrossRevenueByYear[yearHarvested]
			w.nDead++
			fatcattleGrossRevenueByYear[yearHarvested] = w
		}
	}
	return
}

// Determine the discounted value of the revenue
func calculateDiscountedFatcattleRevenueByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		b := fatcattleGrossRevenueByYear[y]

		df := math.Pow(1.+DiscountRate, period)

		b.DiscountedSteerRevenue = b.SteerRevenue / df
		b.DiscountedHeiferRevenue = b.HeiferRevenue / df

		fatcattleGrossRevenueByYear[y] = b
	}
}

// Value calves as fed cattle
func fatcattleSale(nYears int) float64 {

	if *logger.OutputMode == "verbose" {
		fmt.Println("Processing fatcattle sale net returns...")
	}

	calculateWeaningCostsByYear()
	calculateDiscountedWeaningCostsByYear(nYears)

	calculateBackgroundingCostsByYear()
	calculateDiscountedBackgroundingCostsByYear(nYears)

	calculateFatcattleRevenueByYear()
	calculateDiscountedFatcattleRevenueByYear(nYears)

	calculateFatcattleCostsByYear()
	calculateDiscountedFatcattleCostsByYear(nYears)

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nRevenue from fatcattle calf sales:")
		fmt.Println("Sale Year    n Steers	Steer $     Wt Steers   n Heifers      Heifer $    Wt Heifers    n Dead")
		for year := 1; year <= nYears; year++ {
			fmt.Printf("  %5d      %5d %12.2f  %12.1f       %5d  %12.2f  %12.1f   %7d\n", year, int(fatcattleGrossRevenueByYear[year].nSteers),
				fatcattleGrossRevenueByYear[year].SteerRevenue, fatcattleGrossRevenueByYear[year].wtSteers,
				int(fatcattleGrossRevenueByYear[year].nHeifers), fatcattleGrossRevenueByYear[year].HeiferRevenue, fatcattleGrossRevenueByYear[year].wtHeifers,
				fatcattleGrossRevenueByYear[year].nDead)
		}
	}

	return fatcattleNetReturnsToFixedCosts(nYears) // This returns total accumulated net returns/mating
	return 0.0
}

// Process an index with sale as finished cattle
func EvaluateFatCattleIndex(nYears int, burninMarker int, param map[string]interface{}, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) float64 {

	var base animal.Component_t
	base.Component = "D"
	base.TraitName = "base"
	IndexNetReturns := fatcattleSale(nYears) // Discounted and per mating
	IndexNetReturns += cullSale(nYears)      // Discounted and per mating
	IndexNetReturns -= CowCosts(nYears)

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
func calculateDiscountedFatcattleCostsByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		df := math.Pow(1.+DiscountRate, period)

		c := fatcattleGrossCostsByYear[y]

		c.DiscountedSteerCosts = c.SteerCosts / df
		c.DiscountedHeiferCosts = c.HeiferCosts / df

		fatcattleGrossCostsByYear[y] = c
	}
}

// Calculate cost to feed each year
func calculateFatcattleCostsByYear() {

	fatcattleGrossCostsByYear = make(map[int]fatcattleGrossCostsByYear_t)

	for _, calf := range animal.Records {
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer { // Otherwise its a cow and was not fed
				c := fatcattleGrossCostsByYear[calf.YearBorn+1]
				c.SteerCosts += calf.FeedlotTotalFeedIntake * FeedlotFeedCost
				fatcattleGrossCostsByYear[calf.YearBorn+1] = c
			} else if calf.Sex == animal.Heifer {
				c := fatcattleGrossCostsByYear[calf.YearBorn+1]
				c.HeiferCosts += calf.FeedlotTotalFeedIntake * FeedlotFeedCost
				fatcattleGrossCostsByYear[calf.YearBorn+1] = c

			}
		}
	}
	return
}

//  The total discounted net returns to fixed costs for feeding cattle
// and optionally write a table to stdout
// Returns average net returns to LML/mating
func fatcattleNetReturnsToFixedCosts(nYears int) float64 {

	var TotalDiscountedRevenue float64
	var TotalDiscountedNetRevenuePerMating float64

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nDiscounted Returns and Costs for Finished Cattle:")
		fmt.Println("       Returns_____________________  Costs of Finishing_______      Costs of backgrounding______    Costs of weaning_________")
		fmt.Println("Year    $ Actual     $ Discounted    $ Actual     $ Discounted      $ Actual     $ Discounted       $ Actual     $ Discounted     $ Net/Exposure  N Cows Exposed")
	}

	for y := StartYearOfNetReturns; y <= nYears; y++ {

		TotalDiscountedRevenue += fatcattleGrossRevenueByYear[y].DiscountedSteerRevenue + fatcattleGrossRevenueByYear[y].DiscountedHeiferRevenue

		netPerExposure := (fatcattleGrossRevenueByYear[y].DiscountedSteerRevenue + fatcattleGrossRevenueByYear[y].DiscountedHeiferRevenue -
			fatcattleGrossCostsByYear[y].DiscountedSteerCosts - fatcattleGrossCostsByYear[y].DiscountedHeiferCosts -
			backgroundingGrossCostsByYear[y].DiscountedSteerCosts - backgroundingGrossCostsByYear[y].DiscountedHeiferCosts -
			weaningGrossCostsByYear[y].DiscountedSteerCosts - weaningGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])

		TotalDiscountedNetRevenuePerMating += netPerExposure

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d  %10.2f    %10.2f     %10.2f    %10.2f     %10.2f   %10.2f         %10.2f   %10.2f         %10.2f       %7d\n", y,
				fatcattleGrossRevenueByYear[y].SteerRevenue+fatcattleGrossRevenueByYear[y].HeiferRevenue,
				fatcattleGrossRevenueByYear[y].DiscountedSteerRevenue+fatcattleGrossRevenueByYear[y].DiscountedHeiferRevenue,
				fatcattleGrossCostsByYear[y].SteerCosts+fatcattleGrossCostsByYear[y].HeiferCosts,
				fatcattleGrossCostsByYear[y].DiscountedSteerCosts+fatcattleGrossCostsByYear[y].DiscountedHeiferCosts,
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
