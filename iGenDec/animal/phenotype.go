// phenotype
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
	"fmt"
	//"os"
	"math"
	"strings"

	"github.com/blgolden/iGenDecModel/iGenDec/logger"
)

var ResidualStayStdDev float64
var ResidualHpStdDev float64

// Return the trait index in the traits array list
func ResidualIndex(traitName string) (index int) {
	for i, v := range Traits {
		if v == traitName {
			return i
		}
	}
	logger.LogWriter("Request for " + traitName + " not found in Traits[] from the master parameter file.")
	return (-1)
}

// Return the index of this genetic component in the genetic variance matrix
func GeneticIndex(traitName string, comp string) (index int) {
	for i, v := range Components {
		splt := strings.Split(v, ",")
		if splt[0] == traitName {
			if strings.TrimSpace(splt[1]) == comp { // comp = D or M
				return i
			}
		}
	}
	return -1 // No genetic effect - e.g. no maternal additive genetic effect
}

// Return the additive genetic effect
func GeneticEffect(indx int, animal Animal) (u float64) {

	if indx >= 0 {
		u = animal.BreedingValue.At(indx, 0)
	} else {
		u = 0.0
	}

	return u
}

// Return the breed effect on
func BreedEffect(trait string, thisAnimal Animal) (effect float64) {

	effect = 0.0

	// For efficiency do this first
	// Should already know if thisAnimal.Dam != 0
	var m = Component_t{trait, "M"}
	if mat, ok := Breeds[m]; ok && thisAnimal.YearBorn > 0 { // the && allows us to initialize the CD distribution
		for key, breed := range Records[thisAnimal.Dam-1].BreedComposition {
			effect += mat.Effects[key] * breed
		}
	}

	// Additive direct genetic effect
	var d = Component_t{trait, "D"}

	// There will always be a direct effect
	for key, breed := range thisAnimal.BreedComposition {
		effect += Breeds[d].Effects[key] * breed
	}

	return effect

}

// Return the heterosis effect
func HeterosisEffect(trait string, thisAnimal Animal) (effect float64, l bool) {

	sire := Records[thisAnimal.Sire].BreedComposition
	dam := Records[thisAnimal.Dam].BreedComposition

	// Does this trait have a maternal heterosis effect
	var mval HeterosisValues_t
	Ok := true
	var m Component_t
	m.TraitName = trait
	m.Component = "M"

	dsire := sire
	ddam := dam
	if mval, Ok = HeterosisValues[m]; Ok {
		if Records[thisAnimal.Dam].Sire > 0 && Records[thisAnimal.Sire].Dam > 0 {
			dsire = Records[thisAnimal.Sire].BreedComposition
			ddam = Records[thisAnimal.Dam].BreedComposition
		}
		for msbreed, mspct := range dsire {
			for mdbreed, mdpct := range ddam {
				if msbreed != mdbreed {
					mcode := HeterosisCodes[msbreed] + "x" + HeterosisCodes[mdbreed]
					var mhval float64
					ok := true
					if mhval, ok = mval.Values[mcode]; !ok {
						code := HeterosisCodes[mdbreed] + "x" + HeterosisCodes[msbreed]
						if mhval, ok = mval.Values[code]; !ok {
							logger.LogWriterFatal("Cannot find a maternal heterosis value for " + code + " Trait: " + trait)
						}
					}
					effect += mspct * mdpct * mhval
				}
			}
		}
	}

	effect = 0.0

	for sbreed, spct := range sire {
		for dbreed, dpct := range dam {
			if sbreed != dbreed {
				code := HeterosisCodes[sbreed] + "x" + HeterosisCodes[dbreed]
				var c Component_t
				c.TraitName = trait
				c.Component = "D"
				var hval float64
				ok := true
				if hval, ok = HeterosisValues[c].Values[code]; !ok {
					code = HeterosisCodes[dbreed] + "x" + HeterosisCodes[sbreed]
					if hval, ok = HeterosisValues[c].Values[code]; !ok {
						logger.LogWriterFatal("Cannot find a heterosis value for " + code + "Trait:" + trait)
					}
				}
				effect += spct * dpct * hval
			}
		}
	}

	l = true
	return effect, l
}

// Return the BIF aod catagory where 0=2yoa, 1=3yoa, 2=4yoa, 3=5 thru9yoa, and 10 = >=10 yoa
func WhatAod(thisAnimal Animal) (aod int) {

	if thisAnimal.Dam <= 0 {
		aod = 3 // Mature cow
		return aod
	}

	age := thisAnimal.BirthDate - Records[thisAnimal.Dam].BirthDate

	if age >= 639 && age <= 1003 {
		aod = 0
	} else if age >= 1004 && age <= 1369 {
		aod = 1
	} else if age >= 1370 && age <= 1734 {
		aod = 2
	} else if age >= 1735 && age <= 3560 {
		aod = 3
	} else {
		aod = 4
	}

	return

}

// Return the net AOD effect of even crossbreeds
func SexAgeOfDamEffect(trait string, thisAnimal Animal) (effect float64) {

	for s, v := range thisAnimal.BreedComposition {
		var b BTS_t
		b.Breed = s
		b.Trait = trait
		b.Sex = thisAnimal.Sex
		b.Aod = WhatAod(thisAnimal)

		effect += BreedTraitSexAod[b] * v
	}

	return
}

func AgeEffect(trait string, thisAnimal Animal) (effect float64) {

	// Calculate age at event

	if trait == "HP" {
		effect = 0.0
		return effect
	}
	thisDeviation := float64(thisAnimal.BirthDate) -
		Herds[thisAnimal.HerdName].SumBirthDates[thisAnimal.YearBorn]/Herds[thisAnimal.HerdName].NBorn[thisAnimal.YearBorn]

	/*fmt.Println("LOC AGE", thisAnimal.Id, thisAnimal.BirthDate, thisAnimal.YearBorn,
	Herds[thisAnimal.HerdName].SumBirthDates[thisAnimal.YearBorn]/
		Herds[thisAnimal.HerdName].NBorn[thisAnimal.YearBorn],
	float64(thisAnimal.BirthDate)-Herds[thisAnimal.HerdName].SumBirthDates[thisAnimal.YearBorn]/
		Herds[thisAnimal.HerdName].NBorn[thisAnimal.YearBorn])
	*/
	effect = thisDeviation * TraitAgeEffects[trait].Slope

	return effect
}

// Calculate raw phenotype
func Phenotype(thisAnimal Animal, thisTrait string) (pheno float64, l bool) {

	if thisAnimal.YearBorn < 1 {
		l = false
		return 0.0, l
	}

	resIdx := ResidualIndex(thisTrait)

	matIdx := GeneticIndex(thisTrait, "M")

	if matIdx > -1 && thisAnimal.Dam == 0 { // Dam is unknown but trait has a maternal effect
		l = false
		return 0.0, l
	}

	var geneticMaternalEffect float64
	if matIdx >= 0 {
		geneticMaternalEffect = GeneticEffect(matIdx, Records[thisAnimal.Dam-1]) * .5
	}

	geneticDirectEffect := GeneticEffect(GeneticIndex(thisTrait, "D"), thisAnimal)

	// for now permEnvEffect := PermenentEnvEffect(PermEnvIndex(thisTrait))

	breedEffects := BreedEffect(thisTrait, thisAnimal)

	var heterosisEffects float64
	ok := true
	if heterosisEffects, ok = HeterosisEffect(thisTrait, thisAnimal); !ok { // Sire of dam of dam of dam unknown and trait has a maternal het effect
		l = false
		return 0.0, l
	}

	sexAgeOfDamEffects := SexAgeOfDamEffect(thisTrait, thisAnimal) // Only sex effect if not AOD effect

	ageEffect := AgeEffect(thisTrait, thisAnimal)

	pheno = TraitMean[thisTrait] +
		breedEffects +
		heterosisEffects +
		sexAgeOfDamEffects +
		ageEffect +
		geneticDirectEffect +
		geneticMaternalEffect +
		//permEnvEffect +
		thisAnimal.Residual.At(resIdx, 0)

	if thisTrait == PhenotypeOutputTrait {
		fmt.Fprintln(PhenotypeFilePointer, thisAnimal.Id, thisAnimal.YearBorn, TraitMean[thisTrait], breedEffects,
			heterosisEffects, sexAgeOfDamEffects, ageEffect, geneticDirectEffect, geneticMaternalEffect,
			thisAnimal.Residual.At(resIdx, 0), pheno)
	}
	l = true
	return pheno, l

}

// Calculate phenotype without adjusting for age
func PhenotypeAtMeanAge(thisAnimal Animal, thisTrait string) (pheno float64, l bool) {

	if thisAnimal.YearBorn < 1 {
		l = false
		return 0.0, l
	}

	resIdx := ResidualIndex(thisTrait)

	matIdx := GeneticIndex(thisTrait, "M")

	if matIdx > -1 && thisAnimal.Dam == 0 { // Dam is unknown but trait has a maternal effect
		l = false
		return 0.0, l
	}

	var geneticMaternalEffect float64
	if matIdx >= 0 {
		geneticMaternalEffect = GeneticEffect(matIdx, Records[thisAnimal.Dam-1]) * .5
	}

	geneticDirectEffect := GeneticEffect(GeneticIndex(thisTrait, "D"), thisAnimal)

	// for now permEnvEffect := PermenentEnvEffect(PermEnvIndex(thisTrait))

	breedEffects := BreedEffect(thisTrait, thisAnimal)

	var heterosisEffects float64
	ok := true
	if heterosisEffects, ok = HeterosisEffect(thisTrait, thisAnimal); !ok { // Sire of dam of dam of dam unknown and trait has a maternal het effect
		l = false
		return 0.0, l
	}

	sexAgeOfDamEffects := SexAgeOfDamEffect(thisTrait, thisAnimal) // Only sex effect if not AOD effect

	pheno = TraitMean[thisTrait] +
		breedEffects +
		heterosisEffects +
		sexAgeOfDamEffects +
		geneticDirectEffect +
		geneticMaternalEffect +
		//permEnvEffect +
		thisAnimal.Residual.At(resIdx, 0)

	l = true
	return pheno, l

}

// phenotype of weaning wt
func WeaningWtPhenotype(thisAnimal Animal) (float64, bool) {

	pa, ok := PhenotypeAtMeanAge(thisAnimal, "WW")

	if !ok {
		return 0.0, false
	}

	pb, _ := Phenotype(thisAnimal, "BW")

	thisDeviation := float64(thisAnimal.BirthDate) -
		Herds[thisAnimal.HerdName].SumBirthDates[thisAnimal.YearBorn]/Herds[thisAnimal.HerdName].NBorn[thisAnimal.YearBorn]

	pheno := (pa-pb)/205*(thisDeviation+205) + pb

	return pheno, true
}

func withTolerane(a, b float64) bool {
	tolerance := 0.001
	if diff := math.Abs(a - b); diff < tolerance {
		return true
	}
	return false

}

// Convert a stayability value to a conception rate in 21 days value
// Assumes the stayability is for a 60 d breeding season - really 63 but who's counting
func Stay2Concept21days(s float64) float64 {
	// binary search the expression: s = c^3 - 3c^2 + 3c
	if s <= 0.0 {
		return 0.0
	}
	if s >= 1.0 {
		return 1.0
	}

	var c, low, high float64
	low = 0.0
	high = 1.0

	for low <= high {
		c = (low + high) / 2.0

		thiss := math.Pow(c, 3.0) - 3*math.Pow(c, 2.0) + 3*c
		if withTolerane(s, thiss) {
			return c
		}

		if thiss > s {
			high = c
		} else {
			low = c
		}

	}

	return c

}

// Calculate the stayability phenotype at a particular day - e.g. at breeding
// This is 6 yoa conception rate adjusted to a conception rate at a particular
// age.
func StayAtAgePhenotype(thisAnimal Animal, today Date) (pheno float64) {

	geneticDirectEffect := GeneticEffect(GeneticIndex("STAY", "D"), thisAnimal)

	// for now permEnvEffect := PermenentEnvEffect(PermEnvIndex(thisTrait))

	breedEffects := BreedEffect("STAY", thisAnimal)

	var heterosisEffects float64
	ok := true
	heterosisEffects, ok = HeterosisEffect("STAY", thisAnimal)
	if !ok {
		heterosisEffects = 0.0
	}

	//sexAgeOfDamEffects := SexAgeOfDamEffect("STAY", thisAnimal) // Only sex effect if not AOD effect

	daysOfAge := today - thisAnimal.BirthDate

	ageEffect := TraitAgeEffects["STAY"].Slope * (float64(daysOfAge) - TraitAgeEffects["STAY"].Age)

	residual := Rng.NormFloat64() * ResidualStayStdDev // Simulated as uncorrelated to other residuals

	pheno = TraitMean["STAY"] +
		//breedEffects +
		//heterosisEffects +
		//sexAgeOfDamEffects +
		ageEffect +
		geneticDirectEffect +
		//permEnvEffect +
		residual

	pheno = pheno + pheno*(1.0*heterosisEffects) // This is a multiplicative effect because this is a probability

	if StayPhenotypeFilePointer != nil {
		fmt.Fprintln(StayPhenotypeFilePointer,
			thisAnimal.Id,       // 1
			thisAnimal.YearBorn, // 2
			today,               // 3
			TraitMean["STAY"],   // 4
			breedEffects,        // 5
			heterosisEffects,    // 6
			//sexAgeOfDamEffects,  // 7
			ageEffect,           // 8
			daysOfAge,           // 9
			geneticDirectEffect, // 10
			residual,            // 11
			pheno)               // 12
	}

	return pheno
}

// Calculate the heifer pregnancy phenotype at a particular day - e.g. at breeding
func HeiferPregnancyPhenotype(thisAnimal Animal, today Date) (pheno float64) {

	geneticDirectEffect := GeneticEffect(GeneticIndex("HP", "D"), thisAnimal)

	// for now permEnvEffect := PermenentEnvEffect(PermEnvIndex(thisTrait))

	breedEffects := BreedEffect("HP", thisAnimal)

	var heterosisEffects float64
	ok := true
	heterosisEffects, ok = HeterosisEffect("HP", thisAnimal)
	if !ok {
		heterosisEffects = 0.0
	}

	sexAgeOfDamEffects := SexAgeOfDamEffect("HP", thisAnimal) // Only sex effect if not AOD effect

	daysOfAge := today - thisAnimal.BirthDate

	ageEffect := TraitAgeEffects["HP"].Slope * (float64(daysOfAge) - TraitAgeEffects["HP"].Age)

	//residual := thisAnimal.Residual.AtVec(ResidualIndex("HP")) // Simulated as uncorrelated to other residuals
	residual := Rng.NormFloat64() * ResidualHpStdDev

	//pheno = TraitMean["HP"] +
	pheno = breedEffects +
		heterosisEffects +
		sexAgeOfDamEffects +
		ageEffect +
		geneticDirectEffect +
		//permEnvEffect +
		residual

	if HPPhenotypeFilePointer != nil {
		fmt.Fprintln(HPPhenotypeFilePointer,
			thisAnimal.Id,       // 1
			thisAnimal.YearBorn, // 2
			today,               // 3
			TraitMean["HP"],     // 4
			breedEffects,        // 5
			heterosisEffects,    // 6
			sexAgeOfDamEffects,  // 7
			ageEffect,           // 8
			daysOfAge,           // 9
			geneticDirectEffect, // 10
			residual,            // 11
			pheno,               // 12
			thisAnimal.BirthDate)
	}

	return pheno
}

// Calculate the calving difficulty phenotype for of the initial heifer population
// This is used to parameterize the distribution
func CalvingDifficultyPhenotype(thisAnimal Animal, thisBreeding BreedingRec) (pheno float64) {

	geneticDirectEffect := GeneticEffect(GeneticIndex("CD", "D"), thisAnimal)
	geneticMaternalEffect := GeneticEffect(GeneticIndex("CD", "M"), thisAnimal) * .5 // Use the animal's own maternal since dam is unknown

	breedEffects := BreedEffect("CD", thisAnimal)

	var heterosisEffects float64
	ok := true
	heterosisEffects, ok = HeterosisEffect("CD", thisAnimal)
	if !ok {
		heterosisEffects = 0.0
	}

	sexAgeOfDamEffects := SexAgeOfDamEffect("CD", thisAnimal) // Only sex effect if not AOD effect

	daysOfAge := thisBreeding.CalvingDate - thisAnimal.BirthDate - 730 // 730 is 2yoa

	ageEffect := TraitAgeEffects["CD"].Slope * (float64(daysOfAge) - TraitAgeEffects["CD"].Age)

	residual := thisAnimal.Residual.AtVec(ResidualIndex("CD"))

	pheno = TraitMean["CD"] +
		breedEffects +
		heterosisEffects +
		sexAgeOfDamEffects +
		ageEffect +
		geneticDirectEffect +
		geneticMaternalEffect +
		residual

	if CDPhenotypeFilePointer != nil {
		today := 0
		bw, _ := Phenotype(thisAnimal, "BW")
		fmt.Fprintln(CDPhenotypeFilePointer,
			thisAnimal.Id,       // 1
			thisAnimal.YearBorn, // 2
			today,               // 3
			TraitMean["CD"],     // 4
			breedEffects,        // 5
			heterosisEffects,    // 6
			sexAgeOfDamEffects,  // 7
			ageEffect,           // 8
			daysOfAge,           // 9
			geneticDirectEffect, // 10
			residual,            // 11
			pheno,
			bw) // 12

	}
	return pheno
}

// Calculate the mature weight phenotype at a particular day - e.g. at weaning
func MatureWeightAtAgePhenotype(thisAnimal Animal, today Date) (pheno float64) {

	geneticDirectEffect := GeneticEffect(GeneticIndex("MW", "D"), thisAnimal)

	// for now permEnvEffect := PermenentEnvEffect(PermEnvIndex(thisTrait))

	breedEffects := BreedEffect("MW", thisAnimal)

	var heterosisEffects float64
	ok := true
	heterosisEffects, ok = HeterosisEffect("MW", thisAnimal)
	if !ok {
		heterosisEffects = 0.0
	}

	sexAgeOfDamEffects := SexAgeOfDamEffect("MW", thisAnimal) // Only sex effect if not AOD effect

	daysOfAge := today - thisAnimal.BirthDate
	if daysOfAge > 1735 { // BIF Guidelines
		daysOfAge = 1735
	}

	ageEffect := TraitAgeEffects["MW"].Slope * (float64(daysOfAge) - TraitAgeEffects["STAY"].Age)

	residual := thisAnimal.Residual.AtVec(ResidualIndex("MW")) // I know it should be different for each obs but...

	pheno = TraitMean["MW"] +
		breedEffects +
		heterosisEffects +
		sexAgeOfDamEffects +
		ageEffect +
		geneticDirectEffect +
		//permEnvEffect +
		residual

	return pheno
}

// Return the phenotype of the animal at the end of the background period.
func BackgroundingWtPhenotype(calf Animal) float64 {
	p := calf.AumWeanThruBackgrounding[len(calf.AumWeanThruBackgrounding)-1].Weight
	return p
}
