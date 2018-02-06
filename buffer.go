package tester

import (
	"regexp"
	"time"
	"fmt"
	"strings"
	"strconv"
	"github.com/fatih/color"
)

var passRe = regexp.MustCompile("^--- PASS:")
var failRe = regexp.MustCompile("^--- FAIL:")
var runRe  = regexp.MustCompile("^=== RUN")

type buffer struct {
	passColor *color.Color
	failColor *color.Color
	
	bytes 	  []byte
	i 		  int
	pass      int
	fail      int
	lastlinei int
	lastruni  int
	timer 	  *time.Timer
}

func (b *buffer) Write (bytes []byte) (n int, err error) {
	b.timer.Reset(time.Second*10)

	for _,by := range bytes {
		if by == '\n' {
			line :=  b.bytes[b.lastlinei:]
			if passRe.Match(line) {
				b.i++
				b.pass++

				if b.passColor != nil {
					if b.i>1 { fmt.Print(strings.Repeat("\b", len(strconv.Itoa(b.i-1)) )) }

					fmt.Print(b.passColor.Sprint(b.i))
				}
			} else {
				if failRe.Match(line) {
					b.i++
					b.fail++

					if b.failColor != nil {
						if b.i>1 { fmt.Print(strings.Repeat("\b", len(strconv.Itoa(b.i-1)) )) }

						fmt.Print(b.failColor.Sprint(b.i))
					}
				} else {
					if runRe.Match(line) {
						b.lastruni = b.lastlinei
					}
				}
			}

			b.lastlinei = len(b.bytes)+1
		}

		b.bytes = append(b.bytes, by)
	}

	return len(bytes), nil
}
