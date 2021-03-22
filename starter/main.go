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
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/blgolden/iGenDecModel/iGenDec/animal"
	"github.com/blgolden/iGenDecModel/iGenDec/ecoIndex"
	"github.com/blgolden/iGenDecModel/iGenDec/logger"

	"strings"
	"time"

	"github.com/hjson/hjson-go"
	"github.com/remeh/sizedwaitgroup"
	"gonum.org/v1/gonum/stat"
)

var debug bool = false // write a trait's sample values to Samples file for debugging.
var debugTrait string = "base"

var version string = "beta0.0.1"
var modelParam *string
var indexParam *string
var results []float64
var numberSpawned int // Number of simulations to spawn per bump
var seeds []int       // seeds used for each simulation
var paramMaster map[string]interface{}
var outputFile *string
var databasePath *string
var bmean, bstddev, indexErrorVar float64

type mevTable_t struct {
	trait            string
	component        string
	meanNetReturns   float64
	stddevNetReturns float64
	mev              float64
	stddevMeanNR     float64
	correlation      float64 // between traits and index when a dataset is named.
	emphasis         float64 // percent emphasis of this trait
	geneticStdDev    float64 // genetic variance of this component
	headerName       string  // If there's a datafile of EPDs this gets set to the header value
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
	databasePath = flag.String("database-path", "", "Path top level directory where the EPD data are stored")

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
	Print the version number and exit
  -database-path string
    Path to the top level directory where the EPD data are stored`

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

// Load the master hjson so that the variance components can be used to calculate the percent emphasis
func loadGeneticVariances() {
	hjsonFile, err := os.Open(*modelParam)
	if err != nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("Failed to open " + *modelParam)
			fmt.Println(err)
		} else {
			logger.LogWriter("Failed to open parameter file " + *modelParam)
			os.Exit(1)
		}
	}

	defer hjsonFile.Close()

	byteValue, _ := ioutil.ReadAll(hjsonFile)

	if er := hjson.Unmarshal(byteValue, &paramMaster); er != nil {
		logger.LogWriterFatal("Failed to unmarshal general hjson")
	}

	carray, ok := paramMaster["Components"].([]interface{})
	if !ok {
		logger.LogWriterFatal("'Components:' key not found in general parameters hjson.")
	}

	// Get the traits and components in the order they appear in the VC matrix
	for i := range carray {
		animal.Components = append(animal.Components, strings.TrimSpace(carray[i].(string)))
		v := strings.TrimSpace(carray[i].(string))
		splt := strings.Split(v, ",")
		var c animal.Component_t

		c.TraitName = strings.TrimSpace(splt[0])
		c.Component = strings.TrimSpace(splt[1]) // comp = D or M
		animal.ComponentList = append(animal.ComponentList, c)
	}

	// Get the vc matrix
	var Vc []float64

	array, _ := paramMaster["genetic"].([]interface{})
	for k := range array {
		Vc = append(Vc, array[k].(float64))
	}

	n := int(math.Sqrt(float64(len(Vc))))
	for l, c := range mevTable {
		for j, g := range animal.ComponentList {
			if c.trait == g.TraitName && c.component == g.Component {
				c.geneticStdDev = math.Sqrt(Vc[Index2D(j, j, n)])
				if c.trait == "STAY" {
					c.geneticStdDev = c.geneticStdDev * 100.
				}
				mevTable[l] = c
			}
		}
	}

	// calculate the emphasis values
	var sumE float64
	for i := range mevTable {
		sumE += math.Abs(mevTable[i].mev) * mevTable[i].geneticStdDev
	}
	for i := range mevTable {
		mevTable[i].emphasis = math.Abs(mevTable[i].mev) * mevTable[i].geneticStdDev / sumE
	}

}

var filenameXref = "comp_fn_pairs.hjson"

func findFilename(databasePath string) (string, error) {
	// Find the database filename
	infos, err := ioutil.ReadDir(databasePath)
	if err != nil {
		return "", fmt.Errorf("reading database dir: %w", err)
	}
	var databaseFile string
	for _, info := range infos {
		if !info.IsDir() && filepath.Ext(info.Name()) == ".csv" {
			if databaseFile != "" {
				return databaseFile, fmt.Errorf("multiple csv files in directory")
			}
			databaseFile = info.Name()
		}
	}
	if databaseFile == "" {
		return databaseFile, fmt.Errorf("could not find a csv database")
	}
	return databaseFile, nil
}

// If there's a target database then calculate the correlations between the index
// and the trait components
func calculateCorrelations() {
	database, ok := paramMaster["target-database"].(string)
	if !ok {
		return
	}

	if databasePath == nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("target-database specified but -database-path parameter not set")
		}
		logger.LogWriterFatal("target-database specified but -database-path parameter not set")
	}
	comppath := filepath.Join(*databasePath, database)
	csvfilename, err := findFilename(comppath)
	if err != nil {
		logger.LogWriterFatal(err.Error())
	}
	compFile := filepath.Join(comppath, csvfilename)
	csvFile, err := os.Open(compFile)
	if err != nil {
		if *logger.OutputMode == "verbose" {
			fmt.Println("Failed to open " + compFile)
			fmt.Println(err)
		} else {
			logger.LogWriter("Failed to open datafile " + compFile)
			os.Exit(1)
		}
	}

	defer csvFile.Close()

	dataMap := CSVToMap(csvFile)
	xref, _ := loadXref(comppath)

	linkHeaderAndMevTable(xref)

	// Calculate index values
	var score []float64
	var sumY, sumY2 float64
	for _, c := range dataMap {
		var s float64
		for _, f := range mevTable {
			if f.headerName != "" {
				e, _ := strconv.ParseFloat(c[f.headerName], 64)
				s += e * f.mev
			}
		}
		score = append(score, s)
		sumY += s
		sumY2 += s * s
	}

	n := float64(len(score))
	sqrtY := math.Sqrt(sumY2 - math.Pow(sumY, 2)/n)

	for j, f := range mevTable {
		if f.headerName != "" {

			var sumX, sumX2, sumXY float64
			for i, c := range dataMap {
				e, _ := strconv.ParseFloat(c[f.headerName], 64)
				sumX += e
				sumX2 += e * e
				sumXY += e * score[i]

			}
			sqrtX := math.Sqrt(sumX2 - math.Pow(sumX, 2)/n)

			dividend := sumXY - ((sumX * sumY) / n)
			divisor := sqrtX * sqrtY

			mevTable[j].correlation = dividend / divisor
		}
	}

	return
}

// Set the header values in the mevTable
func linkHeaderAndMevTable(xref map[string]Field) {
	for _, x := range xref {
		for i, _ := range mevTable {
			key := mevTable[i].trait + "," + mevTable[i].component
			if x.Key == key {
				mevTable[i].headerName = x.Header
			}
		}
	}
	return
}

// Field describes a field in the CSV file read in from the comp_fn_pair.hjson file
type Field struct {
	Key     string `json:"name"`
	Header  string `json:"header"`
	Comment string `json:"comment"`
	Select  bool   `json:"select"`
	idx     int
}

func loadXref(databasepath string) (map[string]Field, error) {

	xref := make(map[string]Field)

	data, err := ioutil.ReadFile(filepath.Join(databasepath, filenameXref))
	if err != nil {
		return xref, fmt.Errorf("reading database xref: %w", err)
	}

	// Map each field we're searching for using hjson
	var m []interface{}
	if err = hjson.Unmarshal(data, &m); err != nil {
		return xref, fmt.Errorf("unmarshalling xref: %w", err)
	}

	// We can then remarshall/unmarshall the interface to automatically put the struct into our one
	for idx, v := range m {
		data, err = json.Marshal(v)
		if err != nil {
			return xref, fmt.Errorf("remarshalling field %d: %w", idx, err)
		}
		f := Field{}
		if err = json.Unmarshal(data, &f); err != nil {
			return xref, fmt.Errorf("unmarshalling field %d: %w", idx, err)
		}
		f.idx = idx
		xref[f.Key] = f
	}

	return xref, nil
}
func Index2D(row int, col int, dim int) int {
	return dim*col + row
}

// CSVToMap takes a reader and returns an array of dictionaries, using the header row as the keys
func CSVToMap(reader io.Reader) []map[string]string {
	r := csv.NewReader(reader)
	rows := []map[string]string{}
	var header []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if header == nil {
			header = record
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			rows = append(rows, dict)
		}
	}
	return rows
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
		fmt.Printf("%s,%s,%f\n", co.trait, co.component, co.mev*2.0) // Multiply by 2 to apply to EPD
	}
}

func main() {

	verbose := "verbose"
	logger.OutputMode = &verbose

	initialize()

	simulateIndexComponents()

	loadGeneticVariances()

	calculateCorrelations()

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
			e := fmt.Sprintf("%f", co.emphasis)
			f.WriteString("      emphasis: " + e + "\n")
			c := fmt.Sprintf("%f", co.correlation)
			f.WriteString("      correlation: " + c + "\n")
			g := fmt.Sprintf("%f", co.geneticStdDev)
			f.WriteString("      geneticStdDev: " + g + "\n")
			s := fmt.Sprintf("%f", co.mev*2.0) // For application to EPD, not EBV
			f.WriteString("      mev: " + s + "\n   }")
			f.WriteString("\n")
		}
		f.WriteString("   ]\n}\n")
	}

	if debug {
		fmt.Println("debug is set to true.  Do you really want that?")
	}

}
