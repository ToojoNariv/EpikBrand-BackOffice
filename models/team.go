package models

import (
	"strings"
)

// TeamMember représente un membre de l'équipe Epik Brand avec support multilingue
type TeamMember struct {
	ID        int64  `json:"db_id"` // Identifiant interne
	MemberID  string `json:"id"`    // Identifiant unique côté client (ex: 'member-1')
	Prenom    string `json:"prenom"`
	Nom       string `json:"nom"`
	Email     string `json:"email"`
	Photo     string `json:"photo"` // URL de la photo (ImageKit, Cloudinary, etc.)

	// Champ dynamique renvoyé au client selon la langue demandée
	Role      string `json:"role"`

	// Champs de traduction pour le rôle
	RoleFR    string `json:"role_fr"`
	RoleEN    string `json:"role_en"`
	RoleMG    string `json:"role_mg"`
}

// Localize peuple le champ 'Role' dynamiquement selon la langue cible
func (m *TeamMember) Localize(lang string) {
	switch strings.ToLower(lang) {
	case "en":
		m.Role = m.RoleEN
		if m.Role == "" {
			m.Role = m.RoleFR
		}
	case "mg":
		m.Role = m.RoleMG
		if m.Role == "" {
			m.Role = m.RoleFR
		}
	default: // "fr"
		m.Role = m.RoleFR
	}
}
