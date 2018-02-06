package tester

import "github.com/fatih/color"

type ColorSheme struct {
	Folder 		*color.Color
	TestName	*color.Color
	Pass		*color.Color
	Fail		*color.Color
	Ignore 		*color.Color
}

var DefaultColorSheme = &ColorSheme{
	Folder: 	color.New(color.FgYellow),
	TestName:	color.New(color.FgHiWhite),
	Pass:		color.New(color.FgGreen, color.Bold),
	Fail:		color.New(color.FgRed, color.Bold),
	Ignore: 	color.New(color.FgHiBlack, color.Bold),
}