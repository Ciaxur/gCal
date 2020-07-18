package main

import "github.com/fatih/color"

// Output Structure for Color Schemes
type Output struct {
	Error   *color.Color
	Warning *color.Color
	Info    *color.Color
}

// InitOutput - Initializes an Output Object with Preset Colors
func InitOutput() *Output {
	return &Output{
		Error:   color.New(color.FgRed).Add(color.Bold),
		Warning: color.New(color.FgHiMagenta).Add(color.Bold),
		Info:    color.New(color.FgHiGreen).Add(color.Bold),
	}
}

// Out - GLOBAL VARIABLE FOR PRINTS
var Out *Output = InitOutput()
