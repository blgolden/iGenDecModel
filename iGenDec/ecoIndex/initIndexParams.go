// initIndexParams
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
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	hjson "github.com/hjson/hjson-go"

	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	"github.com/blgolden/iGenDecModel/iGenDec/logger"

	"gonum.org/v1/gonum/mat"
)

var IndexType string
var IndexTerminal bool

// setup the map of the array of json name:value pairs - notice "interface{}"
var ParamIndex map[string]interface{}

var gridPrice map[GridValue_t]float64 // Premiums for grid prices
var StartYearOfNetReturns int
var DiscountRate float64
var IndexComponents []animal.Component_t
var AumCost []float64
var BackgroundAumCost []float64
var FeedlotFeedCost float64
var InProgramProportion float64 // proportion of calves that initially may qualify for a grid program - e.g., CHB, CAB
var NetReturns float64

type TraitSexMinWtMaxWt_t struct {
	Trait         string  // Same as traits in master hjson - e.g., WW is weaning weight
	Sex           string  // Sex S=steer and H=heifer calf
	MinWt         float64 //If the weight of the animals is >=
	MaxWt         float64 // and the weight of the animals is <
	PricePerPound float64 // read as per cwt but converted to per lb
}

var PriceTable []TraitSexMinWtMaxWt_t

type GridValue_t struct {
	QualityGrade string // Prime, Program, Choice, Select,Standard
	YieldGrade   int    // 1, 2, 3, 4, 5
}

// Read the table of price per cwt.  Convert it to price per pound
func readPricePerPound() {

	carray, ok := ParamIndex["traitSexPricePerCwt"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'pricePerPound:' key not found in echonomic index hjson.")
	}

	for i := range carray {
		c := strings.Split(carray[i].(string), ",")
		trait := strings.TrimSpace(c[0])
		sex := strings.TrimSpace(c[1])
		min, _ := strconv.ParseFloat(strings.TrimSpace(c[2]), 64)
		max, _ := strconv.ParseFloat(strings.TrimSpace(c[3]), 64)

		var tsmm TraitSexMinWtMaxWt_t
		tsmm.Trait = trait
		tsmm.Sex = sex
		tsmm.MinWt = min
		tsmm.MaxWt = max
		f, _ := strconv.ParseFloat(strings.TrimSpace(c[4]), 64)
		tsmm.PricePerPound = f / 100.0 // Convert from $/cwt to $/lb

		PriceTable = append(PriceTable, tsmm)
	}

}

// If slaughtercattle IndexType then initialize grid pricing
func InitGrid() {

	gridPrice = make(map[GridValue_t]float64)

	carray, ok := ParamIndex["gridPremiums"].([]interface{})

	for i := range carray {
		c := strings.Split(carray[i].(string), ",")
		qg := strings.TrimSpace(c[0])
		for yieldGrade := 1; yieldGrade <= 5; yieldGrade++ {

			var gridValue GridValue_t
			gridValue.QualityGrade = qg
			gridValue.YieldGrade = yieldGrade

			f, _ := strconv.ParseFloat(strings.TrimSpace(c[yieldGrade]), 64)
			gridPrice[gridValue] = f / 100.0 // comes in as $/cwt and converts to $/lb
		}
	}
	ip, ok := ParamIndex["proportionInProgram"].(interface{})
	if ok {
		InProgramProportion, _ = strconv.ParseFloat(strings.TrimSpace(ip.(string)), 64)
	}
}

// Read the index hjson file
func InitIndexParams(indexParam *string) {

	hjsonFile, err := os.Open(*indexParam)
	if err != nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("Failed to open " + *indexParam)
			fmt.Println(err)
		}
		logger.LogWriterFatal("Failed to open index parameter file " + *indexParam)
	}

	defer hjsonFile.Close()

	byteValue, _ := ioutil.ReadAll(hjsonFile)

	if er := hjson.Unmarshal(byteValue, &ParamIndex); er != nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("Could not process the hjson", *indexParam)
			fmt.Println("Panic message:", er)
		}
		logger.LogWriterFatal("failed to unmarshal " + *indexParam)
	}

	readPricePerPound()
	loadAumCostPerMonth()

	return
}

// Read in the AUM cost per month
func loadAumCostPerMonth() {

	carray, ok := ParamIndex["aumCost"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'aumCost:' key not found in economic index hjson.")
	}

	for i := range carray {
		c := carray[i].(float64)
		AumCost = append(AumCost, c)
	}

	if string(ParamIndex["saleEndpoint"].(string)) != "weaning" {

		barray, ok := ParamIndex["backgroundAumCost"].([]interface{})
		if !ok {
			logger.LogWriterFatal("'backgroundAumCost not found in economic index hjson")
		}
		for i := range barray {
			b := barray[i].(float64)
			BackgroundAumCost = append(BackgroundAumCost, b)
		}

		if string(ParamIndex["saleEndpoint"].(string)) != "background" { // Then it must be fatcattle or slaughter
			dr, ok := ParamIndex["feedlotFeedCost"].(interface{})
			if !ok {
				logger.LogWriterFatal("'feedlotFeedCost' key not found in economic index hjson")
			}
			r, _ := strconv.ParseFloat(strings.TrimSpace(dr.(string)), 64)
			FeedlotFeedCost = r

		}
	}

	return
}

// Set the discount rate
func whatDiscountRate() float64 {
	dr := ParamIndex["discountRate"].(interface{})
	r, _ := strconv.ParseFloat(strings.TrimSpace(dr.(string)), 64)

	return r
}

/*
// Reset Records back to heifers
func HeiferReset() {
	for i := range animal.HeiferResetList {
		animal.Records[i].Sex = animal.Heifer
		animal.Records[i].BreedingRecords = nil
		animal.Records[i].Dead = 0
		animal.Records[i].YearCowCulled = 0
		animal.Records[i].Active = false

	}
}

// Reset Records back to active cows
func CowReset() {
	for l := range animal.CowResetList {
		i := animal.CowResetList[l].Id - 1
		animal.Records[i].Sex = animal.Cow
		animal.Records[i].BreedingRecords = nil
		animal.Records[i].Dead = 0
		animal.Records[i].YearCowCulled = 0
		animal.Records[i].Active = true
		animal.Records[i].BreedingRecords = animal.CowResetList[l].BreedingRecords
	}
}
*/
func SetActiveCowList() {
	for _, h := range animal.Herds {
		activeCows := animal.ActiveCows(&h)
		for i := range activeCows {
			animal.CowResetList = append(animal.CowResetList, *activeCows[i])
		}
	}
}

// So they are sold when a terminal index
func CowsToHeifers() {
	for r := range animal.Records {
		if animal.Records[r].Sex == animal.Cow {
			animal.Records[r].Sex = animal.Heifer
		}
	}
}

// main call
func ProcessNetReturns(indexParam *string, nYears int, burninMarker int,
	param map[string]interface{}, gvCholesky mat.Cholesky, rvCholesky mat.Cholesky) {
	//InitIndexParams(indexParam)

	IndexType = WhatSaleEndpoint()
	IndexTerminal = IsIndexTerminal()
	StartYearOfNetReturns = animal.Burnin + 1
	DiscountRate = whatDiscountRate()

	if *logger.OutputMode == "verbose" {
		fmt.Println("Type of economic index:", IndexType, " Terminal:", IndexTerminal)
	}

	if IndexTerminal {
		CowsToHeifers()
		StartYearOfNetReturns = nYears
	}

	switch IndexType {
	case "weaning":
		NetReturns = EvaluateWeaningIndex(nYears, burninMarker, param, gvCholesky, rvCholesky)

	case "background":
		NetReturns = EvaluateBackgroundingIndex(nYears, burninMarker, param, gvCholesky, rvCholesky)

	case "fatcattle":
		NetReturns = EvaluateFatCattleIndex(nYears, burninMarker, param, gvCholesky, rvCholesky)

	case "slaughtercattle":
		NetReturns = EvaluateSlaughterCattleIndex(nYears, burninMarker, param, gvCholesky, rvCholesky)
	}

	if *logger.OutputMode != "verbose" {
		fmt.Printf("%f", NetReturns)
	}
}

// Return the type of index the hjson builds
func WhatSaleEndpoint() (indexType string) {

	indexType = string(ParamIndex["saleEndpoint"].(string))
	if indexType == "" {
		logger.LogWriterFatal("'IndexType:' key not found in ")
	}

	return indexType
}
func IsIndexTerminal() bool {
	indexTerminal := ParamIndex["indexTerminal"].(bool)
	return indexTerminal
}

// Setup the index traits
func LoadIndexComponents() {

	carray, ok := ParamIndex["indexComponents"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'indexComponents:' key not found")
	}
	for i := range carray {

		c := strings.Split(carray[i].(string), ",")
		var e animal.Component_t

		e.TraitName = strings.TrimSpace(c[0])
		e.Component = strings.TrimSpace(c[1])

		IndexComponents = append(IndexComponents, e)

	}
}

// Is this in the index
func IsInIndex(b animal.Component_t) bool {
	for _, c := range IndexComponents {
		if b.TraitName == c.TraitName && b.Component == c.Component {
			return true
		}
	}
	return false
}
