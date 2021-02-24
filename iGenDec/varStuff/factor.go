// factor.go
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
package varStuff

import (
	"fmt"

	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/animal"

	"errors"
	"math"

	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/ecoIndex"
	"gitlab.thetasolutionsllc.com/USDAProject/iGenDecModel/iGenDec/logger"

	"gonum.org/v1/gonum/mat"
)

var GvCholesky mat.Cholesky // The decomposed genetic covariance matrix
var RvCholesky mat.Cholesky // The decomposed residual covariance matrix
var VcMatrix map[string]mat.Symmetric

type Matvec_t struct {
	v []float64
}

var Vc map[string]Matvec_t

// Generic Decompose a covariance matrix
func DecompVar(name string, param map[string]interface{}) (v mat.Cholesky) {
	// Convert the values from the interface to float64 slice
	array, ok := param[name].([]interface{})
	if !ok {
		logger.LogWriterFatal("'" + name + "' key not found")
	}

	var m Matvec_t
	for i := range array {
		m.v = append(m.v, array[i].(float64))
	}
	Vc[name] = m

	// Assign the slice to a NxN gonum matrix type
	n := int(math.Sqrt(float64(len(Vc[name].v))))
	if len(Vc[name].v)%n != 0 {
		if *logger.OutputMode == "verbose" {
			panic(errors.New("The variance matrix in the parameter file is not square"))
		} else {
			logger.LogWriterFatal("The variance matrix in the parameter file is not square")
		}
	}
	if *logger.OutputMode == "verbose" {
		fmt.Printf("The "+name+" covariance matrix is %v x %v\n", n, n)
	}

	VcMatrix[name] = mat.NewSymDense(n, Vc[name].v)

	// Stayability needs a residual std dev so stubbed in here
	if name == "residual" {
		rindx := animal.ResidualIndex("STAY")
		if rindx >= 0 {
			animal.ResidualStayStdDev = math.Sqrt(VcMatrix[name].At(rindx, rindx))
			if *logger.OutputMode == "verbose" {
				fmt.Println("The Stayability residual standard deviation is: ", animal.ResidualStayStdDev)
			}
		}
		rindx = animal.ResidualIndex("HP")
		if rindx >= 0 {
			animal.ResidualHpStdDev = math.Sqrt(VcMatrix[name].At(rindx, rindx))
			if *logger.OutputMode == "verbose" {
				fmt.Println("The Hepfer Pregnancy residual standard deviation is: ", animal.ResidualHpStdDev)
			}
		}
	}

	//fv := mat.Formatted(vcMatrix, mat.Prefix("      "), mat.Squeeze())
	if *logger.OutputMode == "verbose" {
		fmt.Printf("vcMatrix = \n")
		MatPrint(VcMatrix[name])
	}

	// Get the inverse using gonum

	v, _ = Factor(name, VcMatrix[name])

	return v
}

func Factor(name string, vcMatrix mat.Symmetric) (v mat.Cholesky, ok bool) {

	if ok = v.Factorize(vcMatrix); !ok {
		return v, ok
	}

	// This can be commented out in production
	// Extract the factorization and check that it equals the original matrix.
	if *logger.OutputMode == "verbose" {
		var t mat.TriDense
		v.LTo(&t)
		var test mat.Dense
		test.Mul(&t, t.T())
		fmt.Println()
		fmt.Printf("check: L * L^T = \n         %0.4v\n", mat.Formatted(&test, mat.Prefix("          ")))
	}

	return v, ok
}

// Pretty matrix format printout
func MatPrint(X mat.Matrix) {
	fa := mat.Formatted(X, mat.Prefix(""), mat.Squeeze())
	fmt.Printf("%v\n", fa)
}

// Set covariance of all index components to zero except the bumped component
func ConstrainedFactor(c animal.BumpComponent_t) {

	// Still can't do this until all submatrices are PD
	return

	// genetic covaraince matrix
	thisTraitIndx := animal.GeneticIndex(c.TraitName, c.Component)
	_, cols := VcMatrix["genetic"].Dims()
	for _, x := range ecoIndex.IndexComponents {
		if !(c.TraitName == x.TraitName && c.Component == x.Component) {
			indx := animal.GeneticIndex(x.TraitName, x.Component)
			loc := Index2D(indx, thisTraitIndx, cols)
			Vc["genetic"].v[loc] = 0.0
			loc = Index2D(thisTraitIndx, indx, cols)
			Vc["genetic"].v[loc] = 0.0
		}
	}
	var ok bool
	VcMatrix["genetic"] = mat.NewSymDense(cols, Vc["genetic"].v)
	GvCholesky, ok = Factor("genetic", VcMatrix["genetic"])
	if !ok {
		logger.LogWriterFatal("Could not factorize genetic for " + c.TraitName + c.Component)
	}
	// Skip the residual because the submatrices are not pd
	return

	/*
		// residual covaraince matrix
		thisTraitIndx = animal.ResidualIndex(c.TraitName)
		_, cols = VcMatrix["residual"].Dims()
		for _, x := range ecoIndex.IndexComponents {
			if c.TraitName != x.TraitName {
				indx := animal.ResidualIndex(x.TraitName)
				loc := Index2D(indx, thisTraitIndx, cols)
				Vc["residual"].v[loc] = 0.0
				loc = Index2D(thisTraitIndx, indx, cols)
				Vc["residual"].v[loc] = 0.0
			}
		}

		VcMatrix["residual"] = mat.NewSymDense(cols, Vc["residual"].v)
		//MatPrint(VcMatrix["residual"])
		//RvCholesky = Factor("residual", VcMatrix["residual"])

		return
	*/
}

// Return the 1D index of a 2D matrix stored as array
func Index2D(row int, col int, dim int) (loc int) {
	loc = dim*col + row // reversed because this is CMO storage
	return
}

// Returns the genetic variance at component
func VarFromMatrix(loc int, vc mat.Symmetric) float64 {
	v := vc.At(loc, loc)
	return v
}
