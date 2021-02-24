// weaning
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

	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/animal"
	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/logger"

	"gonum.org/v1/gonum/mat"
)

//var grossRevenueByYearWeanedCalves []float64 // Returns
var variableCostsByYearWeanedCalves []float64

var variableCostsByYearCows map[int]float64

type weaningGrossRevenueByYear_t struct {
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

type weaningGrossCostsByYear_t struct {
	SteerCosts            float64
	DiscountedSteerCosts  float64
	HeiferCosts           float64
	DiscountedHeiferCosts float64
}

var weaningGrossRevenueByYear map[int]weaningGrossRevenueByYear_t
var weaningGrossCostsByYear map[int]weaningGrossCostsByYear_t

type cullCowRevenueByYear_t struct {
	nCowsOpen            float64
	nCowsOld             float64
	CowRevenue           float64
	DiscountedCowRevenue float64
}

var cullCowGrosRevenueByYear map[int]cullCowRevenueByYear_t

// calculate the cull cow sales revenue

// Calculate animals total weaning sale revenue
func weaningSaleRevenue(calf animal.Animal) (salePrice float64) {

	weight, ok := animal.WeaningWtPhenotype(calf)

	if !ok {
		fmt.Println("Calf ", calf.Id, "can't get weaning weight. YearBorn:", calf.YearBorn)
	}

	min := float64(int(weight/100.) * 100)
	max := float64(int((weight+100.)/100) * 100)

	if calf.Sex == animal.Steer {
		if min == 800. {
			max = 9999.
		} else if max == 400. {
			min = 0.
		}
	} else if calf.Sex == animal.Heifer {
		if min == 700. {
			max = 9999.
		} else if max == 400. {
			min = 0.
		}
	}

	var tsmm TraitSexMinWtMaxWt_t
	tsmm.MaxWt = max
	tsmm.MinWt = min
	tsmm.Sex = calf.Sex
	tsmm.Trait = "WW"

	salePrice = weight * pricePerPound[tsmm]

	return salePrice
}

// Cost to raise a weanling calf by year
func calculateWeaningCostsByYear() {

	weaningGrossCostsByYear = make(map[int]weaningGrossCostsByYear_t)

	for _, calf := range animal.Records {

		/*if calf.YearBorn == 1 {
			calf.Dead = 0
		}*/

		yearWeaned := int(float64(calf.BirthDate)+205.) / 365
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer {

				for _, a := range calf.AumToWeaning {
					//c := weaningGrossCostsByYear[a.Year]
					c := weaningGrossCostsByYear[yearWeaned]
					c.SteerCosts += a.Aum * AumCost[a.MonthOfYear-1]
					weaningGrossCostsByYear[yearWeaned] = c
				}
			} else if calf.Sex == animal.Heifer {
				for _, a := range calf.AumToWeaning {
					c := weaningGrossCostsByYear[yearWeaned]
					c.HeiferCosts += a.Aum * AumCost[a.MonthOfYear-1]
					weaningGrossCostsByYear[yearWeaned] = c
				}
			}
		}
	}
	return
}

// Revenue from sale of weaning calves
func calculateWeaningRevenueByYear() {

	weaningGrossRevenueByYear = make(map[int]weaningGrossRevenueByYear_t)

	for _, calf := range animal.Records {

		yearWeaned := int(calf.BirthDate+205) / 365
		if calf.YearBorn >= 1 && calf.Dead == 0 {
			if calf.Sex == animal.Steer {

				w := weaningGrossRevenueByYear[yearWeaned]
				w.nSteers++
				w.SteerRevenue += weaningSaleRevenue(calf)
				p, _ := animal.Phenotype(calf, "WW")
				w.wtSteers += p
				weaningGrossRevenueByYear[yearWeaned] = w

			} else if calf.Sex == animal.Heifer { // This works because sex has been set to Cow if she became a replacement

				w := weaningGrossRevenueByYear[yearWeaned]
				w.nHeifers++
				w.HeiferRevenue += weaningSaleRevenue(calf)
				p, _ := animal.Phenotype(calf, "WW")
				w.wtHeifers += p
				weaningGrossRevenueByYear[yearWeaned] = w
			}
		} else {
			w := weaningGrossRevenueByYear[yearWeaned]
			w.nDead++
			weaningGrossRevenueByYear[yearWeaned] = w
		}
	}
	return
}

// Determine the discounted value of the revenue
func calculateDiscountedWeaningRevenueByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		w := weaningGrossRevenueByYear[y]

		df := math.Pow(1.+DiscountRate, period)

		w.DiscountedSteerRevenue = w.SteerRevenue / df
		w.DiscountedHeiferRevenue = w.HeiferRevenue / df

		weaningGrossRevenueByYear[y] = w
	}
}

// Determine the discounted value of the costs
func calculateDiscountedWeaningCostsByYear(nYears int) {
	for y := animal.Burnin + 1; y <= nYears; y++ {

		period := float64(y - StartYearOfNetReturns)

		df := math.Pow(1.+DiscountRate, period)

		c := weaningGrossCostsByYear[y]

		c.DiscountedSteerCosts = c.SteerCosts / df
		c.DiscountedHeiferCosts = c.HeiferCosts / df

		weaningGrossCostsByYear[y] = c
	}
}

// This is used to caalculate the weaning costs per year for
// all indexes except weaning, because weaning does it below in the
// net returns calculation.
func WeaningVariableCostsPerMating(nYears int) float64 {

	var TotalDiscountedCosts float64
	var TotalDiscountedCostsPerMating float64

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nDiscounted Costs of Weanling Calf:")
		fmt.Println("       Costs_______________________ ")
		fmt.Println("Year    $ Actual     $ Discounted  $ cum/Exposure  N Cows Exposed")
	}

	for y := StartYearOfNetReturns; y <= nYears; y++ {

		TotalDiscountedCosts += weaningGrossCostsByYear[y].DiscountedSteerCosts + weaningGrossCostsByYear[y].DiscountedHeiferCosts

		TotalDiscountedCostsPerMating += (weaningGrossCostsByYear[y].DiscountedSteerCosts + weaningGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d  %10.2f    %10.2f     %10.2f       %7d\n", y,
				weaningGrossCostsByYear[y].SteerCosts+weaningGrossCostsByYear[y].HeiferCosts,
				weaningGrossCostsByYear[y].DiscountedSteerCosts+weaningGrossCostsByYear[y].DiscountedHeiferCosts,
				TotalDiscountedCostsPerMating,
				animal.CowsExposedPerYear[y])
		}

	}

	return TotalDiscountedCostsPerMating / float64(nYears-StartYearOfNetReturns+1)
}

//  The total discounted net returns to fixed costs for weaning
// and optionally write a table to stdout
// Returns average cost/mating
func WeaningNetReturnsToFixedCosts(nYears int) float64 {

	var TotalDiscountedRevenue float64
	var TotalDiscountedNetRevenuePerMating float64

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nReturns and Costs from Weanling Calf Sales:")
		fmt.Println("       Returns_____________________  Costs_______________________ ")
		fmt.Println("Year    $ Actual     $ Discounted    $ Actual     $ Discounted  $ Net/Exposure  N Cows Exposed")
	}

	for y := StartYearOfNetReturns; y <= nYears; y++ {

		TotalDiscountedRevenue += weaningGrossRevenueByYear[y].DiscountedSteerRevenue + weaningGrossRevenueByYear[y].DiscountedHeiferRevenue

		netPerExposure := (weaningGrossRevenueByYear[y].DiscountedSteerRevenue + weaningGrossRevenueByYear[y].DiscountedHeiferRevenue -
			weaningGrossCostsByYear[y].DiscountedSteerCosts - weaningGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])

		TotalDiscountedNetRevenuePerMating += netPerExposure

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d  %10.2f    %10.2f     %10.2f   %10.2f         %10.2f       %7d\n", y,
				weaningGrossRevenueByYear[y].SteerRevenue+weaningGrossRevenueByYear[y].HeiferRevenue,
				weaningGrossRevenueByYear[y].DiscountedSteerRevenue+weaningGrossRevenueByYear[y].DiscountedHeiferRevenue,
				weaningGrossCostsByYear[y].SteerCosts+weaningGrossCostsByYear[y].HeiferCosts,
				weaningGrossCostsByYear[y].DiscountedSteerCosts+weaningGrossCostsByYear[y].DiscountedHeiferCosts,
				netPerExposure,
				animal.CowsExposedPerYear[y])
		}

	}

	return TotalDiscountedNetRevenuePerMating / float64(nYears-StartYearOfNetReturns+1)
}

// Value calves at weaning
func weaningSale(nYears int) float64 {

	if *logger.OutputMode == "verbose" {
		fmt.Println("Processing weaning sale net returns...")
	}

	calculateWeaningRevenueByYear()
	calculateDiscountedWeaningRevenueByYear(nYears)
	calculateWeaningCostsByYear()
	calculateDiscountedWeaningCostsByYear(nYears)

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nRevenue from weaning calf sales:")
		fmt.Println("Sale Year    n Steers	Steer $     Wt Steers   n Heifers      Heifer $    Wt Heifers    n Dead")
		for year := 1; year <= nYears; year++ {
			fmt.Printf("  %5d      %5d %12.2f  %12.1f       %5d  %12.2f  %12.1f   %7d\n", year, int(weaningGrossRevenueByYear[year].nSteers),
				weaningGrossRevenueByYear[year].SteerRevenue, weaningGrossRevenueByYear[year].wtSteers,
				int(weaningGrossRevenueByYear[year].nHeifers), weaningGrossRevenueByYear[year].HeiferRevenue, weaningGrossRevenueByYear[year].wtHeifers,
				weaningGrossRevenueByYear[year].nDead)
		}
	}

	return WeaningNetReturnsToFixedCosts(nYears) // This returns total accumulated net returns/mating

}

// Determine the net revenue from cull cow sale
func cullSale(nYears int) float64 {

	cullCowGrosRevenueByYear = make(map[int]cullCowRevenueByYear_t)

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nRevenue from cull cow sales:")
		fmt.Println("Year   Tot Weight  $ Revenue $ DiscountedRev  N CullsOpen N CullsOld")
	}

	var TotalDiscountedNetRevenuePerMating float64

	for y := StartYearOfNetReturns; y <= nYears; y++ {

		var tsmm TraitSexMinWtMaxWt_t
		tsmm.MaxWt = 9999.0
		tsmm.MinWt = 0.0
		tsmm.Sex = animal.Cow
		tsmm.Trait = "MW"

		c := cullCowGrosRevenueByYear[y]

		c.CowRevenue = animal.WtCullCows[y].CumWt * pricePerPound[tsmm]
		c.nCowsOpen = animal.WtCullCows[y].NheadOpen
		c.nCowsOld = animal.WtCullCows[y].NheadOld
		period := float64(y - StartYearOfNetReturns)
		c.DiscountedCowRevenue = c.CowRevenue / math.Pow(1.+DiscountRate, period)

		cullCowGrosRevenueByYear[y] = c

		TotalDiscountedNetRevenuePerMating += c.DiscountedCowRevenue / float64(animal.CowsExposedPerYear[y])

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d  %10.0f %10.2f      %10.2f   %10.0f %10.0f\n", y, animal.WtCullCows[y].CumWt, c.CowRevenue, c.DiscountedCowRevenue, c.nCowsOpen, c.nCowsOld)
		}

	}

	return TotalDiscountedNetRevenuePerMating / float64(nYears-StartYearOfNetReturns+1)

}

// Cost per year for cows
func CowCosts(nYears int) float64 {

	variableCostsByYearCows = make(map[int]float64)

	for _, c := range animal.Records {
		if c.Sex == animal.Cow && c.YearBorn > 0 {
			for _, a := range c.CowAum {
				if a.Year >= StartYearOfNetReturns {
					variableCostsByYearCows[a.Year] += a.Aum * AumCost[a.MonthOfYear-1]
				}
			}
		}
	}

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nCow costs:")
		fmt.Println("Year        $ Cost  $ Discounted          $ Net/Exp")
	}
	var cumDc float64
	for y := StartYearOfNetReturns; y <= nYears; y++ {
		period := float64(y - StartYearOfNetReturns)
		df := math.Pow(1.+DiscountRate, period)
		dr := variableCostsByYearCows[y] / df

		netPerExposure := dr / float64(animal.CowsExposedPerYear[y])
		cumDc += netPerExposure

		if *logger.OutputMode == "verbose" {
			fmt.Printf("%5d     %10.2f  %10.2f        %10.2f\n", y, variableCostsByYearCows[y], dr, netPerExposure)
		}
	}

	return cumDc / float64(nYears-StartYearOfNetReturns+1)
}

// Process an index with sale at weaning
func EvaluateWeaningIndex(nYears int, burninMarker int, param map[string]interface{}, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) float64 {

	var base animal.Component_t
	base.Component = "D"
	base.TraitName = "base"
	IndexNetReturns := weaningSale(nYears) // Discounted and per mating
	IndexNetReturns += cullSale(nYears)    // Discounted and per mating
	IndexNetReturns -= CowCosts(nYears)

	if *logger.OutputMode == "verbose" {

		fmt.Printf("\nPlanning Horizon (in years):                                            %12d\n", nYears-StartYearOfNetReturns+1)
		//fmt.Printf("%d Year Discounted Net Returns to land, management and labor per exp accum: %12.2f\n", nYears-StartYearOfNetReturns+1, NetReturns)
		fmt.Printf("%d year Discounted Net Returns to land, management and labor per exposure:  %12.2f\n", nYears-StartYearOfNetReturns+1,
			IndexNetReturns)
		//} else {

		//}
		fmt.Println("NOTE: all net values are returns to land, management and labor")
	}

	return IndexNetReturns
}

// Average weaning costs per mating accross years of simulation
// Not used by weaning index but used by bg, fat cattle, and grade and yield
func calculateWeaningCostsPerMating(nYears int) float64 {

	var totalDiscountedWeaningCostPerMating float64

	for y := StartYearOfNetReturns; y <= nYears; y++ {
		totalDiscountedWeaningCostPerMating += (weaningGrossCostsByYear[y].DiscountedSteerCosts + weaningGrossCostsByYear[y].DiscountedHeiferCosts) /
			float64(animal.CowsExposedPerYear[y])
	}
	return totalDiscountedWeaningCostPerMating / float64(nYears-StartYearOfNetReturns+1)
}
