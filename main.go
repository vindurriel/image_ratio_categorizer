package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("input an dir")
		os.Exit(1)
	}
	var files []os.FileInfo
	var err error
	dirname := os.Args[1]
	if err = os.Chdir(dirname); err != nil {
		fmt.Printf("Chdir: %v", err)
		os.Exit(2)
	}
	if files, err = ioutil.ReadDir("."); err != nil {
		fmt.Printf("ReadDir: %v", err)
		os.Exit(3)
	}
	var ratio float64
	var standardMap = map[string]float64{
		"16x9": 16.0 / 9.0,
		"3x2":  3.0 / 2.0,
		"4x3":  4.0 / 3.0,
		"5x4":  5.0 / 4.0,
		"1x1":  1.0,
	}
	var ext string
	extMap := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext = strings.ToLower(path.Ext(f.Name()))
		if !extMap[ext] {
			continue
		}
		if ratio, err = getImageRatio(f.Name()); err != nil {
			fmt.Println(err)
			os.Exit(4)
		}
		var res string
		var min = math.MaxFloat64
		for k, v := range standardMap {
			diff := math.Abs(ratio - v)
			if diff < min {
				min = diff
				res = k
			}
		}
		fmt.Printf("mv '%s' %s/\n", f.Name(), res)
	}
}

func getImageRatio(imagePath string) (ratio float64, err error) {
	var file *os.File
	if file, err = os.Open(imagePath); err != nil {
		return 0, errors.Wrapf(err, "open file %v", imagePath)
	}
	var c image.Config
	if c, _, err = image.DecodeConfig(file); err != nil {
		return 0, errors.Wrapf(err, "decode config %v", imagePath)
	}
	width, height := float64(c.Width), float64(c.Height)
	if width*height == 0 {
		return 0, fmt.Errorf("invalid image dimension %v", imagePath)
	}
	if width < height {
		width, height = height, width
	}
	return width / height, nil
}
