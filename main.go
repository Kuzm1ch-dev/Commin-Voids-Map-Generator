package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"time"
)

func main() {
	optSize := flag.Int("size", 32, "Image size, must be power of two.")
	optSamples := flag.Int("samples", 16, "Number of samples. Lower is more rough terrain. Must be power of two.")
	optBlur := flag.Int("blur", 1, "Blur/Smooth size. Somewhere between 1 and 5.")
	optScale := flag.Float64("scale", 1, "Scale. Most lileky 1.0")
	optOut := flag.String("o", "", "Output png filename.")
	flag.Parse()

	if *optOut == "" {
		flag.Usage()
		os.Exit(1)
	}

	h := NewHeightmap(*optSize)
	h.generate(*optSamples, *optScale)
	h.blur(*optBlur)
	h.normalize()
	h.ladderGenerate()
	h.png(*optOut)
}

// Heightmap generates a heightmap based on the Square Diamond algorithm.
type Heightmap struct {
	random         *rand.Rand
	points         []float64
	ladders        []int32
	blockStep      int
	laddersOnBlock int
	y              float64
	width, height  int
}

type generator interface {
	Generate()
}

// NewHeightmap initializes a new Heightmap using the specified size.
func NewHeightmap(size int) *Heightmap {
	h := &Heightmap{}
	h.random = rand.New(rand.NewSource(time.Now().Unix()))
	h.points = make([]float64, size*size)
	h.ladders = make([]int32, size*size)
	h.width = size
	h.height = size
	h.y = 4
	h.blockStep = 8
	h.laddersOnBlock = 2

	// init/randomize
	for x := 0; x < h.width; x++ {
		for y := 0; y < h.height; y++ {
			h.set(x, y, h.frand())
		}
	}
	return h
}

func (h *Heightmap) png(fname string) {
	rect := image.Rect(0, 0, h.width, h.height)
	img := image.NewRGBA(rect)

	for x := 0; x < h.width; x++ {
		for y := 0; y < h.height; y++ {
			val := h.get(x, y)
			col := color.Gray16{uint16(val * 0xffff)}
			img.Set(x, y, col)
		}
	}

	for x := 0; x < h.width; x++ {
		for y := 0; y < h.height; y++ {
			val := h.ladders[(x&(h.width-1))+((y&(h.height-1))*h.width)]
			if val == 1 {
				col := color.RGBA{255, 0, 0, 255}
				img.Set(x, y, col)
			}
		}
	}

	f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Generated image to " + fname)
}

func (h *Heightmap) normalize() {
	var min = 1.0
	var max = 0.0

	for i := 0; i < h.width*h.height; i++ {
		if h.points[i] < min {
			min = h.points[i]
		}
		if h.points[i] > max {
			max = h.points[i]
		}
	}
	rat := max - min
	for i := 0; i < h.width*h.height; i++ {
		h.points[i] = Round((h.points[i]-min)/rat, float64(1.0/h.y))
		/*
			if h.points[i] > .6 {
				h.points[i] = 1
			} else {
				h.points[i] = 0
			}*/
	}

}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}

func (h *Heightmap) ladderGenerate() {
	//Пробегаем по массиву квадратами по 8
	for i := 0; i < int(h.width/h.blockStep); i++ {
		for j := 0; j < int(h.height/h.blockStep); j++ {
			k := 0
			for p := 0; p < h.blockStep*h.blockStep; p++ {
				index := (i * h.blockStep) + (j * h.width * h.blockStep) + (int(p/h.blockStep) * h.width) - (h.blockStep * (int(p / h.blockStep))) + p

				if index != 0 && index != (h).width-1 {
					if (h.points[index-1] != h.points[index]) && (h.points[index] != 0) && (h.points[index-1] != 0) {
						h.ladders[index] = 1
						k++
						if k >= h.laddersOnBlock {
							k = 0
							break
						}
					}
				}
			}
		}
	}
}

func (h *Heightmap) blur(size int) {
	for x := 0; x < h.width; x++ {
		for y := 0; y < h.height; y++ {
			count := 0
			total := 0.0

			for x0 := x - size; x0 <= x+size; x0++ {
				for y0 := y - size; y0 <= y+size; y0++ {
					total += h.get(x0, y0)
					count++
				}
			}
			if count > 0 {
				h.set(x, y, total/float64(count))
			}
		}
	}
}

func (h *Heightmap) frand() float64 {
	return (h.random.Float64() * 2.0) - 1.0
}

func (h *Heightmap) get(x, y int) float64 {
	return h.points[(x&(h.width-1))+((y&(h.height-1))*h.width)]
}

func (h *Heightmap) set(x, y int, val float64) {
	h.points[(x&(h.width-1))+((y&(h.height-1))*h.width)] = val
}

func (h *Heightmap) generate(samples int, scale float64) {
	for samples > 0 {
		h.squarediamond(samples, scale)
		samples /= 2
		scale /= 2.0
	}
}

func (h *Heightmap) squarediamond(step int, scale float64) {
	half := step / 2
	for y := half; y < h.height+half; y += step {
		for x := half; x < h.width+half; x += step {
			h.square(x, y, step, h.frand()*scale)
		}
	}
	for y := 0; y < h.height; y += step {
		for x := 0; x < h.width; x += step {
			h.diamond(x+half, y, step, h.frand()*scale)
			h.diamond(x, y+half, step, h.frand()*scale)
		}
	}
}

func (h *Heightmap) square(x, y, size int, val float64) {
	half := size / 2
	a := h.get(x-half, y-half)
	b := h.get(x+half, y-half)
	c := h.get(x-half, y+half)
	d := h.get(x+half, y+half)
	h.set(x, y, ((a+b+c+d)/4.0)+val)
}

func (h *Heightmap) diamond(x, y, size int, val float64) {
	half := size / 2
	a := h.get(x-half, y)
	b := h.get(x+half, y)
	c := h.get(x, y-half)
	d := h.get(x, y+half)
	h.set(x, y, ((a+b+c+d)/4.0)+val)
}
