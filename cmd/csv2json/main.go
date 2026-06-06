package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type Category struct {
	ID       string     `json:"id"`
	LabelDE  string     `json:"label_de"`
	LabelEN  string     `json:"label_en"`
	Subthema []Category `json:"subthema,omitzero"`
}

func main() {
	file, err := os.Open("data/thema_label.csv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Öffnen der CSV-Datei: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Überspringe Header
	if _, err := reader.Read(); err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Lesen des Headers: %v\n", err)
		os.Exit(1)
	}

	var categories []Category
	parentMap := make(map[string]*Category)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler beim Lesen der Zeile: %v\n", err)
			os.Exit(1)
		}

		class := record[0]
		labelDE := record[1]
		labelEN := record[2]

		parts := strings.Split(class, "-")
		if len(parts) != 2 {
			continue
		}

		parentCode := parts[0]
		subCode := parts[1]

		cat := Category{
			ID:      class,
			LabelDE: labelDE,
			LabelEN: labelEN,
		}

		if subCode == "00" {
			// Es ist ein Hauptpunkt
			categories = append(categories, cat)
			parentMap[parentCode] = &categories[len(categories)-1]
		} else {
			// Es ist ein Unterpunkt
			if parent, ok := parentMap[parentCode]; ok {
				parent.Subthema = append(parent.Subthema, cat)
			} else {
				// Falls der Parent noch nicht existiert (sollte laut CSV nicht vorkommen, aber sicherheitshalber)
				// Wir fügen es vorerst als Root-Level hinzu oder loggen eine Warnung
				fmt.Printf("Warnung: Kein Parent für %s gefunden\n", class)
				categories = append(categories, cat)
			}
		}
	}

	output, err := json.MarshalIndent(categories, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Erstellen von JSON: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("data/thema_label.json", output, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Schreiben der JSON-Datei: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Erfolgreich nach data/thema_label.json konvertiert")
}
