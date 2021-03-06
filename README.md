# iGenDec
iGenDec is web based software that calculates the marginal economic values (MEV) of a selection index suitable for making breeding stock purchase decisions in North American commercial beef cattle production systems.  It uses a simluation model to estimate the second derivitives of the profit function (MEV) for a specific production situation.  The user, supported by a trained facilitator, provides parameters that reflect a commercial cattle operation.  iGenDec then provides a selection index that can be used to value alternatives bull, semen, heifer or embryo purchases.

## The model and starter
The iGenDecModel repository contains the go language code for two of the three components for a web based installation.  In the iGenDecModel/iGenDec directory is the simulation model source code with a few example human readable hjson files. In the iGenDec/starter directory is the starter program which is a wrapper written in go to make multiple runs of the iGenDec model, bumping the bull breeding values of the index components by 1 unit one at a time to estimate the MEV. 

The resulting MEV are suitable for application to expected progeny differences (EPD).

## The web frontend
A third component of iGenDec is the web interaface application maintained in a separate repository, https://github.com/blgolden/igendec. 

View the frontend's README.md for instructions on a complete installation and information on the hjson files.
