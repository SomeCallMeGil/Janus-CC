// Package filler generates lorem ipsum filler data
package filler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// Lorem ipsum word bank
var loremWords = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
	"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore",
	"magna", "aliqua", "enim", "ad", "minim", "veniam", "quis", "nostrud",
	"exercitation", "ullamco", "laboris", "nisi", "aliquip", "ex", "ea", "commodo",
	"consequat", "duis", "aute", "irure", "in", "reprehenderit", "voluptate",
	"velit", "esse", "cillum", "fugiat", "nulla", "pariatur", "excepteur", "sint",
	"occaecat", "cupidatat", "non", "proident", "sunt", "culpa", "qui", "officia",
	"deserunt", "mollit", "anim", "id", "est", "laborum",
}

// Generator generates filler data
type Generator struct {
	rng *rand.Rand
}

// New creates a new filler generator
func New(seed int64) *Generator {
	var rng *rand.Rand
	if seed != 0 {
		rng = rand.New(rand.NewSource(seed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	
	return &Generator{rng: rng}
}

// GenerateFile generates a filler file of the specified format and target size
func (g *Generator) GenerateFile(path string, format string, targetSize int64) error {
	switch format {
	case "txt":
		return g.GenerateTxt(path, targetSize)
	case "csv":
		return g.GenerateCsv(path, targetSize)
	case "json":
		return g.GenerateJson(path, targetSize)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// GenerateTxt generates a text file with lorem ipsum paragraphs
func (g *Generator) GenerateTxt(path string, targetSize int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	written := int64(0)
	
	for written < targetSize {
		// Generate a paragraph
		wordCount := g.rng.Intn(50) + 30 // 30-80 words per paragraph
		paragraph := g.generateParagraph(wordCount)
		
		// Write paragraph
		n, err := f.WriteString(paragraph + "\n\n")
		if err != nil {
			return err
		}
		
		written += int64(n)
		
		// Safety: don't go more than 10% over target
		if written > targetSize*11/10 {
			break
		}
	}
	
	return nil
}

// GenerateCsv generates a CSV file with lorem ipsum data
func (g *Generator) GenerateCsv(path string, targetSize int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	w := csv.NewWriter(f)
	defer w.Flush()
	
	// Generate column headers
	columnCount := g.rng.Intn(5) + 3 // 3-8 columns
	headers := make([]string, columnCount)
	for i := 0; i < columnCount; i++ {
		headers[i] = g.generateWord()
	}
	
	if err := w.Write(headers); err != nil {
		return err
	}
	
	// Estimate bytes per row (rough)
	bytesPerRow := int64(columnCount * 15) // ~15 chars per column
	estimatedRows := int(targetSize / bytesPerRow)
	
	for rowCount := 0; rowCount < estimatedRows; rowCount++ {
		row := make([]string, columnCount)
		for i := 0; i < columnCount; i++ {
			// Mix of words and numbers
			if g.rng.Float64() < 0.3 {
				// Number
				row[i] = fmt.Sprintf("%d", g.rng.Intn(10000))
			} else {
				// Lorem word
				row[i] = g.generateWord()
			}
		}
		
		if err := w.Write(row); err != nil {
			return err
		}
	}
	
	return nil
}

// GenerateJson generates a JSON file with lorem ipsum data
func (g *Generator) GenerateJson(path string, targetSize int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	// Estimate number of objects needed
	bytesPerObject := int64(200) // rough estimate
	objectCount := int(targetSize / bytesPerObject)
	
	if objectCount < 1 {
		objectCount = 1
	}
	
	// Generate array of objects
	objects := make([]map[string]interface{}, objectCount)
	
	for i := 0; i < objectCount; i++ {
		objects[i] = g.generateJsonObject()
	}
	
	// Encode as pretty JSON
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(objects)
}

// generateParagraph generates a lorem ipsum paragraph
func (g *Generator) generateParagraph(wordCount int) string {
	words := make([]string, wordCount)
	
	for i := 0; i < wordCount; i++ {
		words[i] = g.generateWord()
	}
	
	// Capitalize first word
	if len(words) > 0 {
		words[0] = strings.Title(words[0])
	}
	
	// Join and add period
	return strings.Join(words, " ") + "."
}

// generateWord gets a random lorem ipsum word
func (g *Generator) generateWord() string {
	return loremWords[g.rng.Intn(len(loremWords))]
}

// generateJsonObject generates a random JSON object with lorem ipsum values
func (g *Generator) generateJsonObject() map[string]interface{} {
	obj := make(map[string]interface{})
	
	// Random number of fields (3-8)
	fieldCount := g.rng.Intn(6) + 3
	
	for i := 0; i < fieldCount; i++ {
		key := g.generateWord()
		
		// Random value type
		switch g.rng.Intn(5) {
		case 0: // String
			obj[key] = g.generateSentence(5, 15)
		case 1: // Number
			obj[key] = g.rng.Intn(10000)
		case 2: // Float
			obj[key] = g.rng.Float64() * 1000
		case 3: // Boolean
			obj[key] = g.rng.Float64() < 0.5
		case 4: // Array of words
			arraySize := g.rng.Intn(5) + 1
			arr := make([]string, arraySize)
			for j := 0; j < arraySize; j++ {
				arr[j] = g.generateWord()
			}
			obj[key] = arr
		}
	}
	
	// Add common fields for structure
	obj["id"] = g.rng.Intn(100000)
	obj["timestamp"] = time.Now().Add(-time.Duration(g.rng.Intn(365*24)) * time.Hour).Format(time.RFC3339)
	obj["type"] = g.generateWord()
	
	return obj
}

// generateSentence generates a sentence with N words
func (g *Generator) generateSentence(minWords, maxWords int) string {
	wordCount := g.rng.Intn(maxWords-minWords) + minWords
	words := make([]string, wordCount)
	
	for i := 0; i < wordCount; i++ {
		words[i] = g.generateWord()
	}
	
	// Capitalize first word
	if len(words) > 0 {
		words[0] = strings.Title(words[0])
	}
	
	return strings.Join(words, " ")
}

// GenerateToSize generates filler content to exactly meet target size
// This is more precise than GenerateFile but slower
func (g *Generator) GenerateToSize(path string, format string, targetSize int64) error {
	// First pass: generate approximately
	if err := g.GenerateFile(path, format, targetSize); err != nil {
		return err
	}
	
	// Check actual size
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	
	actualSize := info.Size()
	
	// If we're within 10%, good enough
	diff := actualSize - targetSize
	if diff < 0 {
		diff = -diff
	}
	
	tolerance := targetSize / 10 // 10%
	if diff <= tolerance {
		return nil
	}
	
	// If too small, append more data
	if actualSize < targetSize {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		
		needed := targetSize - actualSize
		
		switch format {
		case "txt":
			// Add paragraphs
			for needed > 0 {
				para := g.generateParagraph(50)
				n, _ := f.WriteString("\n\n" + para)
				needed -= int64(n)
			}
		case "csv", "json":
			// These formats are harder to append to correctly
			// Just regenerate if we need precision
			return g.GenerateFile(path, format, targetSize)
		}
	}
	
	// If too large, truncate
	if actualSize > targetSize {
		return os.Truncate(path, targetSize)
	}
	
	return nil
}
