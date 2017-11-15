package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"rsc.io/pdf"
)

func main() {
	file := os.Args[1]
	reader, err := pdf.Open(file)
	if err != nil {
		panic(err)
	}
	for i := 2; i < reader.NumPage()+1; i++ {
		page := reader.Page(i)
		fmt.Printf("Page %d\n", i)

		lines := make(map[float64][]pdf.Text)

		for _, t := range page.Content().Text {
			lines[t.Y] = append(lines[t.Y], t)
		}

		// Find lines that look like transactions
		txLines := make(map[float64]bool)
		for _, texts := range lines {
			for _, t := range texts {
				if t.X == 44.4 {
					txLines[t.Y] = true
				}
			}
		}

		findAmount := func(line float64) (s string) {
			for _, t := range lines[line] {
				if t.X > 50 && t.S != "" {
					s = s + strings.TrimSpace(t.S)
				}
			}
			return s
		}

		var sortedLines []float64
		for line := range txLines {
			sortedLines = append(sortedLines, line)
		}

		sort.Sort(sort.Reverse(sort.Float64Slice(sortedLines)))

		for _, line := range sortedLines {
			var s string
			for _, t := range lines[line] {
				s = s + t.S
			}
			if len(s) > 64 {
				s = s[:64]
			}
			amount, err := strconv.ParseFloat(findAmount(line), 64)
			if err != nil {
				continue
			}
			fmt.Printf("%-64s %5.2f\n", s, amount)
		}
	}

}
