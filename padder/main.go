package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	_ "image/png"
	"math"
	"os"
	"path"
	"strings"
)

var standardMap = map[string]float64{
	"16x9": 16.0 / 9.0,
	"3x2":  3.0 / 2.0,
	"4x3":  4.0 / 3.0,
	"5x4":  5.0 / 4.0,
	"3x3":  1.0,
}

var ratioMap = map[string]string{
	"16x9": "4x3",
	"5x4":  "4x3",
	"3x3":  "4x3",
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("input an file name")
		os.Exit(1)
	}
	inputFile := os.Args[1]
	f, err := os.Open(inputFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer func() { _ = f.Close() }()
	src, _, err := image.Decode(f)
	if err != nil {
		fmt.Println("decode", err)
		os.Exit(3)
	}
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if width < height {
		src = imaging.Rotate90(src)
		width, height = height, width
	}
	var ratioName string
	var min = math.MaxFloat64
	ratio := float64(width) / float64(height)
	for k, v := range standardMap {
		diff := math.Abs(ratio - v)
		if diff < min {
			min = diff
			ratioName = k
		}
	}
	dir, name := path.Split(inputFile)
	toRatioName := ratioMap[ratioName]
	var dst draw.Image
	var zeroPoint image.Point
	if toRatioName == "" {
		dst = image.NewNRGBA(src.Bounds())
		draw.Draw(dst, src.Bounds(), src, zeroPoint, draw.Src)
		toRatioName = ratioName
		fmt.Println(name, "ratioName:", ratioName,
			"oldSize:", height, width,
			"newSize:", width, height)
	} else {
		toRatio := standardMap[toRatioName]
		toWidth, toHeight := width, height
		keepWidth := toRatio < ratio
		var offset1, offset2 int
		if keepWidth {
			toHeight = int(math.Ceil(float64(width) / toRatio))
			offset1 = (toHeight - height) / 2
			offset2 = height + offset1
		} else {
			toWidth = int(math.Ceil(float64(height) * toRatio))
			offset1 = (toWidth - width) / 2
			offset2 = width + offset1
		}
		fmt.Println(name, "ratioName:", ratioName, "keepWidth:", keepWidth,
			"oldSize:", width, height,
			"newSize:", toWidth, toHeight)
		dst = image.NewNRGBA(image.Rect(0, 0, toWidth, toHeight))
		padding := image.NewUniform(color.Black)
		if keepWidth {
			draw.Draw(dst, image.Rect(0, 0, toWidth, offset1), padding, zeroPoint, draw.Src)
			draw.Draw(dst, src.Bounds().Add(image.Point{X: 0, Y: offset1}), src, zeroPoint, draw.Src)
			draw.Draw(dst, image.Rect(0, offset2, toWidth, toHeight), padding, zeroPoint, draw.Src)
		} else {
			draw.Draw(dst, image.Rect(0, 0, offset1, toHeight), padding, zeroPoint, draw.Src)
			draw.Draw(dst, src.Bounds().Add(image.Point{X: offset1, Y: 0}), src, zeroPoint, draw.Src)
			draw.Draw(dst, image.Rect(offset2, 0, toWidth, toHeight), padding, zeroPoint, draw.Src)
		}
	}
	outputFile := path.Join(dir, fmt.Sprintf("%v.%v", toRatioName, name))
	o, err := os.Create(outputFile)
	ext := strings.ToLower(path.Ext(inputFile))
	defer func() { _ = o.Close() }()
	switch ext {
	case ".jpg":
		err = jpeg.Encode(o, dst, &jpeg.Options{Quality: 100})
	case ".png":
		err = png.Encode(o, dst)
	default:
		fmt.Println("invalid ext:", ext)
		os.Exit(4)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(5)
	}
}
