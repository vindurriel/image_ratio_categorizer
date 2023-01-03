package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	_ "image/png"
	"math"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
)

var standardMap = map[string]float64{
	"16x9": 16.0 / 9.0,
	"3x2":  3.0 / 2.0,
	"4x3":  4.0 / 3.0,
	"5x4":  5.0 / 4.0,
	"3x3":  1.0,
}

type p = image.Point

var resize = p{X: 1800, Y: 1200}

type paddingConf struct {
	ratioName string
	fill      image.Image
}

var paddingMap = map[string]*paddingConf{
	"16x9": {
		ratioName: "3x2",
		fill:      paddingBlack,
	},
	"5x4": {
		ratioName: "3x2",
		fill:      paddingBlack,
	},
	"4x3": {
		ratioName: "3x2",
		fill:      paddingBlack,
	},
	"3x3": {
		ratioName: "3x2",
		fill:      paddingWhite,
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("input a dir name")
		os.Exit(1)
	}
	dir := path.Clean(os.Args[1])
	var files []string
	var err error
	if files, err = getImageFiles(dir); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if len(files) == 0 {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(files))
	concurrency := runtime.NumCPU()
	pipes := make([]chan string, concurrency, concurrency)
	for i := 0; i < concurrency; i++ {
		pipes[i] = make(chan string, 0)
		go func(i int) {
			var err error
			for f := range pipes[i] {
				if err = do(path.Join(dir, f)); err != nil {
					fmt.Printf("[E] %v\n", err)
				}
				wg.Done()
			}
		}(i)
	}
	for i, f := range files {
		pipes[i%concurrency] <- f
		fmt.Printf("[I] progress: %v / %v\n", i+1, len(files))
	}
	wg.Wait()
}

func getImageFiles(dir string) (files []string, err error) {
	var entries []os.DirEntry
	if entries, err = os.ReadDir(dir); err != nil {
		return
	}
	extMap := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	for _, f := range entries {
		ext := strings.ToLower(path.Ext(f.Name()))
		if !f.IsDir() && extMap[ext] {
			files = append(files, f.Name())
		}
	}
	return
}

var paddingWhite = image.NewUniform(color.White)
var paddingBlack = image.NewUniform(color.Black)
var zeroPoint p
var resizeFilter = imaging.Lanczos

func do(inputFile string) (err error) {
	f, err := os.Open(inputFile)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	src, _, err := image.Decode(f)
	dir, name := path.Split(inputFile)
	if err != nil {
		return fmt.Errorf("%v decode err: %w", name, err)
	}
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if width < height {
		src = imaging.Rotate90(src)
		width, height = height, width
		fmt.Printf("[I] %v rotate 90\n", name)
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
	var dst draw.Image
	if conf := paddingMap[ratioName]; conf != nil {
		// add padding
		toRatio := standardMap[conf.ratioName]
		toWidth, toHeight := width, height
		keepWidth := toRatio < ratio
		if keepWidth {
			toHeight = int(math.Ceil(float64(width) / toRatio))
		} else {
			toWidth = int(math.Ceil(float64(height) * toRatio))
		}
		dst = image.NewNRGBA(image.Rect(0, 0, toWidth, toHeight))
		padding := conf.fill
		if keepWidth {
			draw.Draw(dst, src.Bounds(), src, zeroPoint, draw.Src)
			draw.Draw(dst, image.Rect(0, height, toWidth, toHeight), padding, zeroPoint, draw.Src)
		} else {
			draw.Draw(dst, src.Bounds(), src, zeroPoint, draw.Src)
			draw.Draw(dst, image.Rect(width, 0, toWidth, toHeight), padding, zeroPoint, draw.Src)
		}
		fmt.Printf("[I] %v padded, ratioName: %v, from: %v %v, to: %v %v, keepWidth: %v\n",
			name, ratioName, width, height, toWidth, toHeight, keepWidth)
	}

	// resize
	if dst == nil {
		// not padded, resize from source
		dst = imaging.Resize(src, resize.X, resize.Y, resizeFilter)
		fmt.Printf("[I] %v resized, ratioName: %v, from: %v %v\n",
			name, ratioName, width, height)
	} else {
		//  padded, resize from dst
		dst = imaging.Resize(dst, resize.X, resize.Y, resizeFilter)
	}

	outputFile := path.Join(dir, fmt.Sprintf("%v.%v", ratioName, name))
	var o *os.File
	if o, err = os.Create(outputFile); err != nil {
		return
	}
	defer func() { _ = o.Close() }()
	ext := strings.ToLower(path.Ext(inputFile))
	switch ext {
	case ".jpg":
		fallthrough
	case ".jpeg":
		err = jpeg.Encode(o, dst, &jpeg.Options{Quality: 100})
	case ".png":
		err = png.Encode(o, dst)
	default:
		return errors.New(fmt.Sprintf("invalid ext: %v", ext))
	}
	if err != nil {
		return fmt.Errorf("write file err: %w", err)
	}
	return
}
