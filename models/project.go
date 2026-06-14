package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
)

// GalleryItem représente un élément de la galerie média (image ou vidéo)
type GalleryItem struct {
	Type string `json:"type"` // "image" ou "video"
	Src  string `json:"src"`  // Chemin d'accès de l'asset
}

// Gallery est une liste d'éléments de galerie stockée sous forme de JSONB dans Postgres
type Gallery []GalleryItem

// Value implémente driver.Valuer pour la conversion en JSONB PostgreSQL
func (g Gallery) Value() (driver.Value, error) {
	if g == nil {
		return json.Marshal([]GalleryItem{})
	}
	return json.Marshal(g)
}

// Scan implémente sql.Scanner pour charger les données du JSONB PostgreSQL
func (g *Gallery) Scan(value interface{}) error {
	if value == nil {
		*g = Gallery{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("échec de conversion du type en []byte")
	}
	return json.Unmarshal(b, g)
}

// Project représente un élément du portfolio Epik Brand avec support multilingue
type Project struct {
	ID            int64    `json:"db_id"` // Identifiant de base de données (interne)
	ProjectID     string   `json:"id"`    // Identifiant unique côté client (ex: 'photo-1')
	
	// Champs dynamiques renvoyés au client selon la langue demandée
	Title         []string `json:"titre"`
	Description   string   `json:"description"`
	
	// Champs de traduction français (FR)
	TitleFRPart1  string   `json:"title_fr_part1"`
	TitleFRPart2  string   `json:"title_fr_part2"`
	DescriptionFR string   `json:"description_fr"`
	
	// Champs de traduction anglais (EN)
	TitleENPart1  string   `json:"title_en_part1"`
	TitleENPart2  string   `json:"title_en_part2"`
	DescriptionEN string   `json:"description_en"`
	
	// Champs de traduction malgache (MG)
	TitleMGPart1  string   `json:"title_mg_part1"`
	TitleMGPart2  string   `json:"title_mg_part2"`
	DescriptionMG string   `json:"description_mg"`
	
	MediaType     string   `json:"mediaType"`
	Src           string   `json:"src"`
	BgColor       string   `json:"bgColor"`
	Gallery       Gallery  `json:"galerie"`
	Category      string   `json:"category"` // "photo", "video", "graphique", "web"
}

// Localize peuple les champs 'Title' et 'Description' dynamiquement selon la langue cible
func (p *Project) Localize(lang string) {
	switch strings.ToLower(lang) {
	case "en":
		p.Title = []string{p.TitleENPart1}
		if p.TitleENPart2 != "" {
			p.Title = append(p.Title, p.TitleENPart2)
		}
		p.Description = p.DescriptionEN
		// Repli vers le français si l'anglais est manquant
		if len(p.Title) == 0 || p.Title[0] == "" {
			p.Title = []string{p.TitleFRPart1}
			if p.TitleFRPart2 != "" {
				p.Title = append(p.Title, p.TitleFRPart2)
			}
		}
		if p.Description == "" {
			p.Description = p.DescriptionFR
		}
	case "mg":
		p.Title = []string{p.TitleMGPart1}
		if p.TitleMGPart2 != "" {
			p.Title = append(p.Title, p.TitleMGPart2)
		}
		p.Description = p.DescriptionMG
		// Repli vers le français si le malgache est manquant
		if len(p.Title) == 0 || p.Title[0] == "" {
			p.Title = []string{p.TitleFRPart1}
			if p.TitleFRPart2 != "" {
				p.Title = append(p.Title, p.TitleFRPart2)
			}
		}
		if p.Description == "" {
			p.Description = p.DescriptionFR
		}
	default: // "fr" (par défaut)
		p.Title = []string{p.TitleFRPart1}
		if p.TitleFRPart2 != "" {
			p.Title = append(p.Title, p.TitleFRPart2)
		}
		p.Description = p.DescriptionFR
	}
}
