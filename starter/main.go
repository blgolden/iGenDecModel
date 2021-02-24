// starter project main.go
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
	//"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/animal"
	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/ecoIndex"
	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/logger"

	//"strings"
	"time"

	"github.com/hjson/hjson-go"
	"github.com/remeh/sizedwaitgroup"
	"gonum.org/v1/gonum/stat"
)

var debug bool = false // write a trait's sample values to Samples file for debugging.
var debugTrait string = "base"

var version string = "alpha0.2.0b"
var modelParam *string
var indexParam *string
var results []float64
var numberSpawned int // Number of simulations to spawn per bump
var seeds []int       // seeds used for each simulation

var outputFile *string

var bmean, bstddev, indexErrorVar float64

type mevTable_t struct {
	trait            string
	component        string
	meanNetReturns   float64
	stddevNetReturns float64
	mev              float64
	stddevMeanNR     float64
}

// Table of marginal economic values
var mevTable []mevTable_t

// Spawn a go routine for each sample
// This is the go routine
func multistart(swg *sizedwaitgroup.SizedWaitGroup, comp animal.Component_t, seed string, c chan float64) {

	defer swg.Done()

	var bump string
	if comp.TraitName != "base" {
		if comp.TraitName == "STAY" || comp.TraitName == "HP" {
			bump = "-bump=" + comp.TraitName + "," + comp.Component + ",.01"
		} else {
			bump = "-bump=" + comp.TraitName + "," + comp.Component + ",1.0"
		}
	}

	model := "-genParm=" + *modelParam
	index := "-indexParm=" + *indexParam

	a := "-seed=" + seed

	response, er := exec.Command("iGenDec", model, index, "-outputMode=quiet", a, bump).CombinedOutput()

	if er != nil {
		log.Fatal(er)
	}

	v, _ := strconv.ParseFloat(string(response), 64)

	c <- v

}

//Launch a simulation with a bump trait e.g., WW,D
func launchSimulations(comp animal.Component_t) (float64, float64) {

	results = nil

	start := time.Now()

	if *logger.OutputMode == "verbose" || *logger.OutputMode == "table" {
		fmt.Println("Bumping: ", comp.TraitName, comp.Component)
	}

	swg := sizedwaitgroup.New(runtime.NumCPU())

	ch := make(chan float64, numberSpawned) // Buffered channel for results

	for i := 0; i < numberSpawned; i++ {
		swg.Add()
		seed := strconv.Itoa(seeds[i])
		go multistart(&swg, comp, seed, ch)
	}

	swg.Wait()

	for i := 0; i < numberSpawned; i++ {
		v := <-ch
		results = append(results, v)
	}

	elapsed := time.Since(start)

	mean, variance := stat.MeanVariance(results, nil)
	if *logger.OutputMode == "verbose" {
		fmt.Println("N Samples:", numberSpawned, "\nMean:", mean, "\nStdDev:", math.Sqrt(variance), "\nStdDev(Mean):", math.Sqrt(variance/float64(numberSpawned)))
		fmt.Println("Total time:", elapsed, "Time per sample:", elapsed.Seconds()/float64(numberSpawned), "Using", runtime.NumCPU(), "CPUs")
	}

	if comp.TraitName == debugTrait && debug {
		fp, _ := os.OpenFile("Samples", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer fp.Close()
		for i, v := range results {
			fmt.Fprintln(fp, i, v, seeds[i])
		}
	}
	return mean, math.Sqrt(variance)
}

// Parse the arg list looking for the input hjson file
func parseArgs() {

	modelParam = flag.String("genParm", "", "The iGenDec parameter file (required)")
	indexParam = flag.String("indexParm", "", "The parameter file for specifying the index")
	logger.OutputMode = flag.String("outputMode", "verbose", "'verbose'(default) or 'table'")
	logger.User = flag.String("user", "admin", "user=[Username]")
	ns := flag.Int("nSamples", 100, "Number of samples per bump (default 100)")
	logger.Seed = flag.Int64("seed", 1234, "Random number generator seed (int64)")
	isVersion := flag.Bool("version", false, "prints the version number of starter")
	outputFile = flag.String("outputFile", "", "Optional jjson file of MEV")

	flag.Parse()

	if *isVersion {
		fmt.Println("Version:", version)
		os.Exit(0)
	}

	numberSpawned = *ns

	if *indexParam == "" {
		if *logger.OutputMode == "verbose" {

			// Print out a syntax message
			syntax := `Usage of ./starter:
  -genParm string
    	The iGenDec parameter file (required)
  -indexParm string
	The index setup hjson file (optional)
  -outputMode string
    	'verbose' or 'table' or 'web' (default "verbose")
  -seed int
    	Random number generator seed (int64) (default 1234)
  -user string
    	user=[Username] (default "admin")
  -nSamples int
	Number of samples per bump (default 100)
  -outputFile string
	Optional hjson file of MEV
  -version
	Print the version number and exit`

			fmt.Printf("\n%s\n\n", syntax)
			logger.LogWriterFatal("no parameter file name provided")
			os.Exit(1)
		}

	}

}

// Read the arguments and initialize the list of index components
func initialize() {

	parseArgs()

	loadIndexParam()

	ecoIndex.LoadIndexComponents()

	rand.Seed(*logger.Seed)

	for i := 0; i < numberSpawned; i++ {
		seeds = append(seeds, rand.Intn(100000))
	}

	results = make([]float64, numberSpawned)
}

// Read in the parameter hjson file and setup the map of param[key] pairs
func loadIndexParam() {

	hjsonFile, err := os.Open(*indexParam)
	if err != nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("Failed to open " + *indexParam)
			fmt.Println(err)
		} else {
			logger.LogWriter("Failed to open parameter file " + *indexParam)
			os.Exit(1)
		}

	}

	defer hjsonFile.Close()

	byteValue, _ := ioutil.ReadAll(hjsonFile)

	// Translate the byte array into the mapped array of name:value pairs
	if er := hjson.Unmarshal(byteValue, &ecoIndex.ParamIndex); er != nil {
		logger.LogWriterFatal("Failed to unmarshal hjson")
	}
}

// Bump each index component by 1 (STAY by .01) and construct the table of MEV
func simulateIndexComponents() {

	var baseComp animal.Component_t // is nil
	baseComp.TraitName = "base"
	var baseMev mevTable_t

	bmean, bstddev = launchSimulations(baseComp)

	baseMev.trait = "base"

	n := float64(numberSpawned)

	for _, co := range ecoIndex.IndexComponents {
		m, s := launchSimulations(co)
		var mev mevTable_t
		mev.component = co.Component
		mev.trait = co.TraitName
		mev.meanNetReturns = m
		mev.stddevNetReturns = s
		mev.mev = m - bmean
		mev.stddevMeanNR = math.Sqrt(s * s / n)

		indexErrorVar += (s*s)/n + (bstddev*bstddev)/n

		mevTable = append(mevTable, mev)
	}

}

// Write a table of MEV to the screen
func publishIndex() {

	fmt.Println("\t ________________________________________________________________________")
	fmt.Println("\t| Trait  | Comp | Mean NRLML   | StdDev(NRLML) |     MEV    | SDMeanNRLML| ")
	fmt.Println("\t|________|______|______________|_______________|____________|____________|")
	fmt.Printf("\t| base   |  -   |  %10.2f  |    %10.2f |      -     |      -     |\n", bmean, bstddev)

	for _, co := range mevTable {
		fmt.Printf("\t|% 5s   |  %s   |  %10.2f  |    %10.2f | %10.2f | %10.2f |\n",
			co.trait, co.component, co.meanNetReturns, co.stddevNetReturns, co.mev, co.stddevMeanNR)
	}
	fmt.Println("\t|________________________________________________________________________|")
	fmt.Printf("\tNote, these MEV are to be applied to EBV, not EPD\n")
	fmt.Printf("\t *Number of samples per bump: %d\n", numberSpawned)
	fmt.Printf("\n\tStd Error of the Index: %10.2f\n\n", math.Sqrt(indexErrorVar))
}

// write to stream
func dumpMev() {
	for _, co := range mevTable {
		fmt.Printf("%s,%s,%f\n", co.trait, co.component, co.mev*2.0)	// Multiply by 2 to apply to EPD
	}
}

func main() {

	verbose := "verbose"
	logger.OutputMode = &verbose

	initialize()

	simulateIndexComponents()

	if *logger.OutputMode == "table" || *logger.OutputMode == "verbose" {
		publishIndex()
	} else if *logger.OutputMode == "web" {
		dumpMev()
	}

	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			logger.LogWriterFatal("Cannot open outputFile")
		}
		_, err = f.WriteString("{\n   indexElement:[\n")
		for _, co := range mevTable {
			if co.trait == "CD" {
				co.trait = "CE"
				co.mev = co.mev * -1.0
			}
			f.WriteString("{\n      trait: " + co.trait + "\n")
			f.WriteString("      component: " + co.component + "\n")
			s := fmt.Sprintf("%f", co.mev*2.0)	// For application to EPD, not EBV
			f.WriteString("      mev: " + s + "\n   }")
			f.WriteString("\n")
		}
		f.WriteString("   ]\n}\n")
	}

	if debug {
		fmt.Println("debug is set to true.  Do you really want that?")
	}

}
