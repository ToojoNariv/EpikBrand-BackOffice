package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"epik-brand-backend/cli"
	"epik-brand-backend/config"
	"epik-brand-backend/db"
	"epik-brand-backend/server"
)

func main() {
	log.Println("[INIT] Initialisation du backend Epik Brand...")

	// 1. Chargement de la configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		showErrorMessage("Erreur de configuration", fmt.Sprintf("Impossible de charger la configuration : %v", err))
		log.Fatalf("[ERREUR] Impossible de charger la configuration: %v", err)
	}

	// 2. Vérifier les arguments d'aide avant la connexion DB
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	// Si REMOTE_URL est configuré dans le .env, on lance l'interface directement vers le serveur cloud
	if len(args) == 0 && cfg.RemoteURL != "" {
		log.Printf("[INIT] Mode client distant activé. Connexion au serveur cloud : %s\n", cfg.RemoteURL)
		runInterface(cfg.RemoteURL, false)
		return
	}

	// 3. Connexion et initialisation de la base PostgreSQL
	database, err := db.ConnectAndInitDB(cfg)
	if err != nil {
		showErrorMessage("Erreur de connexion - PostgreSQL",
			fmt.Sprintf("Impossible de se connecter à la base de données : %v\n\n"+
				"VEUILLEZ VÉRIFIER QUE :\n"+
				"1. PostgreSQL/Neon est bien accessible.\n"+
				"2. Les identifiants dans le fichier .env sont corrects.", err))
		os.Exit(1)
	}
	defer database.Close()

	// 4. Routage selon les arguments de ligne de commande
	if len(args) > 0 {
		cmd := args[0]
		switch cmd {
		case "server":
			log.Println("[INIT] Lancement en mode Serveur API uniquement.")
			err := server.StartServer(cfg, database)
			if err != nil {
				log.Fatalf("[ERREUR] Erreur du serveur: %v", err)
			}
		case "terminal", "cli":
			cli.RunAdminCLI(cfg, database)
		default:
			fmt.Printf("Commande inconnue: '%s'\n", cmd)
		}
	} else {
		// MODE PAR DÉFAUT (double-clic) : Fenêtre logicielle native WebView2
		log.Println("[INIT] Lancement de l'interface graphique d'administration locale...")

		// Démarrer le serveur API en arrière-plan
		go func() {
			err := server.StartServer(cfg, database)
			if err != nil {
				showErrorMessage("Erreur Serveur", fmt.Sprintf("Erreur lors du démarrage du serveur web local : %v", err))
				os.Exit(1)
			}
		}()

		// Attendre brièvement que le serveur démarre
		time.Sleep(600 * time.Millisecond)

		url := "http://localhost:" + cfg.ServerPort
		runInterface(url, true)
	}
}

func printHelp() {
	fmt.Println("\nUsage:")
	fmt.Println("  EpikBrand.exe          : Démarre le serveur et ouvre l'interface graphique logicielle (par défaut)")
	fmt.Println("  EpikBrand.exe server   : Démarre uniquement le serveur REST API (sans ouvrir d'interface)")
	fmt.Println("  EpikBrand.exe terminal : Démarre le mode d'administration console historique dans le terminal")
	fmt.Println("  EpikBrand.exe help     : Affiche cette aide")
	fmt.Println()
}
