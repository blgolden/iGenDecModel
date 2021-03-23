// simulateYears.go
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
	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	//"errors"
	"fmt"

	"github.com/blgolden/iGenDecModel/iGenDec/ecoIndex"
	"github.com/blgolden/iGenDecModel/iGenDec/logger"
	"github.com/blgolden/iGenDecModel/iGenDec/varStuff"

	"strconv"
	"strings"
)

func simulateYears() {

	burnInSimulation()

	burninMarker = len(animal.Records)

	if *animal.BumpComponent != "" {
		var c animal.BumpComponent_t
		s := strings.Split(*animal.BumpComponent, ",")
		if len(s) != 3 {
			logger.LogWriterFatal("-bumpComponent must have three values separated by a comma - e.g., 'WW,D,1")
		}
		c.TraitName = strings.TrimSpace(s[0])
		c.Component = strings.TrimSpace(s[1])
		c.Value, _ = strconv.ParseFloat(strings.TrimSpace(s[2]), 64)

		bumpComponent(c)

		varStuff.ConstrainedFactor(c)
	}

	animal.SimulateBase(param, varStuff.GvCholesky, varStuff.RvCholesky)
}

// Run the simulation for the burnin
func burnInSimulation() {
	if *logger.OutputMode == "verbose" {
		fmt.Printf("\n\tBeginning simulation for %v years\n", nYears)
	}

	for year := 1; year <= animal.Burnin; year++ {
		for _, h := range animal.Herds {
			//fmt.Println("LOC 1")
			animal.Breed(&h, year)
			//fmt.Println("LOC 2")
			animal.Calve(&h, year, varStuff.GvCholesky, varStuff.RvCholesky)
			//fmt.Println("LOC 3")
			animal.CullOpen(&h, year)
			//fmt.Println("LOC 4")
			animal.WriteCowAgeDistribution(animal.Herds, year-1, param)
			//fmt.Println("LOC 5")
			animal.CullOld(&h, year)
			//fmt.Println("LOC 6")
			animal.DetermineCowAum(&h, year)
			//fmt.Println("LOC 7")
		}
	}

	ecoIndex.SetActiveCowList()

	return
}

//Bump a trait's value by 1 unit
func bumpComponent(trait animal.BumpComponent_t) {

	var notInIndexList []int
	for i, c := range animal.ComponentList {
		//fmt.Println("LOC 0", i, c)
		if !ecoIndex.IsInIndex(c) {
			notInIndexList = append(notInIndexList, i)
		}
	}
	idx := animal.GeneticIndex(trait.TraitName, trait.Component)
	//fmt.Println("LOC 1", notInIndexList, idx)
	if idx < 0 {
		logger.LogWriterFatal("This component is not found in Components list: " + trait.TraitName + "," + trait.Component)
	}
	for _, h := range animal.Herds {

		for _, b := range h.Bulls {
			bv := b.BreedingValue.AtVec(idx) + trait.Value
			animal.Records[b.Id-1].BreedingValue.SetVec(idx, bv)
			for t := range notInIndexList {
				tVar := varStuff.VcMatrix["genetic"].At(idx, idx)
				tCov := varStuff.VcMatrix["genetic"].At(idx, t)
				bv := b.BreedingValue.AtVec(t) + trait.Value*tCov/tVar
				//fmt.Println("LOC 2", b.Id, bv, trait.Value, b.BreedingValue.AtVec(t), t, tVar, tCov)
				animal.Records[b.Id-1].BreedingValue.SetVec(t, bv)
			}
		}

	}

	return
}
