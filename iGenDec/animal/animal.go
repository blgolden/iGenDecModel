// herd project herd.go
//
// Defines an animal and its characteristics
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
	"os"

	"gonum.org/v1/gonum/mat"
	//"gonum.org/v1/gonum/stat/distuv"
)

type Date int // Simulation date

var SexCodes []string

const (
	Bull   string = "M" // Breeding male or calf
	Heifer string = "F" // Young female not yet selected
	Cow    string = "C" // A breeding female
	Steer  string = "S" // Going to be fed
)

type Bred bool // Is the animal pregnant

const (
	Open     Bred = false // Not pregnant
	Pregnant Bred = true  // Is bred
)

type AnimalId uint32 // Animal identification numbers sequentially generated starting at 1
// 0 means unknown - e.g. foundation sire and dams

type BreedingRec struct {
	YearBred    int      // Simulation calendar year of breeding
	DateBred    Date     // Within year Simulation date of breeding date. Year = 1 is start of simulation year
	Bred        Bred     // Open or Pregnant
	Bull        AnimalId // Id of sire
	CalvingDate Date     // Simulation date of calving
}

type Trait string

var TraitList = []Trait{}
var TraitMean = make(map[string]float64) // Means of traits mapped to trait name.  Coms from the Traits: key in master.hjson

type Bv int

// This is the animal class
type Animal struct {
	Id        AnimalId // This animal's simulation ID
	Sire      AnimalId // This animal's sire simulation ID
	Dam       AnimalId // This animal's dam simulation ID
	Sex       string   // Sex of this animal
	BirthDate Date     // To calculate age
	YearBorn  int      // Year of simulation born
	Dead      Date     // Animal died - e.g. calving difficulty

	Active         bool // Is this an active member of the herd - Cow or Bull
	DateCowEntered int  // Date a heifer became an active cow
	DateCowCulled  int  // If this is a cow, what year was she culled

	HerdName string // Name of the Herd last active in

	BreedingValue *mat.VecDense // Breeding Values
	Residual      *mat.VecDense // Residual values

	BreedComposition map[string]float64 //
	BreedingRecords  []BreedingRec      // Day of simulation bred if Cow

	AumToWeaning             []Aum_t // By month and year
	AumWeanThruBackgrounding []Aum_t // By month
	CowAum                   []Aum_t

	FeedlotTotalFeedIntake float64 // This animal's feedlot total feed consumption
	HarvestWeight          float64 // Weight as a fat animal at slaughter

	CarcassWeight    float64 // actual carcass weight
	MarblingScore    float64 // actual marbling score when slaughtered
	BackFatThickness float64 // actual backfat thickness when slaughtered
	RibEyArea        float64 // actual ribeye area when slaughtered
}

type FedType int // Type of feeding program to slaughter

// How the animal was finished
const (
	CalfFed      FedType = 0 // Fed at younger ages close to weaning
	Conventional FedType = 1 // Fed at conventional ages slaughters around 22 months
	CullHeifer   FedType = 2 // Open heifer bred to calv at 2yoa
	CullCow      FedType = 3 // Any cull older female not a heifer
)

type Component_t struct {
	TraitName string
	Component string
}
type BumpComponent_t struct {
	TraitName string
	Component string
	Value     float64
}

type BreedEffects_t struct {
	TraitName string
	Component string             // "D" for Direct, "M" for Maternal
	CowOrCalf string             // Trait of a cow or trait of a calf - e.g., stay or bw direct
	Effects   map[string]float64 // Array of breed effects in order of the breeds in the 1st row of the BreedEffects: key in master.hjson
}

var Breeds map[Component_t]BreedEffects_t // See animal.go and technical documentation for this struct type
var BreedsList []string                   // A list of the breeds in the master.hjson.  It is the 1st line of the BreedsEffects: key

var HeterosisCodes map[string]string // The breed to breed classification categories in the HeterosisCodes: key in master.hjson
var HeterosisCrossClasses []string   // The cross class designations from the HeterosisValues talbe 1st line in the master.hjson

type HeterosisValues_t struct {
	TraitName string
	Component string
	Values    map[string]float64 // These are mapped by the string of HeterosisCrossClasses
}

var HeterosisValues map[Component_t]HeterosisValues_t // These are mapped according to their Component_t

// Sex by aod by breed by trait
type BTS_t struct {
	Breed string
	Trait string
	Sex   string
	Aod   int
}

var BreedTraitSexAod map[BTS_t]float64 // mapped by Breed then trait

// Age effects in days
type InterceptSlope_t struct {
	Slope float64
	Age   float64
}

var TraitAgeEffects map[string]InterceptSlope_t

// Optional file to write out a phenotype components for debugging
var PhenotypeFile string        // File name to write output
var PhenotypeOutputTrait string // A valid trait name in the master.hjson
var PhenotypeFilePointer *os.File
var StayPhenotypeFilePointer *os.File // For debugging cow conception routines
var HPPhenotypeFilePointer *os.File
var CDPhenotypeFilePointer *os.File
var CarcassPhenotypeFile *os.File // For optional write of slaughter cattle phenotypes QG,YQ, etc.

//var HeiferPregnancyDistribution distuv.Normal

type Aum_t struct {
	Year        int     // Year this AUM is incurred
	MonthOfYear int     // Calendar month of the year this Aum was accounted
	Aum         float64 // Aum consumed this month
	Weight      float64 // weight end of this month
	Location    int     // For debug where the call came from
}

var CalfAumAt500 float64 // Amount of Aum at 500 lbs calf
var CowAumAt1000 float64 // Amount of AUM at 1000 lbs animal

var BumpComponent *string // Name,Name of the component  to bump the bulls 1 unit after burnin

var BullMerit []float64 // Genetic merit of the foundation bulls fro meritFoundationBulls key

type Sales_t struct {
	NheadOpen float64 // number of head sold open - e.g., cows
	NheadOld  float64 // number of head sold old
	CumWt     float64 // cumulative weight of nhead
}

var WtCullCows map[int]Sales_t

var IndexType string
var IndexTerminal bool
var BackgroundDays float64
var DaysOnFeed float64

var CDVar float64 // phenotypic variance for CDF

var ComponentList []Component_t
