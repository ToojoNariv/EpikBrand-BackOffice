package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"epik-brand-backend/config"
	"epik-brand-backend/models"

	_ "github.com/lib/pq"
)

// ConnectAndInitDB se connecte à Postgres, crée la BDD si nécessaire, et initialise la table
func ConnectAndInitDB(cfg *config.Config) (*sql.DB, error) {
	// 1. Tenter de se connecter à la base par défaut "postgres" pour vérifier/créer la base cible
	dsnDefault := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	
	dbDefault, err := sql.Open("postgres", dsnDefault)
	if err == nil {
		defer dbDefault.Close()
		
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname='%s')", cfg.DBName)
		err = dbDefault.QueryRow(query).Scan(&exists)
		if err == nil && !exists {
			log.Printf("[DB] La base de données '%s' n'existe pas. Création en cours...", cfg.DBName)
			_, err = dbDefault.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
			if err != nil {
				log.Printf("[DB] Avertissement: Impossible de créer la base '%s': %v (elle existe peut-être déjà ou les permissions sont insuffisantes)", cfg.DBName, err)
			} else {
				log.Printf("[DB] Base de données '%s' créée avec succès.", cfg.DBName)
			}
		}
	}

	// 2. Se connecter à la base de données de l'application
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'ouverture de la connexion DB: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("impossible de pinger la base de données: %w", err)
	}

	// 3. Initialiser les tables
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("erreur de création du schéma: %w", err)
	}

	// 3.5 Initialiser l'administrateur par défaut
	if err := seedInitialAdmin(db, cfg.InitialAdminEmail); err != nil {
		log.Printf("[DB] Avertissement lors de l'initialisation de l'administrateur : %v", err)
	}

	// 4. Initialiser les membres de l'équipe par défaut (désactivé)
	/*
	if err := seedDefaultTeamData(db); err != nil {
		log.Printf("[DB] Avertissement lors de l'initialisation de l'équipe: %v", err)
	}
	*/

	return db, nil
}

// seedInitialAdmin crée l'administrateur par défaut si aucun utilisateur n'existe
func seedInitialAdmin(db *sql.DB, adminEmail string) error {
	if adminEmail == "" {
		log.Println("[DB] Aucun email d'administrateur initial défini (INITIAL_ADMIN_EMAIL vide).")
		return nil
	}
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		log.Printf("[DB] Création de l'administrateur par défaut : %s", adminEmail)
		_, err = db.Exec("INSERT INTO users (email, name, role) VALUES ($1, $2, 'admin')", adminEmail, "Admin Initial")
		if err != nil {
			return fmt.Errorf("impossible de créer l'administrateur initial : %w", err)
		}
	}
	return nil
}

// createSchema crée la table des projets avec support multilingue (Français, Anglais, Malgache)
func createSchema(db *sql.DB) error {
	var hasMultiLang bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='projects' AND column_name='title_en_part1')").Scan(&hasMultiLang)
	if err == nil && !hasMultiLang {
		log.Println("[DB] Migration: Ancienne table projects détectée sans support multilingue. Recréation de la table...")
		_, _ = db.Exec("DROP TABLE IF EXISTS projects;")
	}

	query := `
	CREATE TABLE IF NOT EXISTS projects (
		id SERIAL PRIMARY KEY,
		project_id VARCHAR(50) UNIQUE NOT NULL,
		category VARCHAR(50) NOT NULL,
		media_type VARCHAR(20) NOT NULL,
		src TEXT NOT NULL,
		bg_color VARCHAR(20) NOT NULL,
		gallery JSONB NOT NULL,
		
		-- Français (FR)
		title_fr_part1 VARCHAR(100) NOT NULL,
		title_fr_part2 VARCHAR(100),
		description_fr TEXT NOT NULL,
		
		-- Anglais (EN)
		title_en_part1 VARCHAR(100),
		title_en_part2 VARCHAR(100),
		description_en TEXT,
		
		-- Malgache (MG)
		title_mg_part1 VARCHAR(100),
		title_mg_part2 VARCHAR(100),
		description_mg TEXT,
		
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		return err
	}

	// Assure la migration de la colonne 'src' vers le type TEXT si la table existait déjà en VARCHAR(255)
	_, err = db.Exec("ALTER TABLE projects ALTER COLUMN src TYPE TEXT;")
	if err != nil {
		log.Printf("[DB] Note: Échec ou saut de la migration de la colonne 'src' vers TEXT (déjà fait ou type compatible): %v", err)
	}

	// Création de la table team_members
	queryTeam := `
	CREATE TABLE IF NOT EXISTS team_members (
		id SERIAL PRIMARY KEY,
		member_id VARCHAR(50) UNIQUE NOT NULL,
		prenom VARCHAR(100) NOT NULL,
		nom VARCHAR(100) NOT NULL,
		email VARCHAR(100) NOT NULL,
		photo TEXT NOT NULL,
		role_fr VARCHAR(100) NOT NULL,
		role_en VARCHAR(100),
		role_mg VARCHAR(100),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(queryTeam)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la table team_members: %w", err)
	}

	// Création de la table users
	queryUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(100) UNIQUE NOT NULL,
		name VARCHAR(100) NOT NULL,
		role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'moderator')),
		picture_url TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(queryUsers)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la table users: %w", err)
	}

	// S'assurer de la présence de la colonne picture_url si la table existait déjà
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS picture_url TEXT DEFAULT '';")

	// Création de la table sessions
	querySessions := `
	CREATE TABLE IF NOT EXISTS sessions (
		token VARCHAR(64) PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TIMESTAMP NOT NULL
	);
	`
	_, err = db.Exec(querySessions)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la table sessions: %w", err)
	}

	// Création de la table settings
	querySettings := `
	CREATE TABLE IF NOT EXISTS settings (
		key VARCHAR(50) PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err = db.Exec(querySettings)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la table settings: %w", err)
	}

	// Insérer la configuration par défaut de Looker Studio
	_, err = db.Exec("INSERT INTO settings (key, value) VALUES ('looker_studio_url', '') ON CONFLICT (key) DO NOTHING;")
	if err != nil {
		return fmt.Errorf("erreur lors de l'initialisation du paramètre looker_studio_url: %w", err)
	}

	log.Println("[DB] Schéma vérifié/créé avec succès.")
	return nil
}

// InsertProject insère ou met à jour un projet
func InsertProject(db *sql.DB, p *models.Project) error {
	// Si aucun project_id n'est spécifié, on insère d'abord puis on génère l'id auto-incrémenté
	if p.ProjectID == "" {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("erreur lors du démarrage de la transaction: %w", err)
		}
		defer tx.Rollback()

		var id int64
		// On insère avec un project_id temporaire unique pour satisfaire la contrainte UNIQUE
		tempID := fmt.Sprintf("temp-%d", time.Now().UnixNano())
		query := `
		INSERT INTO projects (
			project_id, category, media_type, src, bg_color, gallery,
			title_fr_part1, title_fr_part2, description_fr,
			title_en_part1, title_en_part2, description_en,
			title_mg_part1, title_mg_part2, description_mg
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id;
		`
		err = tx.QueryRow(
			query,
			tempID, p.Category, p.MediaType, p.Src, p.BgColor, p.Gallery,
			p.TitleFRPart1, p.TitleFRPart2, p.DescriptionFR,
			p.TitleENPart1, p.TitleENPart2, p.DescriptionEN,
			p.TitleMGPart1, p.TitleMGPart2, p.DescriptionMG,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("erreur lors de l'insertion initiale du projet: %w", err)
		}

		// On définit l'identifiant propre auto-incrémenté (ex: photo-15)
		p.ProjectID = fmt.Sprintf("%s-%d", p.Category, id)
		
		updateQuery := `UPDATE projects SET project_id = $1 WHERE id = $2;`
		_, err = tx.Exec(updateQuery, p.ProjectID, id)
		if err != nil {
			return fmt.Errorf("erreur lors de la mise à jour de l'identifiant généré: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("erreur lors de la validation (commit) de la transaction: %w", err)
		}
		return nil
	}

	// Si un project_id est spécifié (upsert classique ou import de démo)
	query := `
	INSERT INTO projects (
		project_id, category, media_type, src, bg_color, gallery,
		title_fr_part1, title_fr_part2, description_fr,
		title_en_part1, title_en_part2, description_en,
		title_mg_part1, title_mg_part2, description_mg
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	ON CONFLICT (project_id) 
	DO UPDATE SET 
		category = EXCLUDED.category,
		media_type = EXCLUDED.media_type,
		src = EXCLUDED.src,
		bg_color = EXCLUDED.bg_color,
		gallery = EXCLUDED.gallery,
		
		title_fr_part1 = EXCLUDED.title_fr_part1,
		title_fr_part2 = EXCLUDED.title_fr_part2,
		description_fr = EXCLUDED.description_fr,
		
		title_en_part1 = EXCLUDED.title_en_part1,
		title_en_part2 = EXCLUDED.title_en_part2,
		description_en = EXCLUDED.description_en,
		
		title_mg_part1 = EXCLUDED.title_mg_part1,
		title_mg_part2 = EXCLUDED.title_mg_part2,
		description_mg = EXCLUDED.description_mg;
	`
	_, err := db.Exec(
		query, 
		p.ProjectID, p.Category, p.MediaType, p.Src, p.BgColor, p.Gallery,
		p.TitleFRPart1, p.TitleFRPart2, p.DescriptionFR,
		p.TitleENPart1, p.TitleENPart2, p.DescriptionEN,
		p.TitleMGPart1, p.TitleMGPart2, p.DescriptionMG,
	)
	return err
}

// GetAllProjects récupère tous les projets avec tous leurs champs de traduction
func GetAllProjects(db *sql.DB) ([]models.Project, error) {
	query := `
	SELECT id, project_id, category, media_type, src, bg_color, gallery,
	       title_fr_part1, title_fr_part2, description_fr,
	       title_en_part1, title_en_part2, description_en,
	       title_mg_part1, title_mg_part2, description_mg
	FROM projects 
	ORDER BY id ASC;
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		err := rows.Scan(
			&p.ID, &p.ProjectID, &p.Category, &p.MediaType, &p.Src, &p.BgColor, &p.Gallery,
			&p.TitleFRPart1, &p.TitleFRPart2, &p.DescriptionFR,
			&p.TitleENPart1, &p.TitleENPart2, &p.DescriptionEN,
			&p.TitleMGPart1, &p.TitleMGPart2, &p.DescriptionMG,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// DeleteProject supprime un projet par son project_id client
func DeleteProject(db *sql.DB, projectID string) error {
	query := `DELETE FROM projects WHERE project_id = $1;`
	res, err := db.Exec(query, projectID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("aucun projet trouvé avec l'id '%s'", projectID)
	}
	return nil
}

// seedDefaultData remplit la base avec les projets d'origine traduits si celle-ci est vide
func seedDefaultData(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("[DB] La table projects contient déjà des données. Saut du peuplement initial.")
		return nil
	}

	log.Println("[DB] Initialisation des données multilingues de démonstration (FR, EN, MG) dans PostgreSQL...")

	defaultProjects := []models.Project{
		// PHOTO
		{
			ProjectID:  "photo-1",
			Category:   "photo",
			MediaType:  "image",
			Src:        "/images/Photographie Background.jpg",
			BgColor:    "#D3724D",
			// FR
			TitleFRPart1:  "Atelier chez",
			TitleFRPart2:  "Zina",
			DescriptionFR: "Une immersion artistique au sein de l'atelier de Zina. Ce projet capture la synergie des textures, des contrastes de lumière brute et de la direction artistique soignée pour forger une image de marque authentique et mémorable.",
			// EN
			TitleENPart1:  "Workshop at",
			TitleENPart2:  "Zina's",
			DescriptionEN: "An artistic immersion within Zina's workshop. This project captures the synergy of textures, raw light contrasts, and neat artistic direction to forge an authentic and memorable brand image.",
			// MG
			TitleMGPart1:  "Atrikasa any",
			TitleMGPart2:  "Zina",
			DescriptionMG: "Fandrobohana ara-kanto ao anatin'ny atrikasan'i Zina. Ity tetikasa ity dia maneho ny fiarahan'ny textures, ny fifanoherana amin'ny hazavana ary ny fitarihana ara-kanto voalamina tsara mba hamoronana marika tena izy.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/Photographie Background.jpg"},
				{Type: "image", Src: "/images/A propos.jpg"},
				{Type: "image", Src: "/images/Portfolio.jpg"},
				{Type: "image", Src: "/images/sam-mcghee-KieCLNzKoBo-unsplash.jpg"},
			},
		},
		{
			ProjectID:  "photo-2",
			Category:   "photo",
			MediaType:  "image",
			Src:        "/images/sam-mcghee-KieCLNzKoBo-unsplash.jpg",
			BgColor:    "#1D3B23",
			// FR
			TitleFRPart1:  "Chambre",
			TitleFRPart2:  "Verte",
			DescriptionFR: "Une exploration intime du design d'intérieur et de la nature morte. À travers une palette émeraude profonde et une gestion délicate de la lumière du jour, ce projet insuffle calme et esthétisme haut de gamme.",
			// EN
			TitleENPart1:  "Green",
			TitleENPart2:  "Room",
			DescriptionEN: "An intimate exploration of interior design and still life. Through a deep emerald palette and delicate daylight management, this project breathes calm and high-end aesthetics.",
			// MG
			TitleMGPart1:  "Efitra",
			TitleMGPart2:  "Maitso",
			DescriptionMG: "Fikarohana lalina momba ny haingon-trano sy ny sary tsotra. Amin'ny alalan'ny loko maitso lalina sy ny fitantanana malefaka ny hazavan'ny andro, ity tetikasa ity dia mitondra fahatoniana sy hatsarana avo lenta.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/sam-mcghee-KieCLNzKoBo-unsplash.jpg"},
				{Type: "image", Src: "/images/Photographie Background.jpg"},
				{Type: "image", Src: "/images/A propos.jpg"},
			},
		},
		{
			ProjectID:  "photo-3",
			Category:   "photo",
			MediaType:  "image",
			Src:        "/images/jeff-sheldon-9SyOKYrq-rE-unsplash.jpg",
			BgColor:    "#C59E7A",
			// FR
			TitleFRPart1:  "Studio",
			TitleFRPart2:  "Lumineux",
			DescriptionFR: "Shooting produit haut de gamme réalisé dans un studio baigné de soleil. Des compositions minimalistes et géométriques valorisent l'essentiel et affirment le positionnement premium de la marque.",
			// EN
			TitleENPart1:  "Bright",
			TitleENPart2:  "Studio",
			DescriptionEN: "High-end product shooting inside a sun-bathed studio. Minimalist and geometric compositions highlight the essentials and assert the brand's premium positioning.",
			// MG
			TitleMGPart1:  "Studio",
			TitleMGPart2:  "Mazava",
			DescriptionMG: "Fakana sary vokatra avo lenta natao tao anatin'ny studio feno hazavan'ny masoandro. Fandrindrana tsotra sy geometrika no manasongadina ny tena ilaina sy manamafy ny toeran'ny marika.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/jeff-sheldon-9SyOKYrq-rE-unsplash.jpg"},
				{Type: "image", Src: "/images/Portfolio.jpg"},
				{Type: "image", Src: "/images/Contact Background.jpg"},
			},
		},
		{
			ProjectID:  "photo-4",
			Category:   "photo",
			MediaType:  "image",
			Src:        "/images/A propos.jpg",
			BgColor:    "#2C4A3E",
			// FR
			TitleFRPart1:  "Nature",
			TitleFRPart2:  "Sauvage",
			DescriptionFR: "Reportage de photographie d'aventure en extérieur. Capturer la force de l'instant, le mouvement de l'eau et la rudesse des falaises pour traduire l'esprit d'exploration et de liberté.",
			// EN
			TitleENPart1:  "Wild",
			TitleENPart2:  "Nature",
			DescriptionEN: "Outdoor adventure photography reportage. Capturing the strength of the moment, the flow of water, and the ruggedness of cliffs to translate the spirit of exploration and freedom.",
			// MG
			TitleMGPart1:  "Natiora",
			TitleMGPart2:  "Dia",
			DescriptionMG: "Tatitra sary momba ny fizahan-tany sy ny traikefa any ivelany. Misambotra ny herin'ny fotoana, ny fikorianan'ny rano ary ny haran-dranomasina mba handikana ny sary momba ny fikarohana sy ny fahafahana.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/A propos.jpg"},
				{Type: "image", Src: "/images/Web Background.jpg"},
				{Type: "image", Src: "/images/Offre.jpg"},
			},
		},
		// VIDEO
		{
			ProjectID:  "video-1",
			Category:   "video",
			MediaType:  "video",
			Src:        "/images/Background vidéo.mp4",
			BgColor:    "#AC341E",
			// FR
			TitleFRPart1:  "Publicité",
			TitleFRPart2:  "Raffineries",
			DescriptionFR: "Spot publicitaire cinématique capturant la grandeur industrielle sous un angle épique. Montage rythmé, étalonnage chaud et sound design immersif pour marquer les esprits.",
			// EN
			TitleENPart1:  "Refineries",
			TitleENPart2:  "Commercial",
			DescriptionEN: "Cinematic commercial capturing industrial grandeur from an epic perspective. Rhythm editing, warm color grading, and immersive sound design to leave a lasting impression.",
			// MG
			TitleMGPart1:  "Rindran-tsary",
			TitleMGPart2:  "Raffineries",
			DescriptionMG: "Rindran-tsary fampiroboroboana mampiseho ny fahalehibiazan'ny indostria amin'ny fomba miavaka. Fanitsiana mafana sy feo mahasarika mba hametrahana marika eo amin'ny saina.",
			Gallery: models.Gallery{
				{Type: "video", Src: "/images/Background vidéo.mp4"},
				{Type: "image", Src: "/images/windows-w79mIrYKcK4-unsplash.jpg"},
				{Type: "video", Src: "/images/Background vidéo.mp4"},
			},
		},
		{
			ProjectID:  "video-2",
			Category:   "video",
			MediaType:  "image",
			Src:        "/images/windows-w79mIrYKcK4-unsplash.jpg",
			BgColor:    "#243A4F",
			// FR
			TitleFRPart1:  "Cinéma",
			TitleFRPart2:  "Urbain",
			DescriptionFR: "Recherches esthétiques nocturnes explorant les reflets de néons, l'asphalte humide et les contrastes colorimétriques de la vie citadine tardive.",
			// EN
			TitleENPart1:  "Urban",
			TitleENPart2:  "Cinema",
			DescriptionEN: "Nighttime aesthetic research exploring neon reflections, wet asphalt, and the color contrasts of late-night city life.",
			// MG
			TitleMGPart1:  "Sinema",
			TitleMGPart2:  "An-Tanàn-Dehibe",
			DescriptionMG: "Fikarohana momba ny hatsaran'ny alina, ny taratry ny jiro neon, ny arabe mando ary ny fifanoheran'ny loko amin'ny fiainana alina.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/windows-w79mIrYKcK4-unsplash.jpg"},
				{Type: "video", Src: "/images/Background vidéo.mp4"},
				{Type: "image", Src: "/images/Photographie Background.jpg"},
			},
		},
		{
			ProjectID:  "video-3",
			Category:   "video",
			MediaType:  "video",
			Src:        "/images/Background vidéo.mp4",
			BgColor:    "#3E163F",
			// FR
			TitleFRPart1:  "Court",
			TitleFRPart2:  "Métrage",
			DescriptionFR: "Œuvre de fiction courte axée sur l'émotion visuelle. Les ombres et la gestion fine du clair-obscur guident le spectateur à travers un voyage mental poétique.",
			// EN
			TitleENPart1:  "Short",
			TitleENPart2:  "Film",
			DescriptionEN: "Short fiction work focused on visual emotion. Shadows and fine chiaroscuro guide the viewer through a poetic mental journey.",
			// MG
			TitleMGPart1:  "Horonantsary",
			TitleMGPart2:  "Fohy",
			DescriptionMG: "Tetikasa tantara fohy miompana amin'ny fihetseham-po ara-tsary. Ny alokaloka dia mitarika ny mpijery amin'ny dia poety sy ara-tsaina.",
			Gallery: models.Gallery{
				{Type: "video", Src: "/images/Background vidéo.mp4"},
				{Type: "image", Src: "/images/A propos.jpg"},
				{Type: "video", Src: "/images/Background vidéo.mp4"},
			},
		},
		// GRAPHIQUE
		{
			ProjectID:  "graphique-1",
			Category:   "graphique",
			MediaType:  "image",
			Src:        "/images/jeff-sheldon-9SyOKYrq-rE-unsplash.jpg",
			BgColor:    "#1A1A1A",
			// FR
			TitleFRPart1:  "Identité",
			TitleFRPart2:  "Epik",
			DescriptionFR: "Refonte globale de la charte d'Epik Brand. De la conception du logo minimaliste aux typographies géométriques pour exprimer une sophistication technologique.",
			// EN
			TitleENPart1:  "Epik",
			TitleENPart2:  "Identity",
			DescriptionEN: "Global redesign of Epik Brand's visual identity. From minimalist logo conception to geometric typography to express technological sophistication.",
			// MG
			TitleMGPart1:  "Maha-izy",
			TitleMGPart2:  "Azy Epik",
			DescriptionMG: "Fanavaozana manontolo ny tabilao famantarana ny Epik Brand. Manomboka amin'ny famoronana logo tsotra ka hatramin'ny soratra geometrika mba hanehoana fahaiza-manao ara-teknolojia.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/jeff-sheldon-9SyOKYrq-rE-unsplash.jpg"},
				{Type: "image", Src: "/images/A propos.jpg"},
				{Type: "image", Src: "/images/Portfolio.jpg"},
			},
		},
		{
			ProjectID:  "graphique-2",
			Category:   "graphique",
			MediaType:  "image",
			Src:        "/images/A propos.jpg",
			BgColor:    "#6B7A82",
			// FR
			TitleFRPart1:  "MADE IN",
			TitleFRPart2:  "PAP",
			DescriptionFR: "Création d'identité visuelle et étiquette haut de gamme pour 'Made in PAP', une marque artisanale d'exception arborant fièrement l'emblème du crocodile.",
			// EN
			TitleENPart1:  "MADE IN",
			TitleENPart2:  "PAP",
			DescriptionEN: "Visual identity and high-end label design for 'Made in PAP', an exceptional artisanal brand proudly displaying the crocodile emblem.",
			// MG
			TitleMGPart1:  "MADE IN",
			TitleMGPart2:  "PAP",
			DescriptionMG: "Famoronana famantarana ara-tsary sy marika avo lenta ho an'ny 'Made in PAP', marika miavaka mampiseho am-pireharehana ny sarin'ny voay.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/A propos.jpg"},
				{Type: "image", Src: "/images/Contact Background.jpg"},
				{Type: "image", Src: "/images/Photographie Background.jpg"},
			},
		},
		{
			ProjectID:  "graphique-3",
			Category:   "graphique",
			MediaType:  "image",
			Src:        "/images/Portfolio.jpg",
			BgColor:    "#CE782F",
			// FR
			TitleFRPart1:  "Branding Le",
			TitleFRPart2:  "Zénith",
			DescriptionFR: "Charte graphique et branding pour le restaurant lounge 'Le Zénith'. Création de dessous de verre en cuir gravé et d'éléments imprimés dorés haut de gamme.",
			// EN
			TitleENPart1:  "Le Zenith",
			TitleENPart2:  "Branding",
			DescriptionEN: "Graphic chart and branding for the 'Le Zénith' lounge restaurant. Creation of engraved leather coasters and high-end gold printed elements.",
			// MG
			TitleMGPart1:  "Marika Le",
			TitleMGPart2:  "Zénith",
			DescriptionMG: "Tabilao famantarana sy marika ho an'ny trano fisakafoanana lounge 'Le Zénith'. Famoronana singa hoditra voasokitra tsara sy singa vita pirinty volamena avo lenta.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/Portfolio.jpg"},
				{Type: "image", Src: "/images/Offre.jpg"},
				{Type: "image", Src: "/images/Web Background.jpg"},
			},
		},
		// WEB
		{
			ProjectID:  "web-1",
			Category:   "web",
			MediaType:  "image",
			Src:        "/images/Web Background.jpg",
			BgColor:    "#0F1E36",
			// FR
			TitleFRPart1:  "Code",
			TitleFRPart2:  "Interactif",
			DescriptionFR: "Création de sites web vitrines immersifs avec transitions de pages complexes et interactions fluides gérées par GSAP et Three.js.",
			// EN
			TitleENPart1:  "Interactive",
			TitleENPart2:  "Code",
			DescriptionEN: "Creation of immersive showcase websites with complex page transitions and smooth interactions managed by GSAP and Three.js.",
			// MG
			TitleMGPart1:  "Kaody",
			TitleMGPart2:  "Mifandray",
			DescriptionMG: "Famoronana tranonkala vitrina mahavariana miaraka amin'ny fiovana pejy sarotra sy fifandraisana malefaka tantanin'ny GSAP sy Three.js.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/Web Background.jpg"},
				{Type: "image", Src: "/images/Contact Background.jpg"},
				{Type: "image", Src: "/images/Offre.jpg"},
			},
		},
		{
			ProjectID:  "web-2",
			Category:   "web",
			MediaType:  "image",
			Src:        "/images/Contact Background.jpg",
			BgColor:    "#1E6B5F",
			// FR
			TitleFRPart1:  "E-Commerce",
			TitleFRPart2:  "Premium",
			DescriptionFR: "Plateforme de vente en ligne sur-mesure pour des marques de luxe. Un design ultra-épuré combiné à un parcours d'achat optimisé sans aucune friction.",
			// EN
			TitleENPart1:  "Premium",
			TitleENPart2:  "E-Commerce",
			DescriptionEN: "Tailor-made online sales platform for luxury brands. An ultra-clean design combined with a frictionless optimized user path.",
			// MG
			TitleMGPart1:  "E-Commerce",
			TitleMGPart2:  "Premium",
			DescriptionMG: "Tranonkala fivarotana an-tambajotra natao manokana ho an'ny marika lafo vidy. Endrika madio sy fizotran'ny fividianana voalamina tsara.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/Contact Background.jpg"},
				{Type: "image", Src: "/images/Web Background.jpg"},
				{Type: "image", Src: "/images/Portfolio.jpg"},
			},
		},
		{
			ProjectID:  "web-3",
			Category:   "web",
			MediaType:  "image",
			Src:        "/images/Offre.jpg",
			BgColor:    "#225B82",
			// FR
			TitleFRPart1:  "Platform",
			TitleFRPart2:  "SaaS",
			DescriptionFR: "Dashboard moderne et ergonomique pour des services d'analyse complexes, alliant lisibilité des données, graphiques fluides et thèmes personnalisables.",
			// EN
			TitleENPart1:  "SaaS",
			TitleENPart2:  "Platform",
			DescriptionEN: "Modern and ergonomic dashboard for complex analysis services, combining readable data, smooth graphs, and customizable themes.",
			// MG
			TitleMGPart1:  "SaaS",
			TitleMGPart2:  "Platform",
			DescriptionMG: "Dashboard maoderina sy ergonomika ho an'ny asa fikarohana sarotra, manambatra ny fahafahana mamaky angona, sary mihetsika ary lohahevitra azo ovaina.",
			Gallery: models.Gallery{
				{Type: "image", Src: "/images/Offre.jpg"},
				{Type: "image", Src: "/images/jeff-sheldon-9SyOKYrq-rE-unsplash.jpg"},
				{Type: "image", Src: "/images/sam-mcghee-KieCLNzKoBo-unsplash.jpg"},
			},
		},
	}

	for _, p := range defaultProjects {
		if err := InsertProject(db, &p); err != nil {
			return fmt.Errorf("impossible de peupler le projet %s: %w", p.ProjectID, err)
		}
	}

	log.Printf("[DB] %d projets par défaut (avec traductions FR/EN/MG) ont été insérés.", len(defaultProjects))
	return nil
}

// InsertTeamMember insère ou met à jour un membre de l'équipe
func InsertTeamMember(db *sql.DB, m *models.TeamMember) error {
	// Si aucun member_id n'est spécifié, on l'auto-génère
	if m.MemberID == "" {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("erreur lors du démarrage de la transaction: %w", err)
		}
		defer tx.Rollback()

		var id int64
		tempID := fmt.Sprintf("temp-%d", time.Now().UnixNano())
		query := `
		INSERT INTO team_members (
			member_id, prenom, nom, email, photo,
			role_fr, role_en, role_mg
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id;
		`
		err = tx.QueryRow(
			query,
			tempID, m.Prenom, m.Nom, m.Email, m.Photo,
			m.RoleFR, m.RoleEN, m.RoleMG,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("erreur lors de l'insertion initiale du membre: %w", err)
		}

		m.MemberID = fmt.Sprintf("member-%d", id)
		updateQuery := `UPDATE team_members SET member_id = $1 WHERE id = $2;`
		_, err = tx.Exec(updateQuery, m.MemberID, id)
		if err != nil {
			return fmt.Errorf("erreur lors de la mise à jour de l'identifiant du membre: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("erreur lors de la validation de la transaction: %w", err)
		}
		return nil
	}

	query := `
	INSERT INTO team_members (
		member_id, prenom, nom, email, photo,
		role_fr, role_en, role_mg
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (member_id)
	DO UPDATE SET
		prenom = EXCLUDED.prenom,
		nom = EXCLUDED.nom,
		email = EXCLUDED.email,
		photo = EXCLUDED.photo,
		role_fr = EXCLUDED.role_fr,
		role_en = EXCLUDED.role_en,
		role_mg = EXCLUDED.role_mg;
	`
	_, err := db.Exec(
		query,
		m.MemberID, m.Prenom, m.Nom, m.Email, m.Photo,
		m.RoleFR, m.RoleEN, m.RoleMG,
	)
	return err
}

// GetAllTeamMembers récupère tous les membres de l'équipe
func GetAllTeamMembers(db *sql.DB) ([]models.TeamMember, error) {
	query := `
	SELECT id, member_id, prenom, nom, email, photo,
	       role_fr, role_en, role_mg
	FROM team_members
	ORDER BY id ASC;
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var m models.TeamMember
		err := rows.Scan(
			&m.ID, &m.MemberID, &m.Prenom, &m.Nom, &m.Email, &m.Photo,
			&m.RoleFR, &m.RoleEN, &m.RoleMG,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// DeleteTeamMember supprime un membre de l'équipe
func DeleteTeamMember(db *sql.DB, memberID string) error {
	query := `DELETE FROM team_members WHERE member_id = $1;`
	res, err := db.Exec(query, memberID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("aucun membre trouvé avec l'id '%s'", memberID)
	}
	return nil
}

// seedDefaultTeamData remplit la base avec les membres de l'équipe par défaut s'il n'y en a pas
func seedDefaultTeamData(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM team_members").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	defaultMembers := []models.TeamMember{
		{
			MemberID: "member-1",
			Prenom:   "Nantenaina",
			Nom:      "RANDRIA",
			Email:    "nantenaina@gmail.com",
			Photo:    "/images/Nantenaina.png",
			RoleFR:   "CEO / Directeur Artistique",
			RoleEN:   "CEO / Art Director",
			RoleMG:   "CEO / Mpitarika ara-kanto",
		},
		{
			MemberID: "member-2",
			Prenom:   "Miora Oliva",
			Nom:      "RAHOLIARIVAO",
			Email:    "miora@gmail.com",
			Photo:    "/images/Miora.png",
			RoleFR:   "Project Manager",
			RoleEN:   "Project Manager",
			RoleMG:   "Project Manager",
		},
	}

	for _, m := range defaultMembers {
		if err := InsertTeamMember(db, &m); err != nil {
			return fmt.Errorf("impossible de peupler le membre %s: %w", m.Prenom, err)
		}
	}

	log.Printf("[DB] %d membres de l'équipe par défaut ont été insérés.", len(defaultMembers))
	return nil
}

// GetUserByEmail récupère un utilisateur par son email
func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	var u models.User
	query := `SELECT id, email, name, role, COALESCE(picture_url, ''), created_at FROM users WHERE email = $1`
	err := db.QueryRow(query, email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PictureURL, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserProfile met à jour le nom et la photo d'un utilisateur (appelé lors de sa connexion via Google)
func UpdateUserProfile(db *sql.DB, email, name, pictureURL string) error {
	query := `UPDATE users SET name = $1, picture_url = $2 WHERE email = $3`
	_, err := db.Exec(query, name, pictureURL, email)
	return err
}

// CreateSession enregistre un jeton de session pour un utilisateur
func CreateSession(db *sql.DB, token string, userID int, duration time.Duration) error {
	expiresAt := time.Now().Add(duration)
	query := `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, token, userID, expiresAt)
	return err
}

// GetSessionUser récupère l'utilisateur associé à un jeton de session s'il n'est pas expiré
func GetSessionUser(db *sql.DB, token string) (*models.User, error) {
	var u models.User
	query := `
		SELECT u.id, u.email, u.name, u.role, COALESCE(u.picture_url, ''), u.created_at 
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		WHERE s.token = $1 AND s.expires_at > $2
	`
	err := db.QueryRow(query, token, time.Now()).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PictureURL, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// DeleteSession supprime un jeton de session
func DeleteSession(db *sql.DB, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := db.Exec(query, token)
	return err
}

// GetAllUsers récupère la liste de tous les modérateurs et de l'administrateur
func GetAllUsers(db *sql.DB) ([]models.User, error) {
	query := `SELECT id, email, name, role, COALESCE(picture_url, ''), created_at FROM users ORDER BY role ASC, id ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PictureURL, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// InsertUser ajoute un nouveau modérateur
func InsertUser(db *sql.DB, email, name, role string) error {
	query := `
		INSERT INTO users (email, name, role, picture_url) 
		VALUES ($1, $2, $3, '')
		ON CONFLICT (email) DO UPDATE SET role = EXCLUDED.role, name = EXCLUDED.name
	`
	_, err := db.Exec(query, email, name, role)
	return err
}

// DeleteUser supprime un modérateur par son ID
func DeleteUser(db *sql.DB, id int) error {
	query := `DELETE FROM users WHERE id = $1 AND role != 'admin'`
	res, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("aucun modérateur trouvé ou impossible de supprimer l'administrateur principal")
	}
	return nil
}

// TransferAdminRole transfère le rôle d'administrateur à un modérateur désigné
func TransferAdminRole(db *sql.DB, currentAdminID, newAdminID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Rétrograder l'admin actuel en modérateur
	_, err = tx.Exec(`UPDATE users SET role = 'moderator' WHERE id = $1 AND role = 'admin'`, currentAdminID)
	if err != nil {
		return fmt.Errorf("erreur lors de la rétrogradation de l'administrateur : %w", err)
	}

	// 2. Promouvoir le nouveau modérateur en administrateur
	res, err := tx.Exec(`UPDATE users SET role = 'admin' WHERE id = $1 AND role = 'moderator'`, newAdminID)
	if err != nil {
		return fmt.Errorf("erreur lors de la promotion du nouvel administrateur : %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("le nouvel administrateur désigné n'a pas été trouvé ou n'est pas modérateur")
	}

	// 3. Supprimer toutes les sessions de l'ancien admin et du nouvel admin pour forcer la reconnexion avec les nouveaux rôles
	_, err = tx.Exec(`DELETE FROM sessions WHERE user_id = $1 OR user_id = $2`, currentAdminID, newAdminID)
	if err != nil {
		return fmt.Errorf("erreur lors du nettoyage des sessions : %w", err)
	}

	return tx.Commit()
}

// GetSetting récupère la valeur d'un paramètre par sa clé
func GetSetting(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = $1", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// UpdateSetting met à jour ou insère un paramètre
func UpdateSetting(db *sql.DB, key, value string) error {
	query := `
		INSERT INTO settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`
	_, err := db.Exec(query, key, value)
	return err
}

