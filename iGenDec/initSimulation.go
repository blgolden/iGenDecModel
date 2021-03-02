// initSimulation
package main
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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	"github.com/blgolden/iGenDecModel/iGenDec/ecoIndex"

	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	//	"time"

	"github.com/blgolden/iGenDecModel/iGenDec/logger"
	"github.com/blgolden/iGenDecModel/iGenDec/varStuff"

	"gonum.org/v1/gonum/mat"

	hjson "github.com/hjson/hjson-go"
)

// setup the map of the array of json name:value pairs - notice "interface{}"
var param map[string]interface{}

var paramFile *string // Name of the parameter file
var indexParm *string // Name of the parameter file for configuring the index

var nYears int

// Initialize the simulation
func initSimulation() {

	parseArgs()

	loadParam()

	initSexCodes()

	if *logger.OutputMode == "verbose" {

		runComment, ok := param["Comment"].(interface{})
		if ok {
			fmt.Printf("Comment: %v\n\n", runComment.(string))
		}
	}

	// Trait list
	if *logger.OutputMode == "verbose" {
		fmt.Print("Traits:")
	}

	animal.TraitMean = make(map[string]float64)

	// Process the Traits: key from master.hjson and set the means as a mapped slice
	tarray, ok := param["Traits"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'Traits:' key not found in " + *paramFile)
	}
	for i := range tarray {
		s := strings.Split(tarray[i].(string), ",")

		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%v %v\n", s[0], s[1])
		}
		animal.Traits = append(animal.Traits, strings.TrimSpace(s[0]))
		animal.TraitMean[animal.Traits[i]], _ = strconv.ParseFloat(strings.TrimSpace(s[1]), 64)
	}

	// Genetic components list
	if *logger.OutputMode == "verbose" {
		fmt.Print("Genetic Components:")
	}
	carray, ok := param["Components"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'Components:' key not found in " + *paramFile)
	}
	for i := range carray {
		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%v\n", carray[i].(string))
		}
		animal.Components = append(animal.Components, strings.TrimSpace(carray[i].(string)))
		v := strings.TrimSpace(carray[i].(string))
		splt := strings.Split(v, ",")
		var c animal.Component_t

		c.TraitName = strings.TrimSpace(splt[0])
		c.Component = strings.TrimSpace(splt[1]) // comp = D or M
		animal.ComponentList = append(animal.ComponentList, c)
	}

	varStuff.Vc = make(map[string]varStuff.Matvec_t)
	varStuff.VcMatrix = make(map[string]mat.Symmetric)
	varStuff.GvCholesky = varStuff.DecompVar("genetic", param)
	varStuff.RvCholesky = varStuff.DecompVar("residual", param)

	// Set the rnd seed
	animal.Rng = rand.New(rand.NewSource(*logger.Seed))
	//fmt.Println(*logger.Seed)
	//var seed int64 = 1234 // THIS IS FOR TESTING - CONSTANT SEED
	//animal.Rng.Seed(*logger.Seed) // THIS CAN BE REMOVED IN PRODUCTION

	// Breeding season parameters
	array, ok := param["herds"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'herds:' key not found in " + *paramFile)
	}
	nHerds := len(array)
	if *logger.OutputMode == "verbose" {
		fmt.Printf("\nNumber of herds: %d\n", nHerds)
	}
	if *indexParm != "" {
		ecoIndex.InitIndexParams(indexParm)
		ecoIndex.LoadIndexComponents()
		if ecoIndex.WhatSaleEndpoint() != "weaning" {
			var ok bool
			ecoIndex.BackgroundDays, ok = ecoIndex.ParamIndex["backgroundDays"].(float64)
			if !ok {
				logger.LogWriterFatal("'backgroundDays' not found in index parameter file.")
			}
			if ecoIndex.BackgroundDays <= 0 {
				ecoIndex.BackgroundDays = 1 // In case these are calf fed we need at least 1 day to get the feedlot in weight
			}
			animal.BackgroundDays = ecoIndex.BackgroundDays
		}
		animal.IndexType = ecoIndex.WhatSaleEndpoint()
		if animal.IndexType == "fatcattle" || animal.IndexType == "slaughtercattle" {
			var ok bool
			if animal.DaysOnFeed, ok = ecoIndex.ParamIndex["daysOnFeed"].(float64); !ok {
				logger.LogWriterFatal("daysOnFeed key not found in fat cattle economic index file.  Error at initSimulation")
			}
		}
		if animal.IndexType == "slaughtercattle" {
			ecoIndex.InitGrid()
		}
	}

	animal.IndexTerminal = ecoIndex.IsIndexTerminal()
	if ecoIndex.IsIndexTerminal() { // Terminal indexes do not want cow herd ages to change

		if ecoIndex.WhatSaleEndpoint() != "weaning" {
			animal.Burnin = 1
			animal.YearsPlanningHorizon = 2
		} else {
			animal.Burnin = 1
			animal.YearsPlanningHorizon = 1
		}
	} else {
		animal.Burnin = int(param["burnin"].(float64))
		animal.YearsPlanningHorizon = int(param["planningHorizon"].(float64))
	}
	nYears = animal.Burnin + animal.YearsPlanningHorizon
	//}

	animal.CalfAumAt500, _ = param["calfAum"].(float64)
	animal.CowAumAt1000, _ = param["cowAum"].(float64)

	animal.Herds = make(map[string]animal.Herd)
	for i := range array {
		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%v\n", array[i].(string))
		}
		s := strings.Split(array[i].(string), ",")
		var thisHerd animal.Herd // Need to build in multiple herds
		thisHerd.HerdName = strings.TrimSpace(s[0])
		d, _ := strconv.Atoi(strings.TrimSpace(s[1]))
		thisHerd.NumberCows = d
		d, _ = strconv.Atoi(strings.TrimSpace(s[2]))
		thisHerd.StartBreeding = animal.Date(d)
		d, _ = strconv.Atoi(strings.TrimSpace(s[3]))
		thisHerd.BreedingSeasonLen = animal.Date(d)
		f, _ := strconv.ParseFloat(strings.TrimSpace(s[4]), 64)
		thisHerd.CowConceptionRate = conceptionPerCycle(int64(thisHerd.BreedingSeasonLen), f)
		f, _ = strconv.ParseFloat(strings.TrimSpace(s[5]), 64)
		thisHerd.InitialCalvingDeathLessRate = f

		thisHerd.NBorn = make([]float64, nYears+1+2)
		thisHerd.SumBirthDates = make([]float64, nYears+1+2) // 2 extra years of simulation after planning horizon to get heifers out, etc

		animal.Herds[thisHerd.HerdName] = thisHerd

	}

	if *logger.OutputMode == "verbose" {
		fmt.Printf("\n\tFinished loading %v...\n\n", *paramFile)
	}

	loadBreedEffects()
	loadHeterosis()
	loadCowHerdBreedComposition()
	loadBullBatteryBreedComposition()
	loadCurrentCalvesBreedComposition() // What the trait means are made from for calf traits
	loadBreedTraitSexAod()
	loadTraitAgeEffects()
	loadFoundationBullsMerit()

	adjustBreedEffects()

	animal.WtCullCows = make(map[int]animal.Sales_t) // Accumulate the weight of cull cows
	animal.NHeifersBred = make(map[int]int)

	openOutputFiles()

	initializeTables()

	// Initialize the CD distribution
	dvar := varStuff.VarFromMatrix(animal.GeneticIndex("CD", "D"), varStuff.VcMatrix["genetic"])
	mvar := varStuff.VarFromMatrix(animal.GeneticIndex("CD", "M"), varStuff.VcMatrix["genetic"])
	rvar := varStuff.VarFromMatrix(animal.ResidualIndex("CD"), varStuff.VcMatrix["residual"])
	animal.CDVar = dvar + mvar + rvar
	return
}

// Adjust the breed effects for the composition of the current calves
// such that the current calves average effect is zero.  This is done
// so that the mean input are the current calves, and cows effects
// Because the input values deviate from Angus (or whatever is zeroed)
func adjustBreedEffects() {

	for component, breedEffects := range animal.Breeds {
		if breedEffects.CowOrCalf == "Calf" {
			var adj float64
			for _, c := range animal.CurrentCalvesBreedCompositionTable {
				p := c.Proportion
				for breed, bp := range c.BreedProportions {
					adj += p * bp * breedEffects.Effects[breed]
				}
			}

			for breed, eff := range breedEffects.Effects {
				breedEffects.Effects[breed] = eff - adj
			}
			animal.Breeds[component] = breedEffects
		} else { // Is cow trait
			var adj float64
			for _, c := range animal.FoundationCowHerdBreedCompositionTable {
				p := c.Proportion
				for breed, bp := range c.BreedProportions {
					adj += p * bp * breedEffects.Effects[breed]
				}
			}

			for breed, eff := range breedEffects.Effects {
				breedEffects.Effects[breed] = eff - adj
			}
			animal.Breeds[component] = breedEffects
		}
	}
	//fmt.Println("LOC 1", animal.Breeds)
	return
}

// Initialize summary talbes
func initializeTables() {

	// table to summarize the breeding records
	animal.BreedingRecordsYearTable = make(map[animal.HerdYear_t]animal.BreedingRecordsTable_t)

	animal.CowsExposedPerYear = make(map[int]int)

	return
}

func initSexCodes() {
	animal.SexCodes = []string{animal.Bull, animal.Heifer, animal.Cow, animal.Steer}
}

// Read the TraitAgeEffects from the master hjson
// Trait name and slope in units/day of age
func loadTraitAgeEffects() {

	animal.TraitAgeEffects = make(map[string]animal.InterceptSlope_t)

	carray, ok := param["TraitAgeEffects"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'TraitAgeEffects:' key not found in " + *paramFile)
	}
	for i := range carray {
		c := strings.Split(carray[i].(string), ",")
		trait := strings.TrimSpace(c[0])
		f, _ := strconv.ParseFloat(strings.TrimSpace(c[1]), 64)
		v, _ := strconv.ParseFloat(strings.TrimSpace(c[2]), 64)

		var tis animal.InterceptSlope_t
		tis.Slope = f
		tis.Age = v

		animal.TraitAgeEffects[trait] = tis
	}

	return
}

// Read the BreedTraitSexAod factors
// from the hjson file
func loadBreedTraitSexAod() {

	animal.BreedTraitSexAod = make(map[animal.BTS_t]float64)

	// Load a table of the breed, trait, sex, AOD factors
	carray, ok := param["BreedTraitSexAod"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'BreedTraitSexAod:' key not found in " + *paramFile)
	}
	for i := range carray {
		c := strings.Split(carray[i].(string), ",")
		breed := strings.TrimSpace(c[0])
		trait := strings.TrimSpace(c[1])
		sex := strings.TrimSpace(c[2])

		for k := 3; k < len(c); k++ {

			f, _ := strconv.ParseFloat(strings.TrimSpace(c[k]), 64)

			var b animal.BTS_t
			b.Breed = breed
			b.Trait = trait
			b.Sex = sex
			b.Aod = k - 3

			animal.BreedTraitSexAod[b] = f

		}
	}

	//fmt.Println(animal.BreedTraitSexAod)
}

// Read in the HeterosisCodes: of breed-class
// Designate the HeterosisCodes: for each breed
func loadHeterosis() {

	animal.HeterosisCodes = make(map[string]string)

	// Load a table of the breed to breed classification codes
	carray, ok := param["HeterosisCodes"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'HeterosisCodes:' key not found in " + *paramFile)
	}
	for i := range carray {
		c := strings.Split(carray[i].(string), ",")
		breed := strings.TrimSpace(c[0])
		code := strings.TrimSpace(c[1])
		animal.HeterosisCodes[breed] = code
	}

	if *logger.OutputMode == "verbose" {
		fmt.Println("Heterosis Codes: ", animal.HeterosisCodes)
	}

	// Load the trait by breed classification cross F1 values
	varray, ok := param["HeterosisValues"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'HeterosisValues:' key not found in " + *paramFile)
	}
	b := strings.Split(varray[0].(string), ",") // 1st line is the codes
	for i := 2; i < len(b); i++ {
		animal.HeterosisCrossClasses = append(animal.HeterosisCrossClasses, strings.TrimSpace(b[i]))
	}

	if *logger.OutputMode == "verbose" {
		fmt.Println("Heterosis Cross Classes: ", animal.HeterosisCrossClasses)
	}

	animal.HeterosisValues = make(map[animal.Component_t]animal.HeterosisValues_t, len(varray)-1) // 1st row is a header

	// Load the table of values for each trait-component
	for j := 1; j < len(varray); j++ {
		var a animal.Component_t

		v := strings.Split(varray[j].(string), ",")

		a.TraitName = strings.TrimSpace(v[0])
		a.Component = strings.TrimSpace(v[1])

		h := NewHeterosisValue(a.TraitName, a.Component)

		for k := 2; k < len(v); k++ {
			f, _ := strconv.ParseFloat(strings.TrimSpace(v[k]), 64)
			h.Values[animal.HeterosisCrossClasses[k-2]] = f
		}

		animal.HeterosisValues[a] = h
	}
	return
}

// Do this so we can map the Values
func NewHeterosisValue(traitName string, component string) animal.HeterosisValues_t {
	var h animal.HeterosisValues_t
	h.TraitName = traitName
	h.Component = component
	h.Values = make(map[string]float64)
	return h
}

// Load the merit of the foundation bulls
func loadFoundationBullsMerit() {
	array, ok := param["meritFoundationBulls"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'meritFoundationBulls:' key not found in " + *paramFile)
	}

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nInitial Bulls merit: ")
	}
	for i := 0; i < len(array); i = i + 2 {

		f := array[i].(float64)
		animal.BullMerit = append(animal.BullMerit, f)
	}

}

// Read the breed effects table from the master.hjson file
// And map according to trait name
func loadBreedEffects() {

	array, ok := param["BreedEffects"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'BreedEffects:' key not found in " + *paramFile)
	}
	// Make the list of breed names from the 1st row of the BreedEffects: hjson key
	b := strings.Split(array[0].(string), ",")
	for i := 3; i < len(b); i++ { // Starting at 3 moves past "Trait,Effect,Type" in the header
		animal.BreedsList = append(animal.BreedsList, strings.TrimSpace(b[i]))
	}

	animal.Breeds = make(map[animal.Component_t]animal.BreedEffects_t)

	// Read the trait by breed effects values
	for i := 1; i < len(array); i++ {

		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%v\n", array[i].(string))
		}
		s := strings.Split(array[i].(string), ",")

		var c animal.Component_t
		c.TraitName = strings.TrimSpace(s[0])
		c.Component = strings.TrimSpace(s[1])

		var b animal.BreedEffects_t
		b.TraitName = strings.TrimSpace(s[0])
		b.Component = strings.TrimSpace(s[1])
		b.CowOrCalf = strings.TrimSpace(s[2])

		b.Effects = make(map[string]float64)

		for j := 3; j < len(s); j++ { // The last column is the overall mean they deviate from so -1
			f, _ := strconv.ParseFloat(strings.TrimSpace(s[j]), 64)

			b.Effects[animal.BreedsList[j-3]] = f
		}

		animal.Breeds[c] = b

		//fmt.Println(c, animal.Breeds[c])
	}
	//os.Exit(1)
	return
}

// Read in the cows herd breed composition from COwHerdBreedComposition: key in master.hjson
func loadCowHerdBreedComposition() {

	array, ok := param["CowHerdBreedComposition"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'CowHerdBreedComposition:' key not found in " + *paramFile)
	}
	//fmt.Println(array)
	// Read the breed compositions
	var top float64
	top = 0.0

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nInitial cow herd breed composition: ")
	}

	for i := 0; i < len(array); i = i + 2 {

		f := array[i].(float64)
		top += f / 100.

		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%3.0f%% of cows are:\n", f)
		}
		var b animal.BreedComposition_t
		b.Proportion = top
		b.BreedProportions = make(map[string]float64)
		s := strings.Split(array[i+1].(string), ",")
		for j := 0; j < len(s); j = j + 2 {
			breed := strings.TrimSpace(s[j])
			f, _ := strconv.ParseFloat(strings.TrimSpace(s[j+1]), 64)
			proportion := f / 100.
			b.BreedProportions[breed] = proportion
			if *logger.OutputMode == "verbose" {
				fmt.Printf("\t\t%-8s %3.0f%%\n", breed, f)
			}
		}
		animal.FoundationCowHerdBreedCompositionTable = append(animal.FoundationCowHerdBreedCompositionTable, b)
	}

	//fmt.Println("LOC F", animal.FoundationCowHerdBreedCompositionTable)
	return
}

// Read in the bull battery breed composition from BullBatteryBreedComposition: key in master.hjson
func loadBullBatteryBreedComposition() {

	array, ok := param["BullBatteryBreedComposition"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'BullBatteryBreedComposition:' key not found in " + *paramFile)
	}
	//fmt.Println(array)
	// Read the breed compositions
	var top float64
	top = 0.0

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nInitial bull battery breed composition: ")
	}
	for i := 0; i < len(array); i = i + 2 {

		f := array[i].(float64)
		top += f / 100.

		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%3.0f%% of bulls are:\n", f)
		}
		var b animal.BreedComposition_t
		b.Proportion = top
		b.BreedProportions = make(map[string]float64)
		s := strings.Split(array[i+1].(string), ",")
		for j := 0; j < len(s); j = j + 2 {
			breed := strings.TrimSpace(s[j])
			f, _ := strconv.ParseFloat(strings.TrimSpace(s[j+1]), 64)
			proportion := f / 100.
			b.BreedProportions[breed] = proportion
			if *logger.OutputMode == "verbose" {
				fmt.Printf("\t\t%-8s %3.0f%%\n", breed, f)
			}
		}
		animal.BullBatteryBreedCompositionTable = append(animal.BullBatteryBreedCompositionTable, b)
	}

	return
}

// Read in the current calf crop breed composition from CurrentCalvessBreedComposition: key in master.hjson
// This is used to adjust the breed effect and heterosis effects for calf traits.
func loadCurrentCalvesBreedComposition() {

	array, ok := param["CurrentCalvesBreedComposition"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'CurrentCalvesBreedComposition:' key not found in " + *paramFile)
	}
	//fmt.Println(array)
	// Read the breed compositions
	var top float64
	top = 0.0

	if *logger.OutputMode == "verbose" {
		fmt.Println("\nInitial calves breed composition: ")
	}
	for i := 0; i < len(array); i = i + 2 {

		f := array[i].(float64)
		top += f / 100.

		if *logger.OutputMode == "verbose" {
			fmt.Printf("\t%3.0f%% of calves are:\n", f)
		}
		var b animal.BreedComposition_t
		b.Proportion = top
		b.BreedProportions = make(map[string]float64)
		s := strings.Split(array[i+1].(string), ",")
		for j := 0; j < len(s); j = j + 2 {
			breed := strings.TrimSpace(s[j])
			f, _ := strconv.ParseFloat(strings.TrimSpace(s[j+1]), 64)
			proportion := f / 100.
			b.BreedProportions[breed] = proportion
			if *logger.OutputMode == "verbose" {
				fmt.Printf("\t\t%-8s %3.0f%%\n", breed, f)
			}
		}
		animal.CurrentCalvesBreedCompositionTable = append(animal.CurrentCalvesBreedCompositionTable, b)
	}

	return
}

// Open the output files in master.hjson
func openOutputFiles() {

	c := param["cowagefilename"]
	if c != nil {
		cowagefilename := string(c.(string))
		animal.CowAgeFile, _ = os.Create(cowagefilename)
	}

	c = param["recordsdump"]
	if c != nil {
		animal.RecordsDumpFile = string(c.(string))
	}

	c = param["breedingrecordsdump"]
	if c != nil {
		animal.BreedingRecordsDumpFile = string(c.(string))
	}
	array, ok := param["phenotypeFile"].([]interface{})
	if ok {
		s := strings.Split(array[0].(string), ",")
		animal.PhenotypeFile = strings.TrimSpace(s[0])
		animal.PhenotypeOutputTrait = strings.TrimSpace(s[1])
		f, _ := os.Create(animal.PhenotypeFile)
		animal.PhenotypeFilePointer = f
	}
	c = param["stayPhenotypeFile"]
	if c != nil {
		animal.StayPhenotypeFilePointer, _ = os.Create(string(c.(string)))
	}
	c = param["HPPhenotypeFile"]
	if c != nil {
		animal.HPPhenotypeFilePointer, _ = os.Create(string(c.(string)))
	}
	c = param["CDPhenotypeFile"]
	if c != nil {
		animal.CDPhenotypeFilePointer, _ = os.Create(string(c.(string)))
	}
	c = param["CarcassPhenotypeFile"]
	if c != nil {
		animal.CarcassPhenotypeFile, _ = os.Create(string(c.(string)))
		fmt.Fprintln(animal.CarcassPhenotypeFile, "Id YearBorn CarcassWeight QualityGrade YieldGrade pricePerPound gridPrice progPremium BackFatThickness RibEyArea calf.MarblingScore")
	}
}

// Parse the arg list looking for the input hjson file
func parseArgs() {
	// paramFile = "master.hjson"

	paramFile = flag.String("genParm", "", "The iGenDec parameter file (required)")
	indexParm = flag.String("indexParm", "", "The parameter file for specifying the index")
	logger.OutputMode = flag.String("outputMode", "verbose", "'verbose'(default) or 'model'")
	logger.User = flag.String("user", "admin", "user=[Username]")

	animal.BumpComponent = flag.String("bump", "", "Component to bump 1 unit up after burnin (optional)")

	logger.Seed = flag.Int64("seed", 1234, "Random number generator seed (int64)")

	flag.Parse()

	if *logger.OutputMode == "verbose" {
		fmt.Printf("\n\t*** iGenDec ver %v ***\n\n", version)
	}

	if *paramFile == "" {
		if *logger.OutputMode == "verbose" {
			fmt.Printf("Error: A parameter file name must be provided on the command line\n\tiGenDec -genParm=[file name]\n")
			// Print out a syntax message
			syntax := `Usage of ./iGenDec:
  -genParm string
    	The iGenDec parameter file (required)
  -indexParm string
	The index setup hjson file (optional)
  -outputMode string
    	'verbose'(default) or 'model' (default "verbose")
  -seed int
    	Random number generator seed (int64) (default 1234)
  -user string
    	user=[Username] (default "admin")
  -bump string,string,float
	Name of the genetic component to bump the bulls 1 unit after burnin - e.g. WW,D,1.`

			fmt.Printf("\n%s\n\n", syntax)
			log.Fatal(errors.New("no parameter file name provided"))
		} else {
			//fmt.Printf("error:noParameterFile\n")
			logger.LogWriter("no parameter file name provided")
			os.Exit(1)

		}
	}

}

// Read in the parameter hjson file and setup the map of param[key] pairs
func loadParam() {

	hjsonFile, err := os.Open(*paramFile)
	if err != nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("Failed to open " + *paramFile)
			fmt.Println(err)
		} else {
			logger.LogWriter("Failed to open parameter file " + *paramFile)
			os.Exit(1)
		}

	}
	//fmt.Println("Successfully Opened " + *paramFile)
	// defer the closing of our jsonFile so that we can parse it later on
	defer hjsonFile.Close()

	byteValue, _ := ioutil.ReadAll(hjsonFile)

	// Translate the byte array into the mapped array of name:value pairs
	if er := hjson.Unmarshal(byteValue, &param); er != nil {
		panic(er)
	}

}

// Find the conception rate per cycle using
// a lookup table
func conceptionPerCycle(l int64, c float64) float64 {

	baseCycles := l / 21

	m := l % 21

	var frac float64

	if m == 0 {
		frac = 1.0
	} else {
		frac = 1. - float64(m)/21.0
	}

	var baseA float64
	var baseC float64

	// What is a for base cycles
	for a := .01; a <= 1.0; a = a + .01 {

		baseC = seasonConceptionRate(baseCycles, a)

		if c <= baseC {
			baseA = a
			break
		}
	}

	// What is A for base+1 cycles
	for a := .01; a <= 1.0; a = a + .01 {
		nextC := seasonConceptionRate(baseCycles+1, a)
		if c <= nextC {
			deltaA := baseA - a

			rate := a + frac*deltaA

			return rate
		}
	}

	return 1.0
}

// Calculate the season's conception rate for a given number
// of cycles and a per cycle conception rate
func seasonConceptionRate(cycles int64, a float64) float64 {

	var cumulativeConception float64
	var c int64

	for c = 0; c < cycles; c++ {
		cumulativeConception = cumulativeConception + (1.0-cumulativeConception)*a
	}

	return cumulativeConception
}
