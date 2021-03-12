// background
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
	"math"

	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	"github.com/blgolden/iGenDecModel/iGenDec/logger"

	"gonum.org/v1/gonum/mat"
)

var BackgroundDays float64 // length of the backgrounding program after weaning

type backgroundingGrossRevenueByYear_t struct {
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

type backgroundingGrossCostsByYear_t struct {
	SteerCosts            float64
	DiscountedSteerCosts  float64
	HeiferCosts           float64
	DiscountedHeiferCosts float64
}

var backgroundingGrossRevenueByYear map[int]backgroundingGrossRevenueByYear_t
var backgroundingGrossCostsByYear map[int]backgroundingGrossCostsByYear_t

// Calculate backgrounded animals total sale revenue
func backgroundingSaleRevenue(calf animal.Animal) (salePrice float64) {

	weight := animal.BackgroundingWtPhenotype(calf)

	min := float64(int(weight/100.) * 100)
	max := float64(int((weight+100.)/100) * 100)

	if calf.Sex == animal.Steer { // This is set like weaning for now
		if min >= 800. {
			min = 800.
			max = 9999.
		} else if max <= 400. {
			max = 400.
			min = 0.
		}
	} else if calf.Sex == animal.Heifer { // This works beacause Sex was set to Cow if she became a replacement
		if min >= 700. {
			min = 700.
			max = 9999.
		} else if max <= 400. {
			max = 400.
			min = 0.
		}
	}
	/*
		var tsmm TraitSexMinWtMaxWt_t
		tsmm.MaxWt = max
		tsmm.MinWt = min
		tsmm.Sex = calf.Sex
		tsmm.Trait = "BG"
	*/
	pricePerPound := getPricePerPound(weight, calf.Sex, "BG")
	salePrice = weight * pricePerPound

	//fmt.Println("LOC 1", weight, pricePerPound[tsmm], min, max, tsmm)
	return salePrice
}

// Revenue from sale of backgrounded calves
func calculateBackgroundingRevenueByYear() {

	backgroundingGrossRevenueByYear = make(map[int]backgroundingGrossRevenueByYear_t)

	for _, calf := range animal.Records {

		// Doing this because the distribution for CD is not initialized until after year 1
		// Should probably discard entire year 1 or more results
		/*if calf.YearBorn == 1 {
			calf.Dead = 0
		}*/

		yearBackgrounded := int(float64(calf.BirthDate)+205.+BackgroundDays) / 365
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer {

				w := backgroundingGrossRevenueByYear[yearBackgrounded]
				w.nSteers++
				w.SteerRevenue += backgroundingSaleRevenue(calf)
				p := calf.AumWeanThruBackgrounding[len(calf.AumWeanThruBackgrounding)-1].Weight
				w.wtSteers += p
				backgroundingGrossRevenueByYear[yearBackgrounded] = w

			} else if calf.Sex == animal.Heifer {

				w := backgroundingGrossRevenueByYear[yearBackgrounded]
				w.nHeifers++
				w.HeiferRevenue += backgroundingSaleRevenue(calf)
				p := calf.AumWeanThruBackgrounding[len(calf.AumWeanThruBackgrounding)-1].Weight
				w.wtHeifers += p
				backgroundingGrossRevenueByYear[yearBackgrounded] = w
			}
		} else {
			w := backgroundingGrossRevenueByYear[yearBackgrounded]
			w.nDead++
			//fmt.Println("LOC 1", calf.YearBorn, yearBackgrounded)
			backgroundingGrossRevenueByYear[yearBackgrounded] = w
		}
	}
	return
}

// Determine the discounted value of the revenue
func calculateDiscountedBackgroundingRevenueByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		b := backgroundingGrossRevenueByYear[y]

		df := math.Pow(1.+DiscountRate, period)

		b.DiscountedSteerRevenue = b.SteerRevenue / df
		b.DiscountedHeiferRevenue = b.HeiferRevenue / df

		backgroundingGrossRevenueByYear[y] = b
	}
}

// Average backgrounding costs per mating accross years of simulation
// Not used by backgrounding index but used by fat cattle, and grade and yield
func calculateBackgroundingCostsPerMating(nYears int) float64 {

	var totalDiscountedBackgroundingCostPerMating float64

	for y := StartYearOfNetReturns; y <= nYears; y++ {
		totalDiscountedBackgroundingCostPerMating += (backgroundingGrossCostsByYear[y].DiscountedSteerCosts + backgroundingGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])
	}
	return totalDiscountedBackgroundingCostPerMating / float64(nYears-StartYearOfNetReturns+1)
}

//  The total discounted net returns to fixed costs for backgrounding
// and optionally write a table to stdout
// Returns average cost/mating
func backgroundingNetReturnsToFixedCosts(nYears int) float64 {

	var TotalDiscountedRevenue float64
	var TotalDiscountedNetRevenuePerMating float64

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nDiscounted Returns and Costs for Backgrounded Calves:")
		fmt.Println("       Returns_____________________  Costs of backgrounding______    Costs of weaning_________ ")
		fmt.Println("Year    $ Actual     $ Discounted    $ Actual     $ Discounted       $ Actual     $ Discounted     $ Net/Exposure  N Cows Exposed")
	}

	for y := StartYearOfNetReturns; y <= nYears; y++ {

		TotalDiscountedRevenue += backgroundingGrossRevenueByYear[y].DiscountedSteerRevenue + backgroundingGrossRevenueByYear[y].DiscountedHeiferRevenue

		netPerExposure := (backgroundingGrossRevenueByYear[y].DiscountedSteerRevenue + backgroundingGrossRevenueByYear[y].DiscountedHeiferRevenue -
			backgroundingGrossCostsByYear[y].DiscountedSteerCosts - backgroundingGrossCostsByYear[y].DiscountedHeiferCosts -
			weaningGrossCostsByYear[y].DiscountedSteerCosts - weaningGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])

		TotalDiscountedNetRevenuePerMating += netPerExposure

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d  %10.2f    %10.2f     %10.2f   %10.2f         %10.2f   %10.2f         %10.2f       %7d\n", y,
				backgroundingGrossRevenueByYear[y].SteerRevenue+backgroundingGrossRevenueByYear[y].HeiferRevenue,
				backgroundingGrossRevenueByYear[y].DiscountedSteerRevenue+backgroundingGrossRevenueByYear[y].DiscountedHeiferRevenue,
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

// Value calves after backgrounding
func backgroundingSale(nYears int) float64 {

	if *logger.OutputMode == "verbose" {
		fmt.Println("Processing backgrounding sale net returns...")
	}

	calculateWeaningCostsByYear()
	calculateDiscountedWeaningCostsByYear(nYears)

	calculateBackgroundingRevenueByYear()
	calculateDiscountedBackgroundingRevenueByYear(nYears)

	calculateBackgroundingCostsByYear()
	calculateDiscountedBackgroundingCostsByYear(nYears)

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nRevenue from backgrounded calf sales:")
		fmt.Println("Sale Year    n Steers	Steer $     Wt Steers   n Heifers      Heifer $    Wt Heifers    n Dead")
		for year := 1; year <= nYears; year++ {
			fmt.Printf("  %5d      %5d %12.2f  %12.1f       %5d  %12.2f  %12.1f   %7d\n", year, int(backgroundingGrossRevenueByYear[year].nSteers),
				backgroundingGrossRevenueByYear[year].SteerRevenue, backgroundingGrossRevenueByYear[year].wtSteers,
				int(backgroundingGrossRevenueByYear[year].nHeifers), backgroundingGrossRevenueByYear[year].HeiferRevenue, backgroundingGrossRevenueByYear[year].wtHeifers,
				backgroundingGrossRevenueByYear[year].nDead)
		}
	}

	return backgroundingNetReturnsToFixedCosts(nYears) // This returns total accumulated net returns/mating

}

// Process an index with sale at after backgrounding
func EvaluateBackgroundingIndex(nYears int, burninMarker int, param map[string]interface{}, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) float64 {

	var base animal.Component_t
	base.Component = "D"
	base.TraitName = "base"
	IndexNetReturns := backgroundingSale(nYears) // Discounted and per mating
	IndexNetReturns += cullSale(nYears)          // Discounted and per mating
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
func calculateDiscountedBackgroundingCostsByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		df := math.Pow(1.+DiscountRate, period)

		c := backgroundingGrossCostsByYear[y]

		c.DiscountedSteerCosts = c.SteerCosts / df
		c.DiscountedHeiferCosts = c.HeiferCosts / df

		backgroundingGrossCostsByYear[y] = c
	}
}

// Calculate cost to background each year
func calculateBackgroundingCostsByYear() {
	backgroundingGrossCostsByYear = make(map[int]backgroundingGrossCostsByYear_t)

	for _, calf := range animal.Records {

		yearBackgrounded := int(float64(calf.BirthDate)+205.+BackgroundDays) / 365

		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer {
				for _, a := range calf.AumWeanThruBackgrounding {
					//c := backgroundingGrossCostsByYear[a.Year]
					c := backgroundingGrossCostsByYear[yearBackgrounded]
					c.SteerCosts += a.Aum * BackgroundAumCost[a.MonthOfYear-1]
					//backgroundingGrossCostsByYear[a.Year] = c
					backgroundingGrossCostsByYear[yearBackgrounded] = c
				}
			} else if calf.Sex == animal.Heifer {
				for _, a := range calf.AumWeanThruBackgrounding {
					//c := backgroundingGrossCostsByYear[a.Year]
					c := backgroundingGrossCostsByYear[yearBackgrounded]
					c.HeiferCosts += a.Aum * BackgroundAumCost[a.MonthOfYear-1]
					//backgroundingGrossCostsByYear[a.Year] = c
					backgroundingGrossCostsByYear[yearBackgrounded] = c
				}
			}
		}
	}
	return
}
