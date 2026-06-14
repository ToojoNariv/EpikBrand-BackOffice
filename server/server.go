package server

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
	"crypto/rand"
	"encoding/hex"

	"epik-brand-backend/config"
	"epik-brand-backend/db"
	"epik-brand-backend/models"
)

//go:embed admin_ui/*
var adminUI embed.FS

// StartServer démarre le serveur HTTP REST API et sert l'interface graphique
func StartServer(cfg *config.Config, database *sql.DB) error {
	mux := http.NewServeMux()
	
	// API publiques/privées Projets et Équipe
	mux.HandleFunc("/api/projects", enableCORS(handleProjects(database)))
	mux.HandleFunc("/api/team", enableCORS(handleTeam(database)))

	// API de configuration (Public)
	mux.HandleFunc("/api/config", enableCORS(handleConfig(cfg)))

	// API d'authentification (Public/Privée)
	mux.HandleFunc("/api/auth/google", enableCORS(handleGoogleAuth(cfg, database)))
	mux.HandleFunc("/api/auth/logout", enableCORS(handleLogout(database)))
	mux.HandleFunc("/api/auth/me", enableCORS(handleMe(database)))

	// API de gestion d'utilisateurs (Admin uniquement, protégé par token)
	mux.HandleFunc("/api/users", enableCORS(handleUsers(database)))
	mux.HandleFunc("/api/users/transfer-admin", enableCORS(handleTransferAdmin(database)))

	// Servir les fichiers de l'interface d'administration
	sub, err := fs.Sub(adminUI, "admin_ui")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	log.Printf("[SERVEUR] API et Interface Graphique démarrées sur http://localhost:%s", cfg.ServerPort)
	log.Println("[SERVEUR] En attente de requêtes (Ctrl+C pour arrêter)...")
	
	return http.ListenAndServe(":"+cfg.ServerPort, mux)
}

// enableCORS est un middleware pour autoriser le Cross-Origin Resource Sharing (nécessaire pour le front-end)
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Répondre instantanément aux requêtes de pré-vérification (CORS preflight)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// authenticateRequest valide l'en-tête Authorization et renvoie l'utilisateur associé
func authenticateRequest(database *sql.DB, r *http.Request) (*models.User, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("authentification requise")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	user, err := db.GetSessionUser(database, token)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la validation de session")
	}
	if user == nil {
		return nil, fmt.Errorf("session invalide ou expirée")
	}
	return user, nil
}

// generateSessionToken génère un jeton aléatoire sécurisé
func generateSessionToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GoogleTokenInfo représente le payload renvoyé par l'API tokeninfo de Google
type GoogleTokenInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	EmailVerified string `json:"email_verified"`
	Error         string `json:"error"`
}

// verifyGoogleToken valide le jeton Google auprès de l'API de Google
func verifyGoogleToken(idToken string) (*GoogleTokenInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("connexion à google tokeninfo impossible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errInfo struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errInfo)
		return nil, fmt.Errorf("erreur google api (%d): %s %s", resp.StatusCode, errInfo.Error, errInfo.ErrorDescription)
	}

	var info GoogleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("erreur de décodage google: %w", err)
	}

	if info.Error != "" {
		return nil, fmt.Errorf("erreur token google: %s", info.Error)
	}

	if info.EmailVerified != "true" {
		return nil, fmt.Errorf("adresse email google non vérifiée")
	}

	return &info, nil
}

// handleConfig renvoie les configurations nécessaires au frontend
func handleConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"google_client_id": cfg.GoogleClientID,
		})
	}
}

// handleMe renvoie les informations de l'utilisateur actuellement connecté
func handleMe(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		user, err := authenticateRequest(database, r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(user)
	}
}

// handleGoogleAuth valide l'auth Google et génère une session
func handleGoogleAuth(cfg *config.Config, database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}

		var payload struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Token == "" {
			http.Error(w, `{"error": "Jeton Google manquant ou invalide"}`, http.StatusBadRequest)
			return
		}

		info, err := verifyGoogleToken(payload.Token)
		if err != nil {
			log.Printf("[AUTH] Échec vérification Google : %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		user, err := db.GetUserByEmail(database, info.Email)
		if err != nil {
			log.Printf("[AUTH] Erreur recherche utilisateur: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Erreur interne"})
			return
		}

		if user == nil {
			log.Printf("[AUTH] Accès refusé : %s (non autorisé)", info.Email)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Accès refusé. Vous n'êtes pas enregistré."})
			return
		}

		// Mettre à jour le profil de l'utilisateur avec son vrai nom Google et sa photo de profil
		_ = db.UpdateUserProfile(database, info.Email, info.Name, info.Picture)
		user.Name = info.Name
		user.PictureURL = info.Picture

		// Créer la session pour 7 jours
		sessionToken := generateSessionToken()
		err = db.CreateSession(database, sessionToken, user.ID, 7*24*time.Hour)
		if err != nil {
			log.Printf("[AUTH] Échec de création de la session: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Erreur de création de session"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"token":       sessionToken,
			"email":       user.Email,
			"name":        user.Name,
			"role":        user.Role,
			"picture_url": user.PictureURL,
		})
	}
}

// handleLogout invalide le jeton de session
func handleLogout(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			_ = db.DeleteSession(database, token)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

// handleUsers gère la liste et l'inscription des modérateurs par l'administrateur
func handleUsers(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, err := authenticateRequest(database, r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Seul l'administrateur peut accéder et modifier la liste des modérateurs
		if currentUser.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Accès réservé à l'administrateur"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			users, err := db.GetAllUsers(database)
			if err != nil {
				log.Printf("[SERVEUR] Erreur récupération modérateurs: %v", err)
				http.Error(w, "Erreur serveur", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(users)

		case http.MethodPost:
			var req struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Données d'utilisateur invalides"})
				return
			}

			// Force le rôle moderator
			err := db.InsertUser(database, req.Email, req.Name, "moderator")
			if err != nil {
				log.Printf("[SERVEUR] Erreur insertion modérateur: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Erreur lors de l'enregistrement"})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Modérateur enregistré avec succès"})

		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				http.Error(w, "ID manquant", http.StatusBadRequest)
				return
			}
			var id int
			_, err := fmt.Sscanf(idStr, "%d", &id)
			if err != nil {
				http.Error(w, "ID invalide", http.StatusBadRequest)
				return
			}

			err = db.DeleteUser(database, id)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Modérateur supprimé avec succès"})

		default:
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

// handleTransferAdmin permet à l'admin de transférer son rôle
func handleTransferAdmin(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}

		currentUser, err := authenticateRequest(database, r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if currentUser.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Accès réservé à l'administrateur"})
			return
		}

		var req struct {
			UserID int `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "ID d'utilisateur cible manquant ou invalide"})
			return
		}

		if req.UserID == currentUser.ID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Vous possédez déjà le rôle d'administrateur"})
			return
		}

		err = db.TransferAdminRole(database, currentUser.ID, req.UserID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Le rôle d'administrateur a été transféré. Vous êtes maintenant modérateur.",
		})
	}
}

// handleProjects route les requêtes HTTP de projets en protégeant les écritures
func handleProjects(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			projects, err := db.GetAllProjects(database)
			if err != nil {
				log.Printf("[SERVEUR] Erreur de récupération des projets: %v", err)
				http.Error(w, "Erreur interne: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Récupère la langue demandée (ex: ?lang=en ou ?lang=mg)
			lang := r.URL.Query().Get("lang")
			for i := range projects {
				projects[i].Localize(lang)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(projects)

		case http.MethodPost:
			_, err := authenticateRequest(database, r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			var p models.Project
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				log.Printf("[SERVEUR] JSON invalide dans POST: %v", err)
				http.Error(w, "Requête JSON invalide: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Recréer le titre découpé pour la base s'il vient du tableau simple JSON 'titre' (repli FR)
			if p.TitleFRPart1 == "" && len(p.Title) > 0 {
				p.TitleFRPart1 = p.Title[0]
			}
			if p.TitleFRPart2 == "" && len(p.Title) > 1 {
				p.TitleFRPart2 = p.Title[1]
			}
			// Pareil pour la description (repli FR)
			if p.DescriptionFR == "" && p.Description != "" {
				p.DescriptionFR = p.Description
			}

			if err := db.InsertProject(database, &p); err != nil {
				log.Printf("[SERVEUR] Erreur lors de l'insertion en base: %v", err)
				http.Error(w, "Erreur de base de données: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("[SERVEUR] Projet '%s' sauvegardé/mis à jour avec succès.", p.ProjectID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Projet enregistré avec succès",
				"id":      p.ProjectID,
			})

		case http.MethodDelete:
			_, err := authenticateRequest(database, r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			projectID := r.URL.Query().Get("id")
			if projectID == "" {
				http.Error(w, "Le paramètre de requête '?id=' est obligatoire pour la suppression", http.StatusBadRequest)
				return
			}

			if err := db.DeleteProject(database, projectID); err != nil {
				log.Printf("[SERVEUR] Erreur de suppression du projet '%s': %v", projectID, err)
				http.Error(w, "Erreur de base de données: "+err.Error(), http.StatusNotFound)
				return
			}

			log.Printf("[SERVEUR] Projet '%s' supprimé avec succès.", projectID)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Projet supprimé avec succès",
				"id":      projectID,
			})

		default:
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}

// handleTeam route les requêtes HTTP de l'équipe en protégeant les écritures
func handleTeam(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			members, err := db.GetAllTeamMembers(database)
			if err != nil {
				log.Printf("[SERVEUR] Erreur de récupération des membres de l'équipe: %v", err)
				http.Error(w, "Erreur interne: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Récupère la langue demandée (ex: ?lang=en ou ?lang=mg)
			lang := r.URL.Query().Get("lang")
			for i := range members {
				members[i].Localize(lang)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(members)

		case http.MethodPost:
			_, err := authenticateRequest(database, r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			var m models.TeamMember
			if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
				log.Printf("[SERVEUR] JSON invalide dans POST team: %v", err)
				http.Error(w, "Requête JSON invalide: "+err.Error(), http.StatusBadRequest)
				return
			}

			if err := db.InsertTeamMember(database, &m); err != nil {
				log.Printf("[SERVEUR] Erreur lors de l'insertion en base du membre: %v", err)
				http.Error(w, "Erreur de base de données: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("[SERVEUR] Membre '%s %s' sauvegardé/mis à jour avec succès.", m.Prenom, m.Nom)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Membre enregistré avec succès",
				"id":      m.MemberID,
			})

		case http.MethodDelete:
			_, err := authenticateRequest(database, r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			memberID := r.URL.Query().Get("id")
			if memberID == "" {
				http.Error(w, "Le paramètre de requête '?id=' est obligatoire pour la suppression", http.StatusBadRequest)
				return
			}

			if err := db.DeleteTeamMember(database, memberID); err != nil {
				log.Printf("[SERVEUR] Erreur de suppression du membre '%s': %v", memberID, err)
				http.Error(w, "Erreur de base de données: "+err.Error(), http.StatusNotFound)
				return
			}

			log.Printf("[SERVEUR] Membre '%s' supprimé avec succès.", memberID)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Membre supprimé avec succès",
				"id":      memberID,
			})

		default:
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		}
	}
}
