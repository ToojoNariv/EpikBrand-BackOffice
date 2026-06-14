//go:build !windows
package main

import (
	"log"
	"os/exec"
)

// runInterface sur Linux/MacOS affiche un avertissement ou ouvre simplement le navigateur par défaut
func runInterface(url string, waitForever bool) {
	log.Printf("[INIT] Mode graphique de bureau non supporté sur ce système d'exploitation. Ouverture du navigateur à l'adresse : %s\n", url)
	openBrowser(url)
	if waitForever {
		select {}
	}
}

// showErrorMessage affiche les erreurs dans la console sur les systèmes non-Windows
func showErrorMessage(title, message string) {
	log.Printf("[ERREUR] %s : %s\n", title, message)
}

// openBrowser tente d'ouvrir le navigateur sur Linux ou MacOS
func openBrowser(url string) {
	// Essaye d'ouvrir sur Linux (xdg-open) puis sur MacOS (open)
	_ = exec.Command("xdg-open", url).Start()
}
