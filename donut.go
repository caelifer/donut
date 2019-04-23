package main

import (
	"flag"
	"fmt"
	_ "log"
	"math"
	"strings"
	"time"
)

type Frame string

func (f Frame) String() string {
	return string(f)
}

func trimAt(s string, pos int) string {
	t := []rune(s)
	n := len(t)

	if n > pos {
		return string(t[0:pos])
	} else {
		// Pad with blanks
		return s + strings.Repeat(" ", pos-n)
	}
}

func frameLine(s string, chr rune) string {
	t := []rune(s)

	// Replace first
	t[0] = chr

	// Replace last
	t[len(t)-1] = chr

	return string(t)
}

// Frame lines
const (
	top  = "┌——————————————————————————————————————————————————————————————┬———————————————┐"
	side = '│'
	bttm = "└——————————————————————————————————————————————————————————————————————————————┘"
)

func MakeFrame(pic []rune) Frame {
	lines := strings.Split(string(pic), "\n")

	frameWidth := len([]rune(top))

	// Frame each line
	for i, _ := range lines {
		// Make sure line is frameWidth columns long
		nl := trimAt(lines[i], frameWidth)

		// Blit with side glyphs
		nl = frameLine(nl, side)

		// Special case for line 4
		if i == 4 {
			t := []rune(nl)
			t[79] = '┤'
			nl = string(t)
		}

		lines[i] = nl
	}

	// Add top frame line
	lines = append([]string{top}, lines...) // Poor man's push
	// Add bottom frame line
	lines = append(lines, bttm)

	return Frame(strings.Join(lines, "\n"))
}

func _f(b []rune, ch rune) int {
	max := len(b)
	i := 0
	for ; i < max; i++ {
		if b[i] == ch {
			break
		}
	}
	return i
}
func splice(b []rune, yidx, offs int, val []rune) {
	maxx := _f(b, '\n') + 1
	for i, v := range val {
		if i+offs > maxx {
			break
		}
		b[yidx*maxx+offs+i] = v
	}
}

func normalizeDegrees(degree float64) float64 {
	periods := int(degree / 360)
	return degree - float64(periods*360)
}

const radToDegree = 180 / math.Pi

const FBSZ = 80 * 22

// Preallocate
var frameBuffer = [2][FBSZ]rune{
	[FBSZ]rune{},
	[FBSZ]rune{},
}

var fbIdx = 0

// Torus matrix with light coefficients
var z = make([]float64, FBSZ)

func donut(b []rune, aspectA, aspectB float64, stream chan<- Frame) {
	var A, B float64 = aspectA, aspectB

	// stats
	var fps float64  // frames per second
	frameN := 0      // frame count
	t0 := time.Now() // start time

	// Forever
	for {
		// Rotational parameters
		A += 0.07 // Yaw angle
		B += 0.03 // Roll angle

		sA, cA := math.Sincos(A)
		sB, cB := math.Sincos(B)

		// Blank frame
		for k := 0; k < len(b); k++ {
			if k%80 == 79 {
				b[k] = '\n'
			} else {
				b[k] = ' '
			}
			z[k] = 0 // Zero-out torus matrix
		}

		// Draw torus with rotational aspect

		// x(theta, phy) = (R + r * cos(phy) * cos(theta)
		// y(theta, phy) = (R + r * cos(phy) * sin(theta)
		// z(theta, phy) = r * sin(phy)

		// theta, phy - angles from 0 to 2 * PI
		// R          - major radius
		// r          - minor radius

		// Theta
		for j := float64(0); j < 6.28; j += 0.07 {
			// sin and cos of theta
			st, ct := math.Sincos(j)

			// Phy
			for i := float64(0); i < 6.28; i += 0.02 {

				// Unknown =  2 + cos(theta)
				h := ct + 2

				// sin and cos of phy
				sp, cp := math.Sincos(i)

				// Unknown =  1 / [ sin(phy) * (2 + cos(theta)) * sin(A) + sin(theta) * cos(A) + 5 ]
				// guess - value for our torus matrix as it rotates in 3-D space
				D := 1 / (sp*h*sA + st*cA + 5)

				// Unknown =  sin(phy) * (2 + cos(theta)) * cos(A) - sin(theta) * sin(A)
				t := sp*h*cA - st*sA

				// X-axis coordinate
				x := int(40 + 30*D*(cp*h*cB-t*sB))

				// Y-axis coordinate
				y := int(12 + 15*D*(cp*h*sB+t*cB))

				// Frame pixel index
				o := x + 80*y

				// Light coeficient (grey-scale index)
				N := int(8 * ((st*sA-sp*ct*cA)*cB - sp*ct*sA - st*cA - cp*ct*sB))

				// Draw if we are within frame and pixel is visible
				if y < 22 && y >= 0 && x >= 0 && x < 79 && D > z[o] {
					// Fill
					z[o] = D
					n := 0
					if N > 0 {
						n = N
					}

					// Fill the pixel on the canvas (frame)
					b[o] = []rune(".,-~:;=!*#$@")[n]
					// b[o] = []rune("∙◦▪●☼◊≠≡☺♦☻◙")[n]
				}
			}
		}
		// Count frames
		frameN++

		// Calculate frame per second rate
		fps = float64(frameN) / time.Since(t0).Seconds()

		// display stats
		var (
			frame = []rune(fmt.Sprintf("│ Frame: %5d", frameN))
			rate  = []rune(fmt.Sprintf("│   FPS: %5.1f", fps))
			roll  = []rune(fmt.Sprintf("│  Roll: %5.1f˚", normalizeDegrees(B*radToDegree)))
			yaw   = []rune(fmt.Sprintf("│   Yaw: %5.1f˚", normalizeDegrees(A*radToDegree)))
			bttm  = []rune("└———————————————")
			offs  = 77 - len(frame)
		)
		splice(b, 0, offs, frame)
		splice(b, 1, offs, rate)
		splice(b, 2, offs, roll)
		splice(b, 3, offs, yaw)
		splice(b, 4, offs, bttm)

		// Send frame
		stream <- MakeFrame(b)
	}
}

func genFrameStream(f func([]rune, float64, float64, chan<- Frame)) <-chan Frame {
	stream := make(chan Frame, 1) // Always a room for the next frame

	go func() {
		fbIdx++
		f(frameBuffer[fbIdx%2][:], 0, 0, stream)
	}()

	return stream
}

// Define CLI flags
var (
	runTimeDuration = flag.Duration("d", 5*time.Second, "a run duration")
)

func main() {
	// Initialize flags
	flag.Parse()

	frames := genFrameStream(donut)
	timeout := time.After(*runTimeDuration)

	for {
		select {
		case <-timeout:
			return
		case frame := <-frames:
			fmt.Print("\033[H\033[2J")        // Clear screen
			fmt.Print("\x0c", frame, "\n")    // Print frame
			time.Sleep(30 * time.Millisecond) // Delay between frames
		}
	}
}
