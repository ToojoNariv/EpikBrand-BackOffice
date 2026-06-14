package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"epik-brand-backend/cli"
	"epik-brand-backend/config"
	"epik-brand-backend/db"
	"epik-brand-backend/server"

	"github.com/jchv/go-webview2"
)

func main() {
	log.Println("[INIT] Initialisation du backend Epik Brand...")

	// 1. Chargement de la configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		showWindowsMessageBox("Erreur de configuration", fmt.Sprintf("Impossible de charger la configuration : %v", err))
		log.Fatalf("[ERREUR] Impossible de charger la configuration: %v", err)
	}

	// 2. Vérifier les arguments d'aide avant la connexion DB
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	// NOUVEAU: Si REMOTE_URL est configuré dans le .env, on lance l'interface directement vers le serveur cloud
	if len(args) == 0 && cfg.RemoteURL != "" {
		log.Printf("[INIT] Mode client distant activé. Connexion au serveur cloud : %s\n", cfg.RemoteURL)
		runInterface(cfg.RemoteURL, false)
		return
	}

	// 3. Connexion et initialisation de la base PostgreSQL
	database, err := db.ConnectAndInitDB(cfg)
	if err != nil {
		showWindowsMessageBox("Erreur de connexion - PostgreSQL",
			fmt.Sprintf("Impossible de se connecter à la base de données : %v\n\n"+
				"VEUILLEZ VÉRIFIER QUE :\n"+
				"1. PostgreSQL est bien démarré sur votre machine.\n"+
				"2. Les identifiants dans le fichier .env (DB_PASSWORD, DB_NAME, etc.) sont corrects.\n"+
				"3. Le port 5432 est accessible.", err))
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
		// ─────────────────────────────────────────────────────────────────────
		// MODE PAR DÉFAUT (double-clic) : Fenêtre logicielle native WebView2
		// ─────────────────────────────────────────────────────────────────────
		log.Println("[INIT] Lancement de l'interface graphique d'administration locale...")

		// Démarrer le serveur API en arrière-plan
		go func() {
			err := server.StartServer(cfg, database)
			if err != nil {
				showWindowsMessageBox("Erreur Serveur", fmt.Sprintf("Erreur lors du démarrage du serveur web local : %v", err))
				os.Exit(1)
			}
		}()

		// Attendre brièvement que le serveur démarre
		time.Sleep(600 * time.Millisecond)

		url := "http://localhost:" + cfg.ServerPort
		runInterface(url, true)
	}
}

// runInterface lance l'interface graphique (WebView2 ou navigateur de repli)
func runInterface(url string, waitForever bool) {
	// ── Tentative 1 : Fenêtre native WebView2 (vraie application de bureau) ──
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Epik Brand — Portfolio Manager",
			Width:  1100,
			Height: 750,
			IconId: 2, // Utilise l'icône embarquée dans les ressources Windows (.syso)
			Center: true,
		},
	})

	if w != nil {
		defer w.Destroy()
		w.SetSize(1100, 750, webview2.HintNone)
		w.Navigate(url)
		log.Println("[INIT] Fenêtre native WebView2 ouverte avec succès.")

		// Rendre la fenêtre maximized (plein écran) et personnaliser le header natif
		hwnd := uintptr(w.Window())
		if hwnd != 0 {
			user32 := syscall.NewLazyDLL("user32.dll")
			showWindow := user32.NewProc("ShowWindow")
			// SW_MAXIMIZE = 3 (maximise la fenêtre)
			showWindow.Call(hwnd, 3)

			// Personnaliser le header de la fenêtre native Windows (rendre blanc et propre)
			dwmapi := syscall.NewLazyDLL("dwmapi.dll")
			dwmSetWindowAttribute := dwmapi.NewProc("DwmSetWindowAttribute")
			if dwmSetWindowAttribute.Find() == nil {
				// DWMWA_CAPTION_COLOR = 35
				var captionColor uint32 = 0x00FFFFFF // Blanc (0x00BBGGRR)
				_, _, _ = dwmSetWindowAttribute.Call(
					hwnd,
					35,
					uintptr(unsafe.Pointer(&captionColor)),
					unsafe.Sizeof(captionColor),
				)

				// DWMWA_TEXT_COLOR = 36
				var textColor uint32 = 0x00000000 // Noir (0x00BBGGRR)
				_, _, _ = dwmSetWindowAttribute.Call(
					hwnd,
					36,
					uintptr(unsafe.Pointer(&textColor)),
					unsafe.Sizeof(textColor),
				)

				// DWMWA_BORDER_COLOR = 34
				var borderColor uint32 = 0x00FFFFFF // Blanc (0x00BBGGRR)
				_, _, _ = dwmSetWindowAttribute.Call(
					hwnd,
					34,
					uintptr(unsafe.Pointer(&borderColor)),
					unsafe.Sizeof(borderColor),
				)
			}
		}

		w.Run() // Bloque jusqu'à la fermeture de la fenêtre
		os.Exit(0)
	}

	// ── Tentative 2 (repli) : Navigateur par défaut ──
	log.Println("[INIT] WebView2 non disponible. Ouverture dans le navigateur par défaut...")
	showWindowsMessageBox("Information",
		"Le composant WebView2 n'est pas installé sur cette machine.\n\n"+
			"L'interface va s'ouvrir dans votre navigateur par défaut.\n"+
			"Pour une expérience optimale, installez le WebView2 Runtime depuis :\n"+
			"https://developer.microsoft.com/microsoft-edge/webview2/")
	openBrowser(url)

	if waitForever {
		// En mode repli navigateur, on laisse le serveur tourner indéfiniment
		select {}
	}
}

// showWindowsMessageBox affiche un dialogue pop-up natif Windows sans dépendance CGO ou PowerShell (via user32.dll)
func showWindowsMessageBox(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)

	// MB_OK | MB_ICONINFORMATION | MB_SYSTEMMODAL
	var style uint32 = 0x00000000 | 0x00000040 | 0x00001000
	if title == "Erreur de configuration" || title == "Erreur de connexion - PostgreSQL" || title == "Erreur Serveur" {
		// MB_OK | MB_ICONERROR | MB_SYSTEMMODAL
		style = 0x00000000 | 0x00000010 | 0x00001000
	}

	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")

	_, _, _ = messageBoxW.Call(0, uintptr(unsafe.Pointer(messagePtr)), uintptr(unsafe.Pointer(titlePtr)), uintptr(style))
}

// openBrowser ouvre l'URL passée en paramètre dans le navigateur par défaut de Windows
func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func printHelp() {
	fmt.Println("\nUsage:")
	fmt.Println("  EpikBrand.exe          : Démarre le serveur et ouvre l'interface graphique logicielle (par défaut)")
	fmt.Println("  EpikBrand.exe server   : Démarre uniquement le serveur REST API (sans ouvrir d'interface)")
	fmt.Println("  EpikBrand.exe terminal : Démarre le mode d'administration console historique dans le terminal")
	fmt.Println("  EpikBrand.exe help     : Affiche cette aide")
	fmt.Println()
}


