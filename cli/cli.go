package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"epik-brand-backend/config"
	"epik-brand-backend/db"
	"epik-brand-backend/models"
	"epik-brand-backend/server"
)

// RunAdminCLI lance la boucle d'interaction textuelle pour l'administration
func RunAdminCLI(cfg *config.Config, database *sql.DB) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\n==================================================")
		fmt.Println("      EPIK BRAND RELOOK - PORTFOLIO ADMIN")
		fmt.Println("==================================================")
		fmt.Println("1. Ajouter un nouveau projet au portfolio")
		fmt.Println("2. Afficher la liste des projets existants")
		fmt.Println("3. Supprimer un projet")
		fmt.Println("4. Démarrer le serveur API REST")
		fmt.Println("5. Quitter le logiciel")
		fmt.Println("==================================================")
		
		choix := readLine(scanner, "Choisissez une option (1-5) : ", true)

		switch choix {
		case "1":
			ajouterProjetInteractif(scanner, database)
		case "2":
			listerProjets(database)
		case "3":
			supprimerProjetInteractif(scanner, database)
		case "4":
			fmt.Println("\n[INFO] Démarrage du serveur API...")
			err := server.StartServer(cfg, database)
			if err != nil {
				log.Printf("Erreur fatale du serveur: %v\n", err)
			}
			return
		case "5":
			fmt.Println("Au revoir !")
			return
		default:
			fmt.Println("Option invalide, veuillez choisir un chiffre entre 1 et 5.")
		}
	}
}

// readLine lit une ligne entière depuis l'entrée standard
func readLine(scanner *bufio.Scanner, prompt string, required bool) string {
	for {
		fmt.Print(prompt)
		if !scanner.Scan() {
			return ""
		}
		text := strings.TrimSpace(scanner.Text())
		if required && text == "" {
			fmt.Println("-> Erreur: Ce champ est requis.")
			continue
		}
		return text
	}
}

// ajouterProjetInteractif demande pas à pas les informations du projet dans toutes les langues
func ajouterProjetInteractif(scanner *bufio.Scanner, database *sql.DB) {
	fmt.Println("\n--- AJOUT D'UN PROJET ---")

	projectID := readLine(scanner, "ID unique (ex: photo-5, web-4) : ", true)

	// --- SAISIE DES TITRES ---
	fmt.Println("\n-- TITRES --")
	titleFRPart1 := readLine(scanner, "Titre (Français) - Partie 1 (ex: Atelier chez) : ", true)
	titleFRPart2 := readLine(scanner, "Titre (Français) - Partie 2 (ex: Zina - optionnel) : ", false)

	titleENPart1 := readLine(scanner, "Titre (Anglais) - Partie 1 (laisser vide pour copier le Français) : ", false)
	var titleENPart2 string
	if titleENPart1 != "" {
		titleENPart2 = readLine(scanner, "Titre (Anglais) - Partie 2 (optionnel) : ", false)
	} else {
		titleENPart1 = titleFRPart1
		titleENPart2 = titleFRPart2
	}

	titleMGPart1 := readLine(scanner, "Titre (Malagasy) - Partie 1 (laisser vide pour copier le Français) : ", false)
	var titleMGPart2 string
	if titleMGPart1 != "" {
		titleMGPart2 = readLine(scanner, "Titre (Malagasy) - Partie 2 (optionnel) : ", false)
	} else {
		titleMGPart1 = titleFRPart1
		titleMGPart2 = titleFRPart2
	}

	// Sélection Catégorie
	var category string
	for {
		fmt.Println("\nCatégories :")
		fmt.Println("1. Photographie (photo)")
		fmt.Println("2. Vidéo (video)")
		fmt.Println("3. Graphique design (graphique)")
		fmt.Println("4. Web (web)")
		catChoix := readLine(scanner, "Choisissez la catégorie (1-4) : ", true)
		if catChoix == "1" {
			category = "photo"
			break
		} else if catChoix == "2" {
			category = "video"
			break
		} else if catChoix == "3" {
			category = "graphique"
			break
		} else if catChoix == "4" {
			category = "web"
			break
		}
		fmt.Println("Choix incorrect.")
	}

	// Sélection Type Média Principal
	var mediaType string
	for {
		fmt.Println("\nType de média principal :")
		fmt.Println("1. Image")
		fmt.Println("2. Vidéo")
		mChoix := readLine(scanner, "Choisissez (1-2) : ", true)
		if mChoix == "1" {
			mediaType = "image"
			break
		} else if mChoix == "2" {
			mediaType = "video"
			break
		}
		fmt.Println("Choix incorrect.")
	}

	src := readLine(scanner, "Chemin du média principal (ex: /images/mon-image.jpg) : ", true)
	
	bgColor := readLine(scanner, "Couleur de fond en héxadécimal (ex: #D3724D ou #1A1A1A) : ", true)
	if !strings.HasPrefix(bgColor, "#") {
		bgColor = "#" + bgColor
	}

	// --- SAISIE DES DESCRIPTIONS ---
	fmt.Println("\n-- DESCRIPTIONS --")
	descriptionFR := readLine(scanner, "Description (Français) : ", true)
	
	descriptionEN := readLine(scanner, "Description (Anglais) (laisser vide pour copier le Français) : ", false)
	if descriptionEN == "" {
		descriptionEN = descriptionFR
	}

	descriptionMG := readLine(scanner, "Description (Malagasy) (laisser vide pour copier le Français) : ", false)
	if descriptionMG == "" {
		descriptionMG = descriptionFR
	}

	// Galerie d'images
	var gallery models.Gallery
	fmt.Println("\n--- GALERIE MÉDIA DU PROJET ---")
	for {
		addGal := readLine(scanner, "Ajouter un élément à la galerie ? (o/n) : ", true)
		if strings.ToLower(addGal) != "o" {
			break
		}

		var gType string
		for {
			gTypeChoix := readLine(scanner, "  Type de média de galerie (1. Image, 2. Vidéo) : ", true)
			if gTypeChoix == "1" {
				gType = "image"
				break
			} else if gTypeChoix == "2" {
				gType = "video"
				break
			}
			fmt.Println("  Choix incorrect.")
		}

		gSrc := readLine(scanner, "  Chemin de l'élément de galerie (ex: /images/gal1.jpg) : ", true)
		gallery = append(gallery, models.GalleryItem{
			Type: gType,
			Src:  gSrc,
		})
	}

	// Récapitulatif et validation
	fmt.Println("\n--- RÉCAPITULATIF DU PROJET ---")
	fmt.Printf("ID          : %s\n", projectID)
	fmt.Printf("Titre (FR)  : %s %s\n", titleFRPart1, titleFRPart2)
	fmt.Printf("Titre (EN)  : %s %s\n", titleENPart1, titleENPart2)
	fmt.Printf("Titre (MG)  : %s %s\n", titleMGPart1, titleMGPart2)
	fmt.Printf("Catégorie   : %s\n", category)
	fmt.Printf("Média Type  : %s\n", mediaType)
	fmt.Printf("Source      : %s\n", src)
	fmt.Printf("Couleur     : %s\n", bgColor)
	fmt.Printf("Desc (FR)   : %.50s...\n", descriptionFR)
	fmt.Printf("Galerie     : %d éléments\n", len(gallery))

	confirm := readLine(scanner, "\nConfirmez-vous l'ajout de ce projet ? (o/n) : ", true)
	if strings.ToLower(confirm) == "o" {
		p := &models.Project{
			ProjectID:     projectID,
			Category:      category,
			MediaType:     mediaType,
			Src:           src,
			BgColor:       bgColor,
			Gallery:       gallery,
			TitleFRPart1:  titleFRPart1,
			TitleFRPart2:  titleFRPart2,
			DescriptionFR: descriptionFR,
			TitleENPart1:  titleENPart1,
			TitleENPart2:  titleENPart2,
			DescriptionEN: descriptionEN,
			TitleMGPart1:  titleMGPart1,
			TitleMGPart2:  titleMGPart2,
			DescriptionMG: descriptionMG,
		}

		err := db.InsertProject(database, p)
		if err != nil {
			fmt.Printf("-> Erreur lors de l'enregistrement en base : %v\n", err)
		} else {
			fmt.Println("-> Projet enregistré avec succès !")
		}
	} else {
		fmt.Println("-> Ajout annulé.")
	}
}

// listerProjets affiche la liste de tous les projets en base (affiche le titre français pour le résumé CLI)
func listerProjets(database *sql.DB) {
	projects, err := db.GetAllProjects(database)
	if err != nil {
		fmt.Printf("Erreur de lecture de la base de données : %v\n", err)
		return
	}

	fmt.Println("\n--- LISTE DES PROJETS DANS POSTGRESQL ---")
	if len(projects) == 0 {
		fmt.Println("Aucun projet trouvé.")
		return
	}

	fmt.Printf("%-15s | %-12s | %-30s | %-12s\n", "ID UNIQUE", "CATÉGORIE", "TITRE (FR)", "GALERIE")
	fmt.Println(strings.Repeat("-", 80))
	for _, p := range projects {
		titleStr := p.TitleFRPart1
		if p.TitleFRPart2 != "" {
			titleStr += " " + p.TitleFRPart2
		}
		if len(titleStr) > 28 {
			titleStr = titleStr[:25] + "..."
		}
		
		fmt.Printf("%-15s | %-12s | %-30s | %d images\n", 
			p.ProjectID, p.Category, titleStr, len(p.Gallery))
	}
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Total : %d projets enregistrés.\n", len(projects))
}

// supprimerProjetInteractif supprime un projet spécifié par son ID
func supprimerProjetInteractif(scanner *bufio.Scanner, database *sql.DB) {
	fmt.Println("\n--- SUPPRESSION D'UN PROJET ---")
	projectID := readLine(scanner, "Saisissez l'ID unique du projet à supprimer : ", true)

	// Rechercher d'abord si le projet existe
	projects, err := db.GetAllProjects(database)
	if err != nil {
		fmt.Printf("Erreur de base de données: %v\n", err)
		return
	}

	found := false
	var target models.Project
	for _, p := range projects {
		if p.ProjectID == projectID {
			target = p
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("-> Erreur : Aucun projet n'a été trouvé avec l'identifiant '%s'.\n", projectID)
		return
	}

	titleStr := target.TitleFRPart1
	if target.TitleFRPart2 != "" {
		titleStr += " " + target.TitleFRPart2
	}

	fmt.Printf("Projet trouvé : %s (%s - %s)\n", titleStr, target.ProjectID, target.Category)
	confirm := readLine(scanner, "Êtes-vous SÛR de vouloir supprimer définitivement ce projet ? (o/n) : ", true)
	
	if strings.ToLower(confirm) == "o" {
		err := db.DeleteProject(database, projectID)
		if err != nil {
			fmt.Printf("-> Erreur lors de la suppression : %v\n", err)
		} else {
			fmt.Println("-> Projet supprimé avec succès.")
		}
	} else {
		fmt.Println("-> Suppression annulée.")
	}
}

// AsInt convertit une chaîne en entier, avec repli par défaut
func AsInt(s string, def int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return val
}
